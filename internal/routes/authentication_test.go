package routes

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"hris-core-service/internal/domain/models"
	"hris-core-service/internal/handlers"
	"hris-core-service/internal/middlewares"
	"hris-core-service/internal/shared/observability"
	"hris-core-service/internal/shared/password"
	securityevent "hris-core-service/internal/shared/security"
	auth "hris-core-service/internal/usecases/authentication"
)

type authStore struct {
	user                            models.User
	identity                        models.Identity
	findErr, identityErr, createErr error
	lookups                         int
	revokeErr                       error
	revoked                         models.Identity
	revokedAll                      bool
	revokeCalls                     int
}

func (s *authStore) NormalizeLoginIdentifier(_ context.Context, value string) (string, error) {
	return strings.ToLower(strings.Trim(value, " ")), nil
}
func (s *authStore) FindUserByEmail(context.Context, string) (models.User, error) {
	return s.user, s.findErr
}
func (s *authStore) CreateSession(context.Context, models.User, models.AuthSession) error {
	return s.createErr
}
func (s *authStore) FindIdentity(context.Context, string) (models.Identity, error) {
	s.lookups++
	return s.identity, s.identityErr
}

func (s *authStore) RevokeSession(_ context.Context, identity models.Identity) error {
	s.revoked, s.revokedAll = identity, false
	s.revokeCalls++
	return s.revokeErr
}
func (s *authStore) RevokeAllSessions(_ context.Context, identity models.Identity) error {
	s.revoked, s.revokedAll = identity, true
	s.revokeCalls++
	return s.revokeErr
}
func authRouter(store *authStore, log *logrus.Logger) *gin.Engine {
	router := gin.New()
	_ = router.SetTrustedProxies(nil)
	router.Use(observability.RequestIDMiddleware(), observability.RequestCompletionLogger(log), middlewares.RequestBodyLimit(1024*1024))
	service := auth.NewService(store, 15*time.Minute)
	events := securityevent.NewLogrusRecorder(log)
	SetupAuthenticationRoutes(router, handlers.NewAuthenticationHandler(service, events, testLoginLimiter()), middlewares.BearerAuthentication(service, events))
	return router
}

func TestBearerMiddlewareFailuresAndContextIsolation(t *testing.T) {
	token := base64.RawURLEncoding.EncodeToString(make([]byte, 32))
	store := &authStore{identity: models.Identity{UserID: uuid.New(), SessionID: uuid.New(), Email: "user@example.com"}}
	router := authRouter(store, nil)
	for _, header := range []string{"", "Bearer", "Basic xyz", "Bearer ", "Bearer abc", "Bearer " + token + " extra", "Bearer  " + token} {
		request := httptest.NewRequest("GET", "/v1/auth/me?access_token="+token, nil)
		if header != "" {
			request.Header.Set("Authorization", header)
		}
		result := httptest.NewRecorder()
		router.ServeHTTP(result, request)
		if result.Code != 401 || !strings.Contains(result.Body.String(), "UNAUTHORIZED") {
			t.Fatal("malformed/missing bearer accepted")
		}
	}
	if store.lookups != 0 {
		t.Fatal("malformed header queried persistence")
	}
	// Real expiry/revocation/user-state queries are covered by the PostgreSQL
	// integration; this fake tests only the middleware's storage-denial contract.
	store.identityErr = auth.ErrUnauthenticated
	request := httptest.NewRequest("GET", "/v1/auth/me", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	result := httptest.NewRecorder()
	router.ServeHTTP(result, request)
	if result.Code != 401 || result.Header().Get("WWW-Authenticate") != "Bearer" || !strings.Contains(result.Body.String(), "UNAUTHORIZED") {
		t.Fatal("storage denial must return the generic bearer challenge")
	}
	store.identityErr = nil
	request = httptest.NewRequest("GET", "/v1/auth/me", nil)
	request.Header.Set("Authorization", "bearer "+token)
	result = httptest.NewRecorder()
	router.ServeHTTP(result, request)
	if result.Code != 200 || !strings.Contains(result.Body.String(), store.identity.UserID.String()) || strings.Contains(result.Body.String(), "session_id") {
		t.Fatal("safe current user failed")
	}
	result = httptest.NewRecorder()
	router.ServeHTTP(result, httptest.NewRequest("GET", "/v1/auth/me", nil))
	if result.Code != 401 {
		t.Fatal("identity leaked to next request")
	}
	request = httptest.NewRequest("GET", "/v1/auth/me", nil)
	request.Header.Add("Authorization", "Bearer "+token)
	request.Header.Add("Authorization", "Bearer "+token)
	result = httptest.NewRecorder()
	router.ServeHTTP(result, request)
	if result.Code != 401 {
		t.Fatal("duplicate Authorization accepted")
	}
	store.identityErr = errors.New("private SQL failure")
	request = httptest.NewRequest("GET", "/v1/auth/me", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	result = httptest.NewRecorder()
	router.ServeHTTP(result, request)
	if result.Code != 500 || strings.Contains(result.Body.String(), "private") {
		t.Fatal("operational error not safely distinguished")
	}
}

func TestLoginHTTPValidationGenericDenialAndSafeLogging(t *testing.T) {
	secret := uuid.NewString()
	hash, err := password.Hash(secret)
	if err != nil {
		t.Fatal(err)
	}
	user := models.User{ID: uuid.New(), Email: "user@example.com", DisplayName: "Test", PasswordHash: &hash, IsActive: true}
	store := &authStore{user: user}
	var logs bytes.Buffer
	log := logrus.New()
	log.SetOutput(&logs)
	log.SetFormatter(&logrus.JSONFormatter{})
	router := authRouter(store, log)
	post := func(body string) *httptest.ResponseRecorder {
		t.Helper()
		r := httptest.NewRequest("POST", "/v1/auth/login", strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("Authorization", "secret-header-sentinel")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)
		return w
	}
	for _, tc := range []struct {
		body   string
		status int
	}{
		{"{", 400}, {`{"email":123}`, 400}, {`{} {}`, 400}, {`null`, 422}, {`{}`, 422}, {`{"email":"invalid","password":"x"}`, 422}, {`{"email":"user@example.com","password":""}`, 422},
		{`{"email":"user@example.com","password":"` + strings.Repeat("x", 1025) + `"}`, 422}, {`{"email":"user@example.com","password":"` + strings.Repeat("x", 17000) + `"}`, 413},
	} {
		if result := post(tc.body); result.Code != tc.status {
			t.Fatalf("input boundary: got%d want%d", result.Code, tc.status)
		}
	}
	encode := func(email, value string) string {
		b, _ := json.Marshal(map[string]string{"email": email, "password": value})
		return string(b)
	}
	denied := post(encode(user.Email, "incorrect"))
	if denied.Code != 401 {
		t.Fatal("wrong password accepted")
	}
	generic := denied.Body.String()
	for _, mutate := range []func(){func() { store.findErr = auth.ErrUnauthenticated }, func() { store.findErr = nil; store.user.IsActive = false }, func() { store.user.IsActive = true; store.user.PasswordHash = nil }} {
		mutate()
		result := post(encode(user.Email, secret))
		if result.Code != 401 || result.Body.String() != generic {
			t.Fatal("account state publicly distinguishable")
		}
	}
	store.user = user
	result := post(encode(" USER@example.com ", secret))
	if result.Code != 200 || result.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("valid login failed")
	}
	var payload struct {
		Data struct {
			AccessToken string `json:"access_token"`
		}
	}
	if json.Unmarshal(result.Body.Bytes(), &payload) != nil || !auth.ValidToken(payload.Data.AccessToken) {
		t.Fatal("token missing")
	}
	for _, sensitive := range []string{secret, hash, payload.Data.AccessToken, "secret-header-sentinel"} {
		if strings.Contains(logs.String(), sensitive) {
			t.Fatal("credential leaked into logs")
		}
	}
	if strings.Contains(result.Body.String(), "password_hash") || strings.Contains(result.Body.String(), "token_hash") {
		t.Fatal("hash in response")
	}
	store.findErr = errors.New("private SQL")
	result = post(encode(user.Email, secret))
	if result.Code != 500 || strings.Contains(result.Body.String(), "private") {
		t.Fatal("DB failure leaked or mapped to credential denial")
	}
	store.findErr = nil
	store.createErr = errors.New("private persistence failure")
	result = post(encode(user.Email, secret))
	if result.Code != 500 || strings.Contains(result.Body.String(), "access_token") {
		t.Fatal("persistence failure returned token")
	}
}

func TestSessionLifecycleHTTPContracts(t *testing.T) {
	token := base64.RawURLEncoding.EncodeToString(make([]byte, 32))
	identity := models.Identity{UserID: uuid.New(), SessionID: uuid.New(), Email: "user@example.com"}
	store := &authStore{identity: identity}
	var logs bytes.Buffer
	log := logrus.New()
	log.SetOutput(&logs)
	log.SetFormatter(&logrus.JSONFormatter{})
	router := authRouter(store, log)
	for _, path := range []string{"/v1/auth/logout", "/v1/auth/logout-all"} {
		request := httptest.NewRequest("POST", path, nil)
		result := httptest.NewRecorder()
		before := store.revokeCalls
		router.ServeHTTP(result, request)
		if result.Code != 401 || store.revokeCalls != before {
			t.Fatal("logout route is not protected")
		}
		for _, tc := range []struct {
			err    error
			status int
		}{{nil, 200}, {auth.ErrUnauthenticated, 401}, {errors.New("private revocation SQL"), 500}} {
			store.revokeErr = tc.err
			body := `{"user_id":"` + uuid.NewString() + `","session_id":"` + uuid.NewString() + `"}`
			request = httptest.NewRequest("POST", path, strings.NewReader(body))
			request.Header.Set("Content-Type", "application/json")
			request.Header.Set("Authorization", "Bearer "+token)
			result = httptest.NewRecorder()
			router.ServeHTTP(result, request)
			if result.Code != tc.status || store.revoked.UserID != identity.UserID || store.revoked.SessionID != identity.SessionID || store.revokedAll != (path == "/v1/auth/logout-all") {
				t.Fatal("revocation must use only the authenticated identity")
			}
			if result.Header().Get("Cache-Control") != "no-store" || strings.Contains(result.Body.String(), "private") || strings.Contains(result.Body.String(), token) || strings.Contains(result.Body.String(), "token_hash") {
				t.Fatal("unsafe lifecycle response")
			}
			if tc.err == nil && (!strings.Contains(result.Body.String(), `"data":null`) || !strings.Contains(result.Body.String(), `"success":true`)) {
				t.Fatal("logout must use the existing success envelope")
			}
		}
	}
	if strings.Contains(logs.String(), token) || strings.Contains(logs.String(), "private revocation SQL") {
		t.Fatal("logout credential/error leaked into logs")
	}
}

func testLoginLimiter() *securityevent.LoginLimiter {
	limiter, err := securityevent.NewLoginLimiter(10000, securityevent.LoginPolicy{Limit: 1000, Window: time.Minute}, securityevent.LoginPolicy{Limit: 1000, Window: time.Minute})
	if err != nil {
		panic(err)
	}
	return limiter
}
