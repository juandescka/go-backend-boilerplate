package handlers

import (
	"context"
	"errors"
	"net/http"
	"time"

	"hris-core-service/internal/shared/observability"
	"hris-core-service/internal/shared/response"
	"hris-core-service/internal/usecases/health"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

const readinessTimeout = 2 * time.Second

// OperationalHandler serves public liveness and readiness probes.
type OperationalHandler struct {
	checker   health.ReadinessChecker
	logger    *logrus.Logger
	lifecycle *health.LifecycleState
}

func NewOperationalHandler(checker health.ReadinessChecker, logger *logrus.Logger, lifecycle ...*health.LifecycleState) *OperationalHandler {
	handler := &OperationalHandler{checker: checker, logger: logger}
	if len(lifecycle) > 0 {
		handler.lifecycle = lifecycle[0]
	}
	return handler
}

// Health reports whether the HTTP application process is alive.
//
// @Summary Check application health
// @Description Returns the liveness status without checking dependencies.
// @Tags Operations
// @Accept json
// @Produce json
// @Success 200 {object} response.SuccessResponseBody "Application is healthy"
// @Router /health [get]
func (h *OperationalHandler) Health(c *gin.Context) {
	response.SuccessResponse(c, gin.H{"status": "ok"}, "Service is healthy.", nil, nil, nil)
}

// Ready reports whether the application can serve traffic using its primary database.
//
// @Summary Check application readiness
// @Description Returns readiness after a bounded primary database check.
// @Tags Operations
// @Accept json
// @Produce json
// @Success 200 {object} response.SuccessResponseBody "Application is ready"
// @Failure 503 {object} response.ErrorResponseBody "Service is not ready (SERVICE_NOT_READY)"
// @Router /ready [get]
func (h *OperationalHandler) Ready(c *gin.Context) {
	if h != nil && h.lifecycle != nil && h.lifecycle.IsShuttingDown() {
		h.logReadinessFailure(c, "shutting_down")
		response.ErrorResponseWithCode(c, http.StatusServiceUnavailable, "SERVICE_NOT_READY", "Service is not ready.", nil)
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), readinessTimeout)
	defer cancel()

	if h == nil || h.checker == nil {
		h.logReadinessFailure(c, "unavailable")
		response.ErrorResponseWithCode(c, http.StatusServiceUnavailable, "SERVICE_NOT_READY", "Service is not ready.", nil)
		return
	}
	if err := h.checker.Check(ctx); err != nil {
		h.logReadinessFailure(c, readinessReason(ctx, err))
		response.ErrorResponseWithCode(c, http.StatusServiceUnavailable, "SERVICE_NOT_READY", "Service is not ready.", nil)
		return
	}

	response.SuccessResponse(c, gin.H{"status": "ready"}, "Service is ready.", nil, nil, nil)
}

func (h *OperationalHandler) logReadinessFailure(c *gin.Context, reasonCode string) {
	if h == nil || h.logger == nil {
		return
	}
	fields := logrus.Fields{
		"event_code":     "application.readiness.failed",
		"event_category": "application",
		"request_id":     observability.RequestIDFromContext(c),
		"reason_code":    reasonCode,
	}
	if reasonCode != "shutting_down" {
		fields["dependency"] = "database"
	}
	message := "Service readiness check failed"
	if reasonCode == "shutting_down" {
		message = "Readiness rejected during shutdown"
	}
	h.logger.WithFields(fields).WithContext(c.Request.Context()).Warn(message)
}

func readinessReason(ctx context.Context, err error) string {
	if errors.Is(ctx.Err(), context.DeadlineExceeded) || errors.Is(err, context.DeadlineExceeded) {
		return "timeout"
	}
	return "ping_failed"
}
