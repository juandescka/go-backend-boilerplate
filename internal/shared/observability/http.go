// Package observability contains HTTP-scoped logging and correlation helpers.
package observability

import (
	"context"
	"net"
	"net/http"
	"reflect"
	"runtime/debug"
	"strings"
	"time"

	appmetrics "hris-core-service/internal/shared/metrics"
	"hris-core-service/internal/shared/response"
	"hris-core-service/internal/shared/sanitization"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

const (
	RequestIDHeader     = "X-Request-ID"
	RequestIDContextKey = "request_id"

	maxLoggedPathLength = 2048
)

type requestIDContextKey struct{}

// RequestIDFromStandardContext returns the validated request ID propagated by
// RequestIDMiddleware without exposing the private context key.
func RequestIDFromStandardContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	requestID, _ := ctx.Value(requestIDContextKey{}).(string)
	return requestID
}

// RequestIDMiddleware validates or creates the request correlation ID.
func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := ValidRequestID(c.GetHeader(RequestIDHeader))
		if requestID == "" {
			requestID = uuid.NewString()
		}

		c.Set(RequestIDContextKey, requestID)
		c.Header(RequestIDHeader, requestID)
		c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), requestIDContextKey{}, requestID))
		c.Next()
	}
}

// RequestIDFromContext returns the request ID stored by RequestIDMiddleware.
func RequestIDFromContext(c *gin.Context) string {
	if c == nil {
		return ""
	}
	value, exists := c.Get(RequestIDContextKey)
	if !exists {
		return ""
	}
	requestID, ok := value.(string)
	if !ok {
		return ""
	}
	return requestID
}

// ValidRequestID returns a safe incoming ID, or an empty string when it must
// be replaced with a generated UUID.
func ValidRequestID(value string) string {
	value = strings.TrimSpace(value)
	if len(value) == 0 || len(value) > 128 {
		return ""
	}
	for _, character := range value {
		if character > 127 || !(character >= 'a' && character <= 'z' || character >= 'A' && character <= 'Z' || character >= '0' && character <= '9' || character == '.' || character == '_' || character == ':' || character == '-') {
			return ""
		}
	}
	return value
}

// RequestLoggerEntry attaches the current correlation ID to a log entry.
func RequestLoggerEntry(log *logrus.Logger, c *gin.Context) *logrus.Entry {
	if log == nil {
		return nil
	}
	entry := log.WithField("request_id", RequestIDFromContext(c))
	if c != nil && c.Request != nil {
		return entry.WithContext(c.Request.Context())
	}
	return entry
}

// RequestCompletionLogger emits exactly one event after the later middleware
// and handler have finished.
func RequestCompletionLogger(log *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		startedAt := time.Now()
		c.Next()

		if log == nil {
			return
		}
		status := c.Writer.Status()
		responseSize := c.Writer.Size()
		if responseSize < 0 {
			responseSize = 0
		}
		route := c.FullPath()
		path := SanitizePath(c.Request.URL.Path)
		if route == "" {
			route = path
		}

		fields := logrus.Fields{
			"event_code":          "http.request.completed",
			"event_category":      "http",
			"request_id":          RequestIDFromContext(c),
			"http_method":         c.Request.Method,
			"http_path":           path,
			"http_route":          SanitizePath(route),
			"http_status":         status,
			"latency_ms":          float64(time.Since(startedAt).Microseconds()) / 1000,
			"response_size_bytes": responseSize,
			"client_ip":           CanonicalClientIP(c),
			"user_agent":          SanitizeUserAgent(c.Request.UserAgent()),
		}
		addPrincipalFields(fields, c)
		errorCode := response.ErrorCodeFromContext(c)
		if errorCode != "" {
			fields["error_code"] = errorCode
		}

		entry := log.WithFields(fields).WithContext(c.Request.Context())
		switch {
		case status >= http.StatusInternalServerError:
			entry.Error("HTTP request failed")
		case status == http.StatusUnauthorized && errorCode == "UNAUTHORIZED":
			// A classified authentication denial is routine, not an internal failure.
			entry.Info("HTTP request rejected")
		case status >= http.StatusBadRequest:
			entry.Warn("HTTP request rejected")
		default:
			entry.Info("HTTP request completed")
		}
	}
}

// PanicRecovery logs panics internally and writes the standard error envelope
// only while the response remains writable.
func PanicRecovery(log *logrus.Logger, metricRecorders ...appmetrics.HTTPRecorder) gin.HandlerFunc {
	metricRecorder := appmetrics.HTTPRecorder(appmetrics.NoopRecorder{})
	if len(metricRecorders) > 0 && metricRecorders[0] != nil {
		metricRecorder = metricRecorders[0]
	}
	return func(c *gin.Context) {
		defer func() {
			if recovered := recover(); recovered != nil {
				metricRecorder.RecordPanic()
				if log != nil {
					fields := logrus.Fields{
						"event_code":     "http.request.panic",
						"event_category": "http",
						"request_id":     RequestIDFromContext(c),
						"http_method":    c.Request.Method,
						"http_path":      SanitizePath(c.Request.URL.Path),
						"http_route":     SanitizePath(c.FullPath()),
						"client_ip":      CanonicalClientIP(c),
						"user_agent":     SanitizeUserAgent(c.Request.UserAgent()),
						"panic_type":     panicType(recovered),
						"stack_trace":    string(debug.Stack()),
					}
					log.WithFields(fields).WithContext(c.Request.Context()).Error("HTTP request panicked")
				}
				if !c.Writer.Written() {
					response.ErrorResponse(c, http.StatusInternalServerError, "Internal Server Error", nil)
				}
				c.Abort()
			}
		}()
		c.Next()
	}
}

// NewRequestMetadata converts safe HTTP values to security-event metadata.
func NewRequestMetadata(c *gin.Context) (requestID, clientIP, userAgent, method, path string) {
	if c == nil || c.Request == nil {
		return "", "", "", "", ""
	}
	return RequestIDFromContext(c), CanonicalClientIP(c), SanitizeUserAgent(c.Request.UserAgent()), c.Request.Method, SanitizePath(c.Request.URL.Path)
}

// SanitizeUserAgent prevents log injection and bounds the untrusted header.
func SanitizeUserAgent(value string) string {
	return sanitization.UserAgent(value)
}

// SanitizePath prepares a request path for logs without changing routing.
func SanitizePath(value string) string {
	value = strings.NewReplacer("\r", " ", "\n", " ").Replace(value)
	if len([]rune(value)) <= maxLoggedPathLength {
		return value
	}
	return string([]rune(value)[:maxLoggedPathLength])
}

// CanonicalClientIP uses Gin's configured trusted-proxy resolution.
func CanonicalClientIP(c *gin.Context) string {
	if c == nil {
		return ""
	}
	ip := net.ParseIP(strings.TrimSpace(c.ClientIP()))
	if ip == nil {
		return ""
	}
	return ip.String()
}

func addPrincipalFields(fields logrus.Fields, c *gin.Context) {
	for _, key := range []string{"user_id", "user_role_id", "role_id", "organization_id"} {
		if value, exists := c.Get(key); exists {
			if stringValue, ok := value.(string); ok && stringValue != "" {
				fields[key] = stringValue
			}
		}
	}
}

func panicType(value interface{}) string {
	if value == nil {
		return "nil"
	}
	return reflect.TypeOf(value).String()
}
