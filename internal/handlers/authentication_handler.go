package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"hris-core-service/internal/handlers/dtos/requests"
	"hris-core-service/internal/handlers/dtos/responses"
	"hris-core-service/internal/shared/httphelper"
	"hris-core-service/internal/shared/observability"
	"hris-core-service/internal/shared/password"
	"hris-core-service/internal/shared/response"
	securityevent "hris-core-service/internal/shared/security"
	auth "hris-core-service/internal/usecases/authentication"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

var loginValidator = validator.New()

type AuthenticationHandler struct {
	service *auth.Service
	events  securityevent.Recorder
	limiter *securityevent.LoginLimiter
}

func NewAuthenticationHandler(service *auth.Service, events securityevent.Recorder, limiter *securityevent.LoginLimiter) *AuthenticationHandler {
	if events == nil {
		events = securityevent.NoopRecorder{}
	}
	return &AuthenticationHandler{service: service, events: events, limiter: limiter}
}

// Login authenticates a global user and returns an opaque token once.
// @Summary Log in with email and password
// @Description Returns a database-backed opaque bearer token. Process-local IP and normalized-identifier attempt limits return 429 with Retry-After in seconds. Requires deployed identity migrations; no registration or default account exists.
// @Tags Authentication
// @Accept json
// @Produce json
// @Param credentials body requests.LoginRequest true "Email and password"
// @Success 200 {object} response.SuccessResponseBody{data=responses.LoginResponse}
// @Failure 400 {object} response.ErrorResponseBody "Malformed JSON (BAD_REQUEST)"
// @Failure 401 {object} response.ErrorResponseBody "Invalid email or password (UNAUTHORIZED)"
// @Failure 413 {object} response.ErrorResponseBody "Body too large (REQUEST_BODY_TOO_LARGE)"
// @Failure 415 {object} response.ErrorResponseBody "Unsupported media type (UNSUPPORTED_MEDIA_TYPE)"
// @Failure 422 {object} response.ErrorResponseBody "Invalid credentials input (VALIDATION_FAILED)"
// @Failure 429 {object} response.ErrorResponseBody "Login throttled (RATE_LIMIT_EXCEEDED); Retry-After header gives whole seconds"
// @Failure 500 {object} response.ErrorResponseBody "Operational failure (INTERNAL_SERVER_ERROR)"
// @Failure 503 {object} response.ErrorResponseBody "Verification capacity exhausted (AUTHENTICATION_UNAVAILABLE)"
// @Router /v1/auth/login [post]
func (h *AuthenticationHandler) Login(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	c.Header("Pragma", "no-cache")
	ip := observability.CanonicalClientIP(c)
	if ip == "" {
		h.loginControlFailure(c)
		return
	}
	retry, limitErr := h.limiter.AllowIP(c.Request.Context(), ip)
	if h.loginLimited(c, retry, limitErr, securityevent.ReasonIPLimitExceeded) {
		return
	}
	// This endpoint's bound is smaller than the global API limit. Preserve the
	// existing global decoder policy; only login additionally rejects trailing JSON.
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 16*1024)
	decoder := json.NewDecoder(c.Request.Body)
	var request requests.LoginRequest
	err := decoder.Decode(&request)
	if err == nil {
		var trailing any
		if next := decoder.Decode(&trailing); next != io.EOF {
			err = errors.New("trailing JSON")
			if httphelper.IsRequestBodyTooLarge(next) {
				err = next
			}
		}
	}
	if err != nil {
		if httphelper.IsRequestBodyTooLarge(err) {
			response.ErrorResponseWithCode(c, 413, "REQUEST_BODY_TOO_LARGE", "Request body is too large.", nil)
		} else {
			response.ErrorResponse(c, 400, "Malformed JSON request.", nil)
		}
		return
	}
	request.Email = strings.Trim(request.Email, " ")
	if len(request.Email) > 255 || loginValidator.Struct(request) != nil || !password.ValidInput(request.Password) {
		response.ErrorResponse(c, 422, "A valid email and password are required.", nil)
		return
	}
	normalized, normalizeErr := h.service.NormalizeLoginIdentifier(c.Request.Context(), request.Email)
	if normalizeErr != nil {
		h.loginControlFailure(c)
		return
	}
	retry, limitErr = h.limiter.AllowIdentifier(c.Request.Context(), normalized)
	if h.loginLimited(c, retry, limitErr, securityevent.ReasonIdentifierLimitExceeded) {
		return
	}
	result, err := h.service.Login(c.Request.Context(), request.Email, request.Password, auth.Metadata{IPAddress: observability.CanonicalClientIP(c), UserAgent: observability.SanitizeUserAgent(c.Request.UserAgent())})
	if err != nil {
		status, code, message, outcome, reason := 500, "INTERNAL_SERVER_ERROR", "Authentication could not be completed.", securityevent.OutcomeBlocked, "operation_failed"
		if errors.Is(err, auth.ErrUnauthenticated) {
			status, code, message, outcome, reason = 401, "UNAUTHORIZED", "Invalid email or password.", securityevent.OutcomeDenied, "invalid_credentials"
		}
		if errors.Is(err, auth.ErrBusy) {
			status, code, message, reason = 503, "AUTHENTICATION_UNAVAILABLE", "Authentication is temporarily unavailable.", "verification_capacity"
		}
		h.record(c, securityevent.CodeLoginFailed, outcome, reason, "")
		response.ErrorResponseWithCode(c, status, code, message, nil)
		return
	}
	h.record(c, securityevent.CodeLoginSucceeded, securityevent.OutcomeSuccess, "", result.User.ID.String())
	response.SuccessResponse(c, responses.LoginResponse{AccessToken: result.AccessToken, TokenType: "Bearer", ExpiresAt: result.ExpiresAt, User: responses.AuthUser{ID: result.User.ID.String(), Email: result.User.Email, DisplayName: result.User.DisplayName, EmailVerifiedAt: result.User.EmailVerifiedAt}}, "Login successful.", nil, nil, nil)
}

func (h *AuthenticationHandler) loginControlFailure(c *gin.Context) {
	h.record(c, securityevent.CodeLoginFailed, securityevent.OutcomeBlocked, "operation_failed", "")
	response.ErrorResponseWithCode(c, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "Authentication could not be completed.", nil)
}

func (h *AuthenticationHandler) loginLimited(c *gin.Context, retry time.Duration, err error, reason string) bool {
	if err != nil {
		h.loginControlFailure(c)
		return true
	}
	if retry <= 0 {
		return false
	}
	seconds := int64(math.Ceil(retry.Seconds()))
	c.Header("Retry-After", strconv.FormatInt(seconds, 10))
	requestID, ip, agent, method, path := observability.NewRequestMetadata(c)
	h.events.Record(c.Request.Context(), securityevent.Event{Code: securityevent.CodeLoginRateLimited, Outcome: securityevent.OutcomeBlocked, ReasonCode: reason, RetryAfterSeconds: &seconds, Request: securityevent.NewRequestMetadataWithRequestID(requestID, ip, agent, method, path)})
	response.ErrorResponseWithCode(c, http.StatusTooManyRequests, "RATE_LIMIT_EXCEEDED", "Too many login attempts. Try again later.", nil)
	return true
}

// Me returns the safe identity resolved by the bearer middleware.
// @Summary Get the authenticated global user
// @Description Validates session expiry, revocation and user activation on every request. Authentication grants no tenant or RBAC privileges.
// @Tags Authentication
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.SuccessResponseBody{data=responses.AuthUser}
// @Failure 401 {object} response.ErrorResponseBody "Authentication required (UNAUTHORIZED)"
// @Failure 500 {object} response.ErrorResponseBody "Operational failure (INTERNAL_SERVER_ERROR)"
// @Router /v1/auth/me [get]
func (h *AuthenticationHandler) Me(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	identity, ok := auth.IdentityFromContext(c.Request.Context())
	if !ok {
		response.ErrorResponse(c, http.StatusUnauthorized, "Authentication is required.", nil)
		return
	}
	response.SuccessResponse(c, responses.AuthUser{ID: identity.UserID.String(), Email: identity.Email, DisplayName: identity.DisplayName, EmailVerifiedAt: identity.EmailVerifiedAt}, "Current user retrieved.", nil, nil, nil)
}

// Logout revokes the credential that authenticated this request.
// @Summary Log out the current session
// @Description Revokes and retains the current session. No request payload is required. The revoked bearer is rejected on subsequent requests, including repeated logout.
// @Tags Authentication
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.SuccessResponseBody "Session revoked; data is null"
// @Failure 401 {object} response.ErrorResponseBody "Invalid or no longer valid caller (UNAUTHORIZED)"
// @Failure 413 {object} response.ErrorResponseBody "Body too large (REQUEST_BODY_TOO_LARGE)"
// @Failure 415 {object} response.ErrorResponseBody "Unsupported media type (UNSUPPORTED_MEDIA_TYPE)"
// @Failure 500 {object} response.ErrorResponseBody "Operational failure (INTERNAL_SERVER_ERROR)"
// @Router /v1/auth/logout [post]
func (h *AuthenticationHandler) Logout(c *gin.Context) { h.logout(c, false) }

// LogoutAll revokes the authenticated user's active sessions.
// @Summary Log out all current user sessions
// @Description Revokes active sessions for the authenticated user, including the caller. No body identifiers control scope. Concurrent creation is serialized; a fresh password login ordered after logout-all may create a new valid session.
// @Tags Authentication
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.SuccessResponseBody "User sessions revoked; data is null"
// @Failure 401 {object} response.ErrorResponseBody "Invalid or no longer valid caller (UNAUTHORIZED)"
// @Failure 413 {object} response.ErrorResponseBody "Body too large (REQUEST_BODY_TOO_LARGE)"
// @Failure 415 {object} response.ErrorResponseBody "Unsupported media type (UNSUPPORTED_MEDIA_TYPE)"
// @Failure 500 {object} response.ErrorResponseBody "Operational failure (INTERNAL_SERVER_ERROR)"
// @Router /v1/auth/logout-all [post]
func (h *AuthenticationHandler) LogoutAll(c *gin.Context) { h.logout(c, true) }

func (h *AuthenticationHandler) logout(c *gin.Context, all bool) {
	c.Header("Cache-Control", "no-store")
	c.Header("Pragma", "no-cache")
	identity, ok := auth.IdentityFromContext(c.Request.Context())
	err := auth.ErrUnauthenticated
	if ok {
		if all {
			err = h.service.LogoutAll(c.Request.Context(), identity)
		} else {
			err = h.service.Logout(c.Request.Context(), identity)
		}
	}
	if err != nil {
		status, code, message, outcome, reason := 500, "INTERNAL_SERVER_ERROR", "Session logout could not be completed.", securityevent.OutcomeBlocked, "operation_failed"
		if errors.Is(err, auth.ErrUnauthenticated) {
			status, code, message, outcome, reason = 401, "UNAUTHORIZED", "Authentication is required.", securityevent.OutcomeDenied, "invalid_session"
			c.Header("WWW-Authenticate", "Bearer")
		}
		h.record(c, securityevent.CodeLogoutFailed, outcome, reason, "")
		response.ErrorResponseWithCode(c, status, code, message, nil)
		return
	}
	code, message := securityevent.CodeLogoutSucceeded, "Logged out."
	if all {
		code, message = securityevent.CodeLogoutAllSuccess, "All sessions logged out."
	}
	h.record(c, code, securityevent.OutcomeSuccess, "", identity.UserID.String())
	response.SuccessResponse(c, nil, message, nil, nil, nil)
}

func (h *AuthenticationHandler) record(c *gin.Context, code, outcome, reason, userID string) {
	requestID, ip, agent, method, path := observability.NewRequestMetadata(c)
	h.events.Record(c.Request.Context(), securityevent.Event{Code: code, Outcome: outcome, ReasonCode: reason, UserID: userID, Request: securityevent.NewRequestMetadataWithRequestID(requestID, ip, agent, method, path)})
}
