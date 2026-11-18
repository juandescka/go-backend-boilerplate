package middlewares

import (
	"bytes"
	"io"
	"mime"
	"net/http"
	"strings"

	"hris-core-service/config"
	"hris-core-service/internal/shared/response"

	"github.com/gin-gonic/gin"
)

const (
	requestBodyTooLargeCode  = "REQUEST_BODY_TOO_LARGE"
	unsupportedMediaTypeCode = "UNSUPPORTED_MEDIA_TYPE"
)

// SecurityHeaders applies the API's safe, transport-independent response headers.
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("Referrer-Policy", "no-referrer")
		// Set this before CORS/body rejection so even early auth errors cannot
		// become cacheable. Transport-level errors remain the server's concern.
		if c.Request.URL.Path == "/v1/auth" || strings.HasPrefix(c.Request.URL.Path, "/v1/auth/") {
			c.Header("Cache-Control", "no-store")
			c.Header("Pragma", "no-cache")
		}
		c.Next()
	}
}

// RequestBodyLimit bounds JSON API bodies before binding or route middleware.
func RequestBodyLimit(maxBytes int64, multipartPaths ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !methodMayHaveBody(c.Request.Method) || c.Request.Body == nil {
			c.Next()
			return
		}
		// Multipart endpoints enforce their own file-aware limits. Applying the
		// JSON body limit here would both reject multipart media types and count
		// multipart framing against the uploaded file's limit.
		if isMultipartMediaType(c.GetHeader("Content-Type")) {
			if multipartPathAllowed(c.Request.URL.Path, multipartPaths) {
				c.Next()
				return
			}
			response.ErrorResponseWithCode(c, http.StatusUnsupportedMediaType, unsupportedMediaTypeCode, "Multipart content is not supported for this endpoint.", nil)
			c.Abort()
			return
		}

		if c.Request.ContentLength > maxBytes {
			response.ErrorResponseWithCode(c, http.StatusRequestEntityTooLarge, requestBodyTooLargeCode, "Request body is too large.", nil)
			c.Abort()
			return
		}

		limitedBody := http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		if c.Request.ContentLength == 0 {
			c.Request.Body = limitedBody
			c.Next()
			return
		}

		// A chunked request has no reliable Content-Length. Peek a single byte so an
		// actually empty body remains compatible while a non-empty body is checked.
		firstByte := make([]byte, 1)
		n, err := limitedBody.Read(firstByte)
		if n > 0 {
			if !isJSONMediaType(c.GetHeader("Content-Type")) {
				response.ErrorResponseWithCode(c, http.StatusUnsupportedMediaType, unsupportedMediaTypeCode, "Content-Type must be application/json.", nil)
				c.Abort()
				return
			}
			c.Request.Body = io.NopCloser(io.MultiReader(bytes.NewReader(firstByte[:n]), limitedBody))
			c.Next()
			return
		}
		if err != nil && err != io.EOF {
			// MaxBytesReader reports this before JSON binding for a boundary-sized body.
			response.ErrorResponseWithCode(c, http.StatusRequestEntityTooLarge, requestBodyTooLargeCode, "Request body is too large.", nil)
			c.Abort()
			return
		}

		c.Request.Body = limitedBody
		c.Next()
	}
}

func isMultipartMediaType(contentType string) bool {
	mediaType, _, err := mime.ParseMediaType(contentType)
	return err == nil && strings.EqualFold(mediaType, "multipart/form-data")
}

func multipartPathAllowed(requestPath string, allowedPaths []string) bool {
	for _, allowedPath := range allowedPaths {
		if requestPath == allowedPath {
			return true
		}
	}
	return false
}

// CORSMiddleware enforces the configured exact-origin browser policy.
func CORSMiddleware(cfg config.CORSConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		wildcard := len(cfg.AllowedOrigins) == 1 && cfg.AllowedOrigins[0] == "*"
		if !wildcard {
			// Include denied and absent origins: caches must not reuse a response
			// selected under a different browser-origin decision.
			addVary(c.Writer.Header(), "Origin")
		}
		if c.Request.Method == http.MethodOptions {
			addVary(c.Writer.Header(), "Access-Control-Request-Method", "Access-Control-Request-Headers")
		}
		origin := c.GetHeader("Origin")
		if origin == "" {
			c.Next()
			return
		}

		if !corsOriginAllowed(origin, cfg.AllowedOrigins) {
			response.ErrorResponseWithCode(c, http.StatusForbidden, "CORS_ORIGIN_NOT_ALLOWED", "Origin is not allowed.", nil)
			c.Abort()
			return
		}

		if c.Request.Method == http.MethodOptions && !validPreflight(c.Request) {
			response.ErrorResponseWithCode(c, http.StatusForbidden, "CORS_PREFLIGHT_NOT_ALLOWED", "Preflight request is not allowed.", nil)
			c.Abort()
			return
		}

		if wildcard {
			c.Header("Access-Control-Allow-Origin", "*")
		} else {
			c.Header("Access-Control-Allow-Origin", origin)
		}
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type, X-LMS-Client, X-Request-ID, Accept")
		c.Header("Access-Control-Expose-Headers", "X-Request-ID, Retry-After")
		if cfg.AllowCredentials {
			c.Header("Access-Control-Allow-Credentials", "true")
		}

		if c.Request.Method == http.MethodOptions {
			c.Status(http.StatusNoContent)
			c.Abort()
			return
		}
		c.Next()
	}
}

func validPreflight(request *http.Request) bool {
	if len(request.Header.Values("Access-Control-Request-Method")) != 1 {
		return false
	}
	switch request.Header.Get("Access-Control-Request-Method") {
	case "GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS":
	default:
		return false
	}
	for _, value := range request.Header.Values("Access-Control-Request-Headers") {
		for _, name := range strings.Split(value, ",") {
			switch strings.ToLower(strings.TrimSpace(name)) {
			case "authorization", "content-type", "x-lms-client", "x-request-id", "accept":
			default:
				return false
			}
		}
	}
	return true
}

func addVary(header http.Header, names ...string) {
	for _, name := range names {
		found := false
		for _, value := range header.Values("Vary") {
			for _, existing := range strings.Split(value, ",") {
				if strings.EqualFold(strings.TrimSpace(existing), name) || strings.TrimSpace(existing) == "*" {
					found = true
				}
			}
		}
		if !found {
			header.Add("Vary", name)
		}
	}
}

func methodMayHaveBody(method string) bool {
	switch method {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	default:
		return false
	}
}

func isJSONMediaType(value string) bool {
	mediaType, _, err := mime.ParseMediaType(value)
	if err != nil {
		return false
	}
	return mediaType == "application/json" || strings.HasSuffix(mediaType, "+json") && strings.HasPrefix(mediaType, "application/")
}

func corsOriginAllowed(origin string, allowedOrigins []string) bool {
	for _, allowedOrigin := range allowedOrigins {
		if allowedOrigin == "*" || origin == allowedOrigin {
			return true
		}
	}
	return false
}
