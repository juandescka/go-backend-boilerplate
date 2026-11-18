package cmd

import (
	"bytes"
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"hris-core-service/config"
	"hris-core-service/internal/shared/httphelper"
	"hris-core-service/internal/shared/observability"
	"hris-core-service/internal/shared/response"
)

func TestHTTPBoundarySecurity(t *testing.T) {
	router := gin.New()
	_ = router.SetTrustedProxies(nil)
	router.Use(observability.RequestIDMiddleware(), observability.PanicRecovery(nil))
	router.Use(func(c *gin.Context) { c.Writer.Header().Add("Vary", "Accept-Encoding"); c.Next() })
	router.Use(httpBoundaryMiddleware(config.CORSConfig{AllowedOrigins: []string{"https://client.example.test"}}, 1024)...)
	router.POST("/v1/auth/login", func(c *gin.Context) {
		if _, err := io.Copy(io.Discard, c.Request.Body); err != nil {
			if httphelper.IsRequestBodyTooLarge(err) {
				response.ErrorResponseWithCode(c, 413, "REQUEST_BODY_TOO_LARGE", "Request body is too large.", nil)
				return
			}
			t.Fatal(err)
		}
		c.Status(200)
	})
	router.GET("/health", func(c *gin.Context) { c.Status(200) })
	router.GET("/v1/auth/panic", func(*gin.Context) { panic("private SQL path credential sentinel") })
	for _, tc := range []struct {
		name, method, path, origin, requestedMethod, requestedHeaders, media, body string
		chunked                                                                    bool
		status                                                                     int
	}{
		{name: "allowed", method: "POST", path: "/v1/auth/login", origin: "https://client.example.test", media: "application/json", body: `{}`, status: 200},
		{name: "rejected", method: "POST", path: "/v1/auth/login", origin: "https://attacker.example.test", status: 403},
		{name: "origin exact", method: "POST", path: "/v1/auth/login", origin: "https://client.example.test.evil.test", status: 403},
		{name: "preflight", method: "OPTIONS", path: "/v1/auth/login", origin: "https://client.example.test", requestedMethod: "POST", requestedHeaders: "authorization, Content-Type, X-Request-ID", status: 204},
		{name: "preflight method", method: "OPTIONS", path: "/v1/auth/login", origin: "https://client.example.test", requestedMethod: "TRACE", status: 403},
		{name: "preflight header", method: "OPTIONS", path: "/v1/auth/login", origin: "https://client.example.test", requestedMethod: "POST", requestedHeaders: "X-Unsafe", status: 403},
		{name: "preflight absent method", method: "OPTIONS", path: "/v1/auth/login", origin: "https://client.example.test", status: 403},
		{name: "plain probe", method: "GET", path: "/health", status: 200},
		{name: "size known", method: "POST", path: "/v1/auth/login", media: "application/json", body: strings.Repeat("x", 1025), status: 413},
		{name: "size chunked", method: "POST", path: "/v1/auth/login", media: "application/json", body: strings.Repeat("x", 1025), chunked: true, status: 413},
		{name: "media", method: "POST", path: "/v1/auth/login", media: "text/plain", body: "secret", status: 415},
		{name: "panic", method: "GET", path: "/v1/auth/panic", status: 500},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
			if tc.chunked {
				r.ContentLength = -1
			}
			for name, value := range map[string]string{"Origin": tc.origin, "Access-Control-Request-Method": tc.requestedMethod, "Access-Control-Request-Headers": tc.requestedHeaders, "Content-Type": tc.media} {
				if value != "" {
					r.Header.Set(name, value)
				}
			}
			w := httptest.NewRecorder()
			router.ServeHTTP(w, r)
			if w.Code != tc.status {
				t.Fatalf("status=%d want=%d", w.Code, tc.status)
			}
			if w.Header().Get("X-Content-Type-Options") != "nosniff" || w.Header().Get("Referrer-Policy") != "no-referrer" || w.Header().Get("X-Frame-Options") != "DENY" {
				t.Fatal("security headers missing")
			}
			vary := strings.Join(w.Header().Values("Vary"), ",")
			if !strings.Contains(vary, "Origin") || !strings.Contains(vary, "Accept-Encoding") {
				t.Fatal("Vary lost on decision")
			}
			if tc.method == "OPTIONS" && (!strings.Contains(vary, "Access-Control-Request-Method") || !strings.Contains(vary, "Access-Control-Request-Headers")) {
				t.Fatal("preflight vary missing")
			}
			if strings.HasPrefix(tc.path, "/v1/auth/") && (w.Header().Get("Cache-Control") != "no-store" || w.Header().Get("Pragma") != "no-cache") {
				t.Fatal("auth early response cacheable")
			}
			if strings.Contains(w.Body.String(), "sentinel") || strings.Contains(w.Body.String(), "secret") {
				t.Fatal("private error leaked")
			}
			if tc.status == 204 && w.Body.Len() != 0 {
				t.Fatal("preflight body")
			}
			if tc.status == 403 && w.Header().Get("Access-Control-Allow-Origin") != "" {
				t.Fatal("rejected CORS response allowed")
			}
			if tc.name == "allowed" && (w.Header().Get("Access-Control-Allow-Origin") != tc.origin || w.Header().Get("Access-Control-Allow-Credentials") != "" || !strings.Contains(w.Header().Get("Access-Control-Expose-Headers"), "Retry-After")) {
				t.Fatal("CORS bearer policy")
			}
		})
	}
}

func TestBootstrapPasswordPolicy(t *testing.T) {
	for _, tc := range []struct {
		value []byte
		valid bool
	}{
		{[]byte(strings.Repeat("a", 14)), false}, {[]byte(strings.Repeat("a", 15)), true},
		{[]byte(strings.Repeat("界", 14)), false}, {[]byte(strings.Repeat("界", 15)), true},
		{bytes.Repeat([]byte("a"), 1024), true}, {bytes.Repeat([]byte("a"), 1025), false},
		{append(bytes.Repeat([]byte("a"), 15), 0xff), false},
	} {
		if validBootstrapPassword(tc.value) != tc.valid {
			t.Fatal("bootstrap password boundary mismatch")
		}
	}
}
