package routes

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"hris-core-service/internal/handlers"
	"hris-core-service/internal/shared/observability"
	securityevent "hris-core-service/internal/shared/security"
	auth "hris-core-service/internal/usecases/authentication"
)

func TestLoginAbuseHTTP(t *testing.T) {
	for _, trusted := range []bool{false, true} {
		t.Run(map[bool]string{false: "direct", true: "trusted proxy"}[trusted], func(t *testing.T) {
			var logs bytes.Buffer
			log := logrus.New()
			log.SetOutput(&logs)
			log.SetFormatter(&logrus.JSONFormatter{})
			store := &authStore{findErr: auth.ErrUnauthenticated}
			service := auth.NewService(store, time.Minute)
			limiter, _ := securityevent.NewLoginLimiter(100, securityevent.LoginPolicy{Limit: 3, Window: time.Minute}, securityevent.LoginPolicy{Limit: 1, Window: 5 * time.Minute})
			router := gin.New()
			_ = router.SetTrustedProxies(nil)
			if trusted {
				_ = router.SetTrustedProxies([]string{"192.0.2.1"})
			}
			router.Use(observability.RequestIDMiddleware(), observability.RequestCompletionLogger(log))
			events := securityevent.NewLogrusRecorder(log)
			SetupAuthenticationRoutes(router, handlers.NewAuthenticationHandler(service, events, limiter), func(c *gin.Context) { c.Next() })
			post := func(email, forwarded string, ctx context.Context) *httptest.ResponseRecorder {
				data, _ := json.Marshal(map[string]string{"email": email, "password": "password-private-sentinel"})
				r := httptest.NewRequest("POST", "/v1/auth/login", bytes.NewReader(data)).WithContext(ctx)
				r.RemoteAddr = "192.0.2.1:1234"
				r.Header.Set("Content-Type", "application/json")
				r.Header.Set("X-Forwarded-For", forwarded)
				r.Header.Set("Authorization", "Bearer token-private-sentinel")
				w := httptest.NewRecorder()
				router.ServeHTTP(w, r)
				return w
			}
			ctx := context.Background()
			if w := post("USER@example.test", "198.51.100.1", ctx); w.Code != 401 {
				t.Fatalf("first login %d", w.Code)
			}
			w := post(" user@example.test ", "198.51.100.2", ctx)
			if w.Code != 429 || !validRetryAfter(w, 300) || !strings.Contains(w.Body.String(), "RATE_LIMIT_EXCEEDED") {
				t.Fatalf("identifier limit: %d %v", w.Code, w.Header())
			}
			if w := post("different@example.test", "198.51.100.3", ctx); w.Code != 401 {
				t.Fatal("distinct identifier blocked")
			}
			w = post("third@example.test", "198.51.100.4", ctx)
			if trusted && w.Code != 401 || !trusted && (w.Code != 429 || !validRetryAfter(w, 60)) {
				t.Fatalf("IP trust: %d", w.Code)
			}
			canceled, cancel := context.WithCancel(ctx)
			cancel()
			if w := post("cancel@example.test", "198.51.100.5", canceled); w.Code != 500 {
				t.Fatal("canceled security control failed open")
			}
			for _, secret := range []string{"password-private-sentinel", "token-private-sentinel", "USER@example.test", "user@example.test"} {
				if strings.Contains(logs.String(), secret) {
					t.Fatal("secret/identifier in logs")
				}
			}
			if strings.Count(logs.String(), `"event_code":"http.request.completed"`) != 5 {
				t.Fatal("completion duplication")
			}
			wantLimited := 2
			if trusted {
				wantLimited = 1
			}
			if strings.Count(logs.String(), `"event_code":"auth.login.rate_limited"`) != wantLimited {
				t.Fatal("throttle event duplication")
			}
		})
	}
}

func validRetryAfter(w *httptest.ResponseRecorder, max int) bool {
	seconds, err := strconv.Atoi(w.Header().Get("Retry-After"))
	return err == nil && seconds > 0 && seconds <= max
}
