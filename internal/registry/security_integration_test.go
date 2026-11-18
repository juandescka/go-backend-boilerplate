package registry

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"hris-core-service/config"
	"hris-core-service/internal/handlers"
	"hris-core-service/internal/routes"
	securityevent "hris-core-service/internal/shared/security"
)

// Reuse the authentication suite's owned PostgreSQL fixture and cleanup. The
// ordinary lifecycle suite has generous fixture budgets; these checks use small
// independent budgets and the actual CLI to verify enforcement and composition.
func verifyLoginHardening(t *testing.T, reg *Registry, observer *pgx.Conn, binary, secret string) {
	t.Helper()
	limiter, _ := securityevent.NewLoginLimiter(100, securityevent.LoginPolicy{Limit: 100, Window: time.Minute}, securityevent.LoginPolicy{Limit: 1, Window: time.Minute})
	router := gin.New()
	_ = router.SetTrustedProxies(nil)
	routes.SetupAuthenticationRoutes(router, handlers.NewAuthenticationHandler(reg.Authentication, nil, limiter), func(c *gin.Context) { c.Next() })
	post := func(email, value string) *httptest.ResponseRecorder {
		t.Helper()
		body, _ := json.Marshal(map[string]string{"email": email, "password": value})
		r := httptest.NewRequest("POST", "/v1/auth/login", strings.NewReader(string(body)))
		r.RemoteAddr = "192.0.2.50:1234"
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)
		return w
	}
	for _, input := range []string{" INTEGRATION@MAPATU.TEST ", " İST@mapatu.test ", " Σ@mapatu.test "} {
		var normalized string
		if observer.QueryRow(context.Background(), "SELECT lower(btrim($1::text))", input).Scan(&normalized) != nil {
			t.Fatal("normalization fixture failed")
		}
		want := 401
		value := "incorrect"
		if strings.Contains(input, "INTEGRATION") {
			want = 200
			value = secret
		}
		if w := post(input, value); w.Code != want {
			t.Fatalf("DB-normalized first admission: %d want%d", w.Code, want)
		}
		if w := post(normalized, value); w.Code != 429 || w.Header().Get("Retry-After") == "" {
			t.Fatal("equivalent DB identifiers bypassed throttle")
		}
	}
	t.Log("PASS actual PostgreSQL case/space/non-ASCII identifier normalization, valid login and generic 429")

	// Exercise actual server composition with its own limiter, the built CLI,
	// and the same disposable DB; no developer administrator is touched.
	cfg := config.AppConfig
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal("test port unavailable")
	}
	cfg.App.Port = listener.Addr().(*net.TCPAddr).Port
	address := listener.Addr().String()
	_ = listener.Close()
	dir := t.TempDir()
	encoded, err := json.Marshal(cfg)
	if err != nil || os.WriteFile(filepath.Join(dir, "config.json"), encoded, 0600) != nil {
		t.Fatal("isolated server config unavailable")
	}
	command := exec.Command(binary, "server")
	command.Dir = dir
	command.Env = append(os.Environ(), "APP_ENV=development", "AUTH_LOGIN_IP_RATE_LIMIT=1000", "AUTH_LOGIN_IDENTIFIER_FAILURE_LIMIT=1", "AUTH_LOGIN_IDENTIFIER_FAILURE_WINDOW_SECONDS=3", "CORS_ALLOWED_ORIGINS=https://client.example.test", "CORS_ALLOW_CREDENTIALS=false", "TRUSTED_PROXIES=", "HTTP_READ_HEADER_TIMEOUT_SECONDS=1")
	command.Stdout, command.Stderr = io.Discard, io.Discard
	if command.Start() != nil {
		t.Fatal("isolated server failed to start")
	}
	done := make(chan error, 1)
	go func() { done <- command.Wait() }()
	stopped := false
	t.Cleanup(func() {
		if !stopped {
			_ = command.Process.Signal(os.Interrupt)
			select {
			case <-done:
			case <-time.After(5 * time.Second):
				_ = command.Process.Kill()
				<-done
			}
		}
	})
	transport := &http.Transport{}
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport, Timeout: 3 * time.Second}
	baseURL := "http://" + address
	deadline := time.Now().Add(10 * time.Second)
	for {
		r, err := client.Get(baseURL + "/health")
		if err == nil {
			_ = r.Body.Close()
			if r.StatusCode == 200 {
				break
			}
		}
		select {
		case <-done:
			stopped = true
			t.Fatal("isolated server exited before readiness (output withheld)")
		default:
		}
		if time.Now().After(deadline) {
			t.Fatal("isolated server startup timed out")
		}
		time.Sleep(20 * time.Millisecond)
	}
	call := func(method, path, body string, headers map[string]string) (int, http.Header, string) {
		t.Helper()
		r, _ := http.NewRequest(method, baseURL+path, strings.NewReader(body))
		for name, value := range headers {
			r.Header.Set(name, value)
		}
		resp, err := client.Do(r)
		if err != nil {
			t.Fatal("isolated HTTP request failed (details withheld)")
		}
		defer resp.Body.Close()
		data, err := io.ReadAll(io.LimitReader(resp.Body, 32768))
		if err != nil {
			t.Fatal("HTTP response read failed")
		}
		return resp.StatusCode, resp.Header, string(data)
	}
	status, _, _ := call("GET", "/ready", "", nil)
	if status != 200 {
		t.Fatal("isolated readiness failed")
	}
	status, header, _ := call("OPTIONS", "/v1/auth/login", "", map[string]string{"Origin": "https://client.example.test", "Access-Control-Request-Method": "POST", "Access-Control-Request-Headers": "Authorization, Content-Type"})
	if status != 204 || header.Get("Cache-Control") != "no-store" {
		t.Fatal("live preflight/security headers failed")
	}
	status, _, _ = call("OPTIONS", "/v1/auth/login", "", map[string]string{"Origin": "https://client.example.test", "Access-Control-Request-Method": "TRACE"})
	if status != 403 {
		t.Fatal("invalid live preflight accepted")
	}
	status, _, spec := call("GET", "/swagger/doc.json", "", nil)
	if status != 200 || !strings.Contains(spec, `"429"`) {
		t.Fatal("live generated throttle contract missing")
	}
	status, _, _ = call("GET", "/docs", "", nil)
	if status != 200 {
		t.Fatal("live Scalar route regression")
	}
	// Real incomplete-header transport check complements source/config review.
	conn, err := net.DialTimeout("tcp", address, time.Second)
	if err != nil {
		t.Fatal("timeout fixture connection failed")
	}
	_ = conn.SetDeadline(time.Now().Add(4 * time.Second))
	started := time.Now()
	_, err = io.WriteString(conn, "GET /health HTTP/1.1\r\nHost: localhost\r\n")
	if err != nil {
		_ = conn.Close()
		t.Fatal("timeout fixture write failed")
	}
	var first [1]byte
	_, readErr := conn.Read(first[:])
	_ = conn.Close()
	if netErr, ok := readErr.(net.Error); ok && netErr.Timeout() {
		t.Fatal("server failed to bound incomplete headers")
	}
	if time.Since(started) > 3*time.Second {
		t.Fatal("header timeout materially exceeded configured bound")
	}
	body, _ := json.Marshal(map[string]string{"email": "integration@mapatu.test", "password": secret})
	headers := map[string]string{"Content-Type": "application/json", "X-Forwarded-For": "198.51.100.99"}
	status, header, _ = call("POST", "/v1/auth/login", string(body), headers)
	if status != 200 || header.Get("Cache-Control") != "no-store" {
		t.Fatal("live password login failed")
	}
	status, header, payload := call("POST", "/v1/auth/login", string(body), headers)
	if status != 429 || header.Get("Retry-After") == "" || strings.Contains(payload, secret) || !strings.Contains(payload, "RATE_LIMIT_EXCEEDED") {
		t.Fatal("actual registry limiter not enforced safely")
	}
	deadline = time.Now().Add(6 * time.Second)
	for status == 429 && time.Now().Before(deadline) {
		time.Sleep(50 * time.Millisecond)
		status, _, _ = call("POST", "/v1/auth/login", string(body), headers)
	}
	if status != 200 {
		t.Fatal("live login did not recover after fixed-window expiration")
	}
	for i := 0; i < 12; i++ {
		status, _, _ = call("GET", "/health", "", nil)
		if status != 200 {
			t.Fatal("login limiter disrupted health")
		}
	}
	if command.Process.Signal(os.Interrupt) != nil {
		t.Fatal("isolated shutdown signal failed")
	}
	select {
	case err := <-done:
		stopped = true
		if err != nil {
			t.Fatal("isolated graceful shutdown failed")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("isolated shutdown timed out")
	}
	t.Log("PASS actual CLI HTTP server: preflight, no-store, generated 429, Scalar, incomplete-header timeout, login/throttle/expiration, health isolation and graceful shutdown")
}
