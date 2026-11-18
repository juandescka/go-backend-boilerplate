package middlewares

import (
	"errors"
	"net/http"
	"strings"

	"hris-core-service/internal/shared/observability"
	"hris-core-service/internal/shared/response"
	securityevent "hris-core-service/internal/shared/security"
	auth "hris-core-service/internal/usecases/authentication"

	"github.com/gin-gonic/gin"
)

func BearerAuthentication(service *auth.Service, events securityevent.Recorder) gin.HandlerFunc {
	if events == nil {
		events = securityevent.NoopRecorder{}
	}
	return func(c *gin.Context) {
		c.Header("Cache-Control", "no-store")
		headers := c.Request.Header.Values("Authorization")
		var token string
		// Reject ambiguity instead of choosing credentials a proxy may parse differently.
		if len(headers) == 1 && len(headers[0]) == 50 && strings.EqualFold(headers[0][:7], "Bearer ") {
			token = headers[0][7:]
		}
		identity, err := service.Authenticate(c.Request.Context(), token)
		if err != nil {
			status, code, message, outcome, reason := http.StatusUnauthorized, "UNAUTHORIZED", "Authentication is required.", securityevent.OutcomeDenied, "invalid_session"
			if !errors.Is(err, auth.ErrUnauthenticated) {
				status, code, message, outcome, reason = 500, "INTERNAL_SERVER_ERROR", "Authentication could not be completed.", securityevent.OutcomeBlocked, "operation_failed"
			}
			if status == 401 {
				c.Header("WWW-Authenticate", "Bearer")
			}
			requestID, ip, agent, method, path := observability.NewRequestMetadata(c)
			events.Record(c.Request.Context(), securityevent.Event{Code: securityevent.CodeContextInvalid, Outcome: outcome, ReasonCode: reason, Request: securityevent.NewRequestMetadataWithRequestID(requestID, ip, agent, method, path)})
			response.ErrorResponseWithCode(c, status, code, message, nil)
			c.Abort()
			return
		}
		c.Request = c.Request.WithContext(auth.WithIdentity(c.Request.Context(), identity))
		c.Set("user_id", identity.UserID.String()) // Existing safe completion-log field.
		c.Next()
	}
}
