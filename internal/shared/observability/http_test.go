package observability_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"hris-core-service/config"
	"hris-core-service/internal/domain/models"
	"hris-core-service/internal/handlers"
	"hris-core-service/internal/middlewares"
	"hris-core-service/internal/routes"
	"hris-core-service/internal/shared/observability"
	"hris-core-service/internal/shared/response"
	securityevent "hris-core-service/internal/shared/security"
	auth "hris-core-service/internal/usecases/authentication"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type loggingStore struct{ operational bool }

func (s loggingStore) NormalizeLoginIdentifier(_ context.Context, value string) (string, error) {
	return strings.ToLower(strings.Trim(value, " ")), nil
}
func (s loggingStore) FindUserByEmail(context.Context, string) (models.User, error) {
	if s.operational {
		return models.User{}, errors.New("private-database-sentinel")
	}
	return models.User{}, auth.ErrUnauthenticated
}
func (loggingStore) CreateSession(context.Context, models.User, models.AuthSession) error {
	return errors.New("unexpected session creation")
}
func (loggingStore) FindIdentity(context.Context, string) (models.Identity, error) {
	return models.Identity{}, auth.ErrUnauthenticated
}
func (loggingStore) RevokeSession(context.Context, models.Identity) error {
	return auth.ErrUnauthenticated
}
func (loggingStore) RevokeAllSessions(context.Context, models.Identity) error {
	return auth.ErrUnauthenticated
}

func loggingRouter(output *bytes.Buffer) (*gin.Engine, *logrus.Logger) {
	gin.SetMode(gin.TestMode)
	log := logrus.New()
	log.SetOutput(output)
	log.SetFormatter(&logrus.JSONFormatter{})
	router := gin.New()
	_ = router.SetTrustedProxies(nil)
	router.Use(observability.RequestIDMiddleware(), observability.RequestCompletionLogger(log), observability.PanicRecovery(log))
	return router, log
}

func readLogs(t *testing.T, output *bytes.Buffer) []map[string]any {
	t.Helper()
	var entries []map[string]any
	for _, line := range strings.Split(strings.TrimSpace(output.String()), "\n") {
		var entry map[string]any
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			t.Fatalf("invalid structured log: %v", err)
		}
		if _, exists := entry["message"]; exists {
			t.Fatal("custom message duplicates Logrus native msg")
		}
		entries = append(entries, entry)
	}
	return entries
}

func TestHTTPLoggingBoundaries(t *testing.T) {
	for _, tc := range []struct {
		name, method, path, body, code, level, message, diagnostic, diagnosticLevel string
		status                                                                      int
	}{
		{name: "success", method: "GET", path: "/health", status: 200, body: `{"success":true,"message":"Service is healthy.","data":{"status":"ok"}}`, level: "info", message: "HTTP request completed"},
		{name: "cors", method: "POST", path: "/v1/auth/login", status: 403, body: `{"error":{"code":"CORS_ORIGIN_NOT_ALLOWED","message":"Origin is not allowed."}}`, code: "CORS_ORIGIN_NOT_ALLOWED", level: "warning", message: "HTTP request rejected"},
		{name: "authentication", method: "POST", path: "/v1/auth/login", status: 401, body: `{"error":{"code":"UNAUTHORIZED","message":"Invalid email or password."}}`, code: "UNAUTHORIZED", level: "info", message: "HTTP request rejected", diagnostic: "auth.login.failed", diagnosticLevel: "info"},
		{name: "internal", method: "POST", path: "/v1/auth/login", status: 500, body: `{"error":{"code":"INTERNAL_SERVER_ERROR","message":"Authentication could not be completed."}}`, code: "INTERNAL_SERVER_ERROR", level: "error", message: "HTTP request failed", diagnostic: "auth.login.failed", diagnosticLevel: "error"},
		{name: "readiness", method: "GET", path: "/ready", status: 503, body: `{"error":{"code":"SERVICE_NOT_READY","message":"Service is not ready."}}`, code: "SERVICE_NOT_READY", level: "error", message: "HTTP request failed", diagnostic: "application.readiness.failed", diagnosticLevel: "warning"},
		{name: "panic", method: "GET", path: "/panic", status: 500, body: `{"error":{"code":"INTERNAL_SERVER_ERROR","message":"Internal Server Error"}}`, code: "INTERNAL_SERVER_ERROR", level: "error", message: "HTTP request failed", diagnostic: "http.request.panic", diagnosticLevel: "error"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var output bytes.Buffer
			router, log := loggingRouter(&output)
			router.Use(middlewares.CORSMiddleware(config.CORSConfig{AllowedOrigins: []string{"https://allowed.example.test"}}))
			routes.SetupRoutes(router, handlers.NewOperationalHandler(nil, log))
			service := auth.NewService(loggingStore{operational: tc.name == "internal"}, 15*time.Minute)
			events := securityevent.NewLogrusRecorder(log)
			routes.SetupAuthenticationRoutes(router, handlers.NewAuthenticationHandler(service, events, testLoginLimiter()), middlewares.BearerAuthentication(service, events))
			router.GET("/panic", func(*gin.Context) { panic("private-panic-sentinel") })
			request := httptest.NewRequest(tc.method, tc.path+"?token=private-query-sentinel", strings.NewReader(`{"email":"user@example.test","password":"private-password-sentinel"}`))
			request.Header.Set("Content-Type", "application/json")
			request.Header.Set("X-Request-ID", "log-fixture-"+tc.name)
			request.Header.Set("Authorization", "Bearer private-token-sentinel")
			request.Header.Set("Cookie", "session=private-cookie-sentinel")
			request.Header.Set("User-Agent", "test\r\n"+strings.Repeat("a", 300))
			if tc.name == "cors" {
				request.Header.Set("Origin", "https://rejected.example.test")
			} else {
				request.Header.Set("Origin", "https://allowed.example.test")
			}
			result := httptest.NewRecorder()
			router.ServeHTTP(result, request)
			if result.Code != tc.status || result.Body.String() != tc.body {
				t.Fatalf("response changed: status=%d body=%s", result.Code, result.Body.String())
			}
			id := "log-fixture-" + tc.name
			if result.Header().Get("X-Request-ID") != id {
				t.Fatal("response correlation changed")
			}
			completion, diagnostic := 0, 0
			for _, entry := range readLogs(t, &output) {
				if entry["request_id"] != id {
					t.Fatal("diagnostic/completion correlation mismatch")
				}
				if entry["event_code"] == "http.request.completed" {
					completion++
					if entry["level"] != tc.level || entry["msg"] != tc.message || entry["http_status"] != float64(tc.status) {
						t.Fatalf("wrong completion classification: %v", entry)
					}
					code, _ := entry["error_code"].(string)
					if code != tc.code || entry["http_path"] != tc.path || entry["http_route"] != tc.path {
						t.Fatalf("wrong completion metadata: %v", entry)
					}
					if agent := entry["user_agent"].(string); len([]rune(agent)) > 256 || strings.ContainsAny(agent, "\r\n") {
						t.Fatal("unsafe user agent")
					}
				} else if entry["event_code"] == tc.diagnostic && entry["level"] == tc.diagnosticLevel {
					diagnostic++
					if entry["msg"] == entry["event_code"] {
						t.Fatal("diagnostic lacks readable native message")
					}
				} else {
					t.Fatalf("unexpected duplicate/diagnostic: %v", entry)
				}
			}
			wantDiagnostic := 0
			if tc.diagnostic != "" {
				wantDiagnostic = 1
			}
			if completion != 1 || diagnostic != wantDiagnostic {
				t.Fatalf("completion=%d diagnostic=%d", completion, diagnostic)
			}
			for _, secret := range []string{"private-password-sentinel", "private-token-sentinel", "private-cookie-sentinel", "private-query-sentinel", "private-database-sentinel", "private-panic-sentinel", "user@example.test", "https://rejected.example.test", "https://allowed.example.test"} {
				if strings.Contains(output.String(), secret) {
					t.Fatalf("sensitive field leaked: %s", secret)
				}
			}
		})
	}
}

func TestErrorContextMatchesCommittedResponse(t *testing.T) {
	for _, tc := range []struct {
		name, code string
		status     int
		handler    gin.HandlerFunc
	}{
		{"unclassified", "", 401, func(c *gin.Context) { c.String(401, "plain denial") }},
		{"distinct-policy", "POLICY_REJECTED", 403, func(c *gin.Context) { response.ErrorResponseWithCode(c, 403, "POLICY_REJECTED", "Denied", nil) }},
		{"distinct-permission", "PERMISSION_DENIED", 403, func(c *gin.Context) { response.ErrorResponseWithCode(c, 403, "PERMISSION_DENIED", "Denied", nil) }},
		{"late-after-success", "", 200, func(c *gin.Context) {
			c.String(200, "ok")
			response.ErrorResponseWithCode(c, 500, "LATE_ERROR", "Late", nil)
		}},
		{"late-after-error", "FIRST_ERROR", 403, func(c *gin.Context) {
			response.ErrorResponseWithCode(c, 403, "FIRST_ERROR", "First", nil)
			response.ErrorResponseWithCode(c, 403, "SECOND_ERROR", "Late", nil)
		}},
		{"late-different-status", "FIRST_ERROR", 403, func(c *gin.Context) {
			response.ErrorResponseWithCode(c, 403, "FIRST_ERROR", "First", nil)
			response.ErrorResponseWithCode(c, 500, "SECOND_ERROR", "Late", nil)
		}},
		{"unsafe-code", "", 400, func(c *gin.Context) { response.ErrorResponseWithCode(c, 400, "untrusted\r\ncode", "Denied", nil) }},
		{"oversize-code", "", 400, func(c *gin.Context) { response.ErrorResponseWithCode(c, 400, strings.Repeat("A", 65), "Denied", nil) }},
		{"success-code", "", 200, func(c *gin.Context) { response.ErrorResponseWithCode(c, 200, "NOT_AN_ERROR", "OK", nil) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var output bytes.Buffer
			router, _ := loggingRouter(&output)
			router.GET("/test", tc.handler)
			result := httptest.NewRecorder()
			router.ServeHTTP(result, httptest.NewRequest("GET", "/test", nil))
			entries := readLogs(t, &output)
			if len(entries) != 1 || result.Code != tc.status {
				t.Fatal("unexpected completion/response")
			}
			code, exists := entries[0]["error_code"]
			if tc.code == "" && exists || tc.code != "" && code != tc.code {
				t.Fatalf("misleading error context: %v", entries[0])
			}
			if tc.name == "unclassified" && entries[0]["level"] != "warning" {
				t.Fatal("unclassified denial should retain warning fallback")
			}
		})
	}
}

func TestConcurrentRequestLogIsolation(t *testing.T) {
	var output bytes.Buffer
	router, _ := loggingRouter(&output)
	router.GET("/test/:kind", func(c *gin.Context) {
		if observability.RequestIDFromStandardContext(c.Request.Context()) != c.GetHeader("X-Request-ID") {
			panic("correlation mismatch")
		}
		switch c.Param("kind") {
		case "0":
			response.ErrorResponseWithCode(c, 403, "POLICY_REJECTED", "Denied", nil)
		case "1":
			c.String(200, "ok")
		default:
			c.String(400, "unclassified")
		}
	})
	const count = 90
	var wg sync.WaitGroup
	for i := 0; i < count; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			request := httptest.NewRequest("GET", fmt.Sprintf("/test/%d", i%3), nil)
			id := fmt.Sprintf("concurrent-%d", i)
			request.Header.Set("X-Request-ID", id)
			result := httptest.NewRecorder()
			router.ServeHTTP(result, request)
			if result.Header().Get("X-Request-ID") != id {
				t.Error("response ID isolation failed")
			}
		}(i)
	}
	wg.Wait()
	seen := make(map[string]bool)
	for _, entry := range readLogs(t, &output) {
		id, _ := entry["request_id"].(string)
		var i int
		if _, err := fmt.Sscanf(id, "concurrent-%d", &i); err != nil || i < 0 || i >= count || seen[id] {
			t.Fatal("unexpected/duplicate request ID")
		}
		seen[id] = true
		status := []int{403, 200, 400}[i%3]
		code, exists := entry["error_code"]
		if entry["event_code"] != "http.request.completed" || entry["http_status"] != float64(status) || i%3 == 0 && code != "POLICY_REJECTED" || i%3 != 0 && exists {
			t.Fatalf("request metadata crossed boundaries: %v", entry)
		}
	}
	if len(seen) != count {
		t.Fatal("missing request completions")
	}
}

func TestSecurityEventOperationalSeverity(t *testing.T) {
	for _, tc := range []struct{ code, outcome, reason, level, message string }{
		{securityevent.CodeContextInvalid, securityevent.OutcomeDenied, "invalid_session", "info", "Session authentication failed"},
		{securityevent.CodeLogoutFailed, securityevent.OutcomeBlocked, "operation_failed", "error", "Session logout failed"},
		{securityevent.CodeLoginFailed, securityevent.OutcomeBlocked, "verification_capacity", "warning", "Authentication failed"},
		{securityevent.CodeLogoutAllSuccess, securityevent.OutcomeSuccess, "", "info", "User sessions logged out"},
	} {
		t.Run(tc.code+"/"+tc.reason, func(t *testing.T) {
			var output bytes.Buffer
			_, log := loggingRouter(&output)
			securityevent.NewLogrusRecorder(log).Record(context.Background(), securityevent.Event{Code: tc.code, Outcome: tc.outcome, ReasonCode: tc.reason})
			entries := readLogs(t, &output)
			if len(entries) != 1 || entries[0]["event_code"] != tc.code || entries[0]["outcome"] != tc.outcome || entries[0]["level"] != tc.level || entries[0]["msg"] != tc.message {
				t.Fatalf("wrong security severity: %v", entries)
			}
		})
	}
}

func testLoginLimiter() *securityevent.LoginLimiter {
	limiter, err := securityevent.NewLoginLimiter(10000, securityevent.LoginPolicy{Limit: 1000, Window: time.Minute}, securityevent.LoginPolicy{Limit: 1000, Window: time.Minute})
	if err != nil {
		panic(err)
	}
	return limiter
}
