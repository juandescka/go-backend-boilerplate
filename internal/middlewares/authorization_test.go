package middlewares

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"hris-core-service/config"
	"hris-core-service/internal/domain/models"
	"hris-core-service/internal/shared/observability"
	auth "hris-core-service/internal/usecases/authentication"
	access "hris-core-service/internal/usecases/authorization"
)

type permissionStore struct {
	allowed bool
	err     error
	calls   int
}

func (s *permissionStore) Allows(context.Context, uuid.UUID, string, access.Scope) (bool, error) {
	s.calls++
	return s.allowed, s.err
}

func TestPermissionBoundary(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tenant := uuid.New()
	for _, tc := range []struct {
		name                 string
		identity             bool
		allowed              bool
		storeErr, resolveErr error
		scope                access.Scope
		permission           string
		status, calls        int
	}{
		{name: "unauthenticated", scope: access.Scope{Type: "TENANT", TenantID: tenant}, permission: "fixture.inspect", status: 401},
		{name: "membership is not permission", identity: true, scope: access.Scope{Type: "TENANT", TenantID: tenant}, permission: "fixture.inspect", status: 403, calls: 1},
		{name: "explicit allowed", identity: true, allowed: true, scope: access.Scope{Type: "TENANT", TenantID: tenant}, permission: "fixture.inspect", status: 200, calls: 1},
		{name: "database fails closed", identity: true, allowed: true, storeErr: errors.New("private sql sentinel"), scope: access.Scope{Type: "TENANT", TenantID: tenant}, permission: "fixture.inspect", status: 500, calls: 1},
		{name: "resource lookup fails closed", identity: true, resolveErr: errors.New("private resource sentinel"), permission: "fixture.inspect", status: 500},
		{name: "ownership mismatch", identity: true, resolveErr: access.ErrForbidden, permission: "fixture.inspect", status: 403},
		{name: "invalid shape", identity: true, allowed: true, scope: access.Scope{Type: "TENANT", TenantID: tenant, BranchID: uuid.New()}, permission: "fixture.inspect", status: 403},
		{name: "missing permission", identity: true, allowed: true, scope: access.Scope{Type: "TENANT", TenantID: tenant}, status: 403},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store := &permissionStore{allowed: tc.allowed, err: tc.storeErr}
			var logs bytes.Buffer
			logger := logrus.New()
			logger.SetFormatter(&logrus.JSONFormatter{})
			logger.SetOutput(&logs)
			router := gin.New()
			router.Use(observability.RequestIDMiddleware(), observability.RequestCompletionLogger(logger), SecurityHeaders(), CORSMiddleware(config.CORSConfig{}))
			called := false
			router.GET("/fixture", RequirePermission(access.NewService(store), tc.permission, func(*gin.Context) (access.Scope, error) { return tc.scope, tc.resolveErr }), func(c *gin.Context) {
				called = true
				if scope, ok := access.ScopeFromContext(c.Request.Context()); !ok || scope != tc.scope {
					t.Error("authorized scope was not passed to the handler")
				}
				c.Status(200)
			})
			request := httptest.NewRequest("GET", "/fixture", nil)
			request.Header.Set("X-Request-ID", "permission-boundary")
			if tc.identity {
				request = request.WithContext(auth.WithIdentity(request.Context(), models.Identity{UserID: uuid.New(), SessionID: uuid.New()}))
			}
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			if response.Code != tc.status || called != (tc.status == 200) || store.calls != tc.calls {
				t.Fatalf("status=%d handler=%v queries=%d", response.Code, called, store.calls)
			}
			if strings.Contains(response.Body.String(), "private") {
				t.Fatal("internal error leaked")
			}
			if response.Header().Get("X-Request-ID") != "permission-boundary" || response.Header().Get("X-Content-Type-Options") != "nosniff" {
				t.Fatal("authorization response lost global headers")
			}
			wantCode := map[int]string{401: "UNAUTHORIZED", 403: "FORBIDDEN", 500: "INTERNAL_SERVER_ERROR"}[tc.status]
			if tc.status != 200 {
				var envelope struct {
					Error struct{ Code string } `json:"error"`
				}
				if json.Unmarshal(response.Body.Bytes(), &envelope) != nil || envelope.Error.Code != wantCode {
					t.Fatal("authorization error envelope mismatch")
				}
			}
			var event struct {
				EventCode string `json:"event_code"`
				ErrorCode string `json:"error_code"`
				RequestID string `json:"request_id"`
				Status    int    `json:"http_status"`
			}
			// One JSON object proves a single completion event, including aborted
			// requests; private store/resolver errors must not reach telemetry.
			if json.Unmarshal(logs.Bytes(), &event) != nil || event.EventCode != "http.request.completed" || event.ErrorCode != wantCode || event.RequestID != "permission-boundary" || event.Status != tc.status || strings.Contains(logs.String(), "private") {
				t.Fatal("authorization completion correlation or privacy failed")
			}
		})
	}
}
