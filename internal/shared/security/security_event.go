// Package securityevent defines the internal, structured records emitted for
// authentication security decisions. These records are operational telemetry,
// not API responses or audit-log records.
package securityevent

import (
	"context"
	"net"
	"strings"

	"hris-core-service/internal/shared/sanitization"

	"github.com/sirupsen/logrus"
)

const (
	CategoryAuthentication = "authentication"

	CodeLoginSucceeded   = "auth.login.succeeded"
	CodeLoginFailed      = "auth.login.failed"
	CodeLoginRateLimited = "auth.login.rate_limited"
	CodeRefreshInvalid   = "auth.refresh.invalid"
	CodeRefreshRateLimit = "auth.refresh.rate_limited"
	CodeContextInvalid   = "auth.context.invalid"
	CodeLogoutAllSuccess = "auth.logout_all.succeeded"
	CodeLogoutSucceeded  = "auth.logout.succeeded"
	CodeLogoutFailed     = "auth.logout.failed"

	OutcomeSuccess = "success"
	OutcomeDenied  = "denied"
	OutcomeBlocked = "blocked"

	ReasonUserNotFound                 = "user_not_found"
	ReasonUserInactive                 = "user_inactive"
	ReasonPasswordMismatch             = "password_mismatch"
	ReasonIPLimitExceeded              = "ip_limit_exceeded"
	ReasonIdentifierLimitExceeded      = "identifier_limit_exceeded"
	ReasonTokenNotFound                = "token_not_found"
	ReasonTokenExpired                 = "token_expired"
	ReasonAuthenticationContextInvalid = "authentication_context_invalid"
)

// RequestMetadata is the framework-independent request context permitted in a
// security event. It intentionally contains no headers, bodies, or tokens.
type RequestMetadata struct {
	RequestID string
	ClientIP  string
	UserAgent string
	Method    string
	Path      string
}

// Event is one internal authentication security event. Empty optional fields
// are omitted from the structured log output.
type Event struct {
	Code              string
	Outcome           string
	ReasonCode        string
	Request           RequestMetadata
	IdentifierHash    string
	UserID            string
	UserRoleID        string
	RoleID            string
	OrganizationID    *string
	RetryAfterSeconds *int64
}

// Recorder records authentication security events without changing the
// authentication operation that caused them.
type Recorder interface {
	Record(ctx context.Context, event Event)
}

// NoopRecorder is useful for callers and tests that do not need event output.
type NoopRecorder struct{}

func (NoopRecorder) Record(context.Context, Event) {}

// LogrusRecorder emits events with the application's existing JSON logrus
// logger. Logrus hooks may fail internally, but Record intentionally has no
// error result so event delivery cannot affect authentication behavior.
type LogrusRecorder struct {
	logger *logrus.Logger
}

func NewLogrusRecorder(logger *logrus.Logger) Recorder {
	if logger == nil {
		return NoopRecorder{}
	}
	return &LogrusRecorder{logger: logger}
}

func (r *LogrusRecorder) Record(ctx context.Context, event Event) {
	if r == nil || r.logger == nil {
		return
	}

	fields := logrus.Fields{
		"event_code":     event.Code,
		"event_category": CategoryAuthentication,
		"outcome":        event.Outcome,
	}
	addString(fields, "reason_code", event.ReasonCode)
	addString(fields, "request_id", event.Request.RequestID)
	addString(fields, "client_ip", event.Request.ClientIP)
	addString(fields, "user_agent", event.Request.UserAgent)
	addString(fields, "http_method", event.Request.Method)
	addString(fields, "http_path", event.Request.Path)
	addString(fields, "identifier_hash", event.IdentifierHash)
	addString(fields, "user_id", event.UserID)
	addString(fields, "user_role_id", event.UserRoleID)
	addString(fields, "role_id", event.RoleID)
	if event.OrganizationID != nil && *event.OrganizationID != "" {
		fields["organization_id"] = *event.OrganizationID
	}
	if event.RetryAfterSeconds != nil {
		fields["retry_after_seconds"] = *event.RetryAfterSeconds
	}

	entry := r.logger.WithFields(fields)
	if ctx != nil {
		entry = entry.WithContext(ctx)
	}
	if event.Code == CodeLoginSucceeded || event.Code == CodeLogoutAllSuccess || event.Code == CodeLogoutSucceeded {
		entry.Info(eventMessage(event.Code))
		return
	}
	if event.Code == CodeLoginFailed || event.Code == CodeContextInvalid || event.Code == CodeLogoutFailed {
		switch {
		case event.Outcome == OutcomeDenied:
			entry.Info(eventMessage(event.Code))
		case event.Outcome == OutcomeBlocked && event.ReasonCode == "operation_failed":
			entry.Error(eventMessage(event.Code))
		default:
			entry.Warn(eventMessage(event.Code))
		}
		return
	}
	entry.Warn(eventMessage(event.Code))
}

func eventMessage(code string) string {
	switch code {
	case CodeLoginSucceeded:
		return "Authentication succeeded"
	case CodeLoginFailed:
		return "Authentication failed"
	case CodeLoginRateLimited:
		return "Login attempt limited"
	case CodeContextInvalid:
		return "Session authentication failed"
	case CodeLogoutSucceeded:
		return "Session logged out"
	case CodeLogoutAllSuccess:
		return "User sessions logged out"
	case CodeLogoutFailed:
		return "Session logout failed"
	default:
		return code
	}
}

// NewRequestMetadata prepares safe request metadata at the HTTP boundary.
func NewRequestMetadata(clientIP, userAgent, method, path string) RequestMetadata {
	return NewRequestMetadataWithRequestID("", clientIP, userAgent, method, path)
}

// NewRequestMetadataWithRequestID prepares safe request metadata with an HTTP
// correlation ID that has already been validated by the request middleware.
func NewRequestMetadataWithRequestID(requestID, clientIP, userAgent, method, path string) RequestMetadata {
	return RequestMetadata{
		RequestID: requestID,
		ClientIP:  canonicalIP(clientIP),
		UserAgent: SanitizeUserAgent(userAgent),
		Method:    method,
		Path:      path,
	}
}

// SanitizeUserAgent prevents log injection and bounds an untrusted header.
func SanitizeUserAgent(value string) string {
	return sanitization.UserAgent(value)
}

func addString(fields logrus.Fields, key, value string) {
	if value != "" {
		fields[key] = value
	}
}

func canonicalIP(value string) string {
	ip := net.ParseIP(strings.TrimSpace(value))
	if ip == nil {
		return ""
	}
	return ip.String()
}
