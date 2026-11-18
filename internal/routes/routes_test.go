package routes

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"hris-core-service/internal/handlers"
	appmetrics "hris-core-service/internal/shared/metrics"
	"hris-core-service/internal/shared/observability"
	"hris-core-service/internal/usecases/health"

	"github.com/gin-gonic/gin"
)

type readinessFunc func(context.Context) error

func (f readinessFunc) Check(ctx context.Context) error { return f(ctx) }

func TestOperationalRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tc := range []struct {
		name     string
		path     string
		checker  health.ReadinessChecker
		shutdown bool
		wantCode int
		wantBody string
	}{
		{"liveness ignores DB failure", "/health", readinessFunc(func(context.Context) error {
			t.Fatal("liveness must not call the database checker")
			return nil
		}), false, 200, `{"success":true,"message":"Service is healthy.","data":{"status":"ok"}}`},
		{"ready", "/ready", readinessFunc(func(ctx context.Context) error {
			deadline, ok := ctx.Deadline()
			if !ok || time.Until(deadline) > 2*time.Second {
				t.Fatal("readiness must set a deadline of at most two seconds")
			}
			return nil
		}), false, 200, `{"success":true,"message":"Service is ready.","data":{"status":"ready"}}`},
		{"ping failure", "/ready", readinessFunc(func(context.Context) error {
			return errors.New("sensitive database error")
		}), false, 503, `{"error":{"code":"SERVICE_NOT_READY","message":"Service is not ready."}}`},
		{"checker unavailable", "/ready", nil, false, 503, `{"error":{"code":"SERVICE_NOT_READY","message":"Service is not ready."}}`},
		{"shutting down", "/ready", readinessFunc(func(context.Context) error {
			t.Fatal("shutdown readiness must skip the database")
			return nil
		}), true, 503, `{"error":{"code":"SERVICE_NOT_READY","message":"Service is not ready."}}`},
		{"timeout", "/ready", readinessFunc(func(ctx context.Context) error {
			<-ctx.Done()
			return ctx.Err()
		}), false, 503, `{"error":{"code":"SERVICE_NOT_READY","message":"Service is not ready."}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			lifecycle := health.NewLifecycleState()
			if tc.shutdown {
				lifecycle.BeginShutdown()
			}
			router := gin.New()
			router.Use(observability.RequestIDMiddleware(), observability.PanicRecovery(nil))
			SetupRoutes(router, handlers.NewOperationalHandler(tc.checker, nil, lifecycle))
			res := httptest.NewRecorder()
			router.ServeHTTP(res, httptest.NewRequest(http.MethodGet, tc.path, nil))
			if res.Code != tc.wantCode || strings.TrimSpace(res.Body.String()) != tc.wantBody {
				t.Fatalf("got HTTP %d %s", res.Code, res.Body.String())
			}
			if !json.Valid(res.Body.Bytes()) || res.Header().Get("X-Request-ID") == "" {
				t.Fatal("expected JSON envelope and global request ID middleware")
			}
		})
	}
}

func TestReadinessHonorsRequestCancellation(t *testing.T) {
	router := gin.New()
	SetupRoutes(router, handlers.NewOperationalHandler(readinessFunc(func(ctx context.Context) error {
		if !errors.Is(ctx.Err(), context.Canceled) {
			t.Fatal("request cancellation was not propagated")
		}
		return ctx.Err()
	}), nil))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	res := httptest.NewRecorder()
	router.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/ready", nil).WithContext(ctx))
	if res.Code != http.StatusServiceUnavailable {
		t.Fatalf("got HTTP %d", res.Code)
	}
}

func TestMetricsRouteUsesExistingComponent(t *testing.T) {
	component, err := appmetrics.NewComponent(true, "test_hris")
	if err != nil {
		t.Fatal(err)
	}
	router := gin.New()
	SetupMetricsRoute(router, "/metrics", component.Handler())
	res := httptest.NewRecorder()
	router.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if res.Code != http.StatusOK || !strings.Contains(res.Body.String(), "test_hris_http_panics_total") {
		t.Fatalf("existing metrics component not exposed: HTTP %d", res.Code)
	}
}
