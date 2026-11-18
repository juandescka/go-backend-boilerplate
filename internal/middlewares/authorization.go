package middlewares

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"hris-core-service/internal/shared/response"
	auth "hris-core-service/internal/usecases/authentication"
	access "hris-core-service/internal/usecases/authorization"
)

// RequirePermission runs after authentication. Resolve the resource's stored
// owners before returning its scope; requested tenant IDs alone prove no access.
func RequirePermission(service *access.Service, permission string, resolve func(*gin.Context) (access.Scope, error)) gin.HandlerFunc {
	return func(c *gin.Context) {
		var scope access.Scope
		var err error
		if _, ok := auth.IdentityFromContext(c.Request.Context()); !ok {
			err = auth.ErrUnauthenticated
		} else if resolve == nil {
			err = access.ErrUnavailable
		} else {
			scope, err = resolve(c)
			if err == nil {
				var authorizedContext context.Context
				authorizedContext, err = service.Authorize(c.Request.Context(), permission, scope)
				if err == nil {
					c.Request = c.Request.WithContext(authorizedContext)
				}
			}
		}
		if err != nil {
			status, code, message := http.StatusForbidden, "FORBIDDEN", "Access is denied."
			if errors.Is(err, auth.ErrUnauthenticated) {
				status, code, message = http.StatusUnauthorized, "UNAUTHORIZED", "Authentication is required."
				c.Header("WWW-Authenticate", "Bearer")
			} else if !errors.Is(err, access.ErrForbidden) {
				status, code, message = http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "Authorization could not be completed."
			}
			response.ErrorResponseWithCode(c, status, code, message, nil)
			c.Abort()
			return
		}
		c.Next()
	}
}
