package cmd

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cobra"

	"hris-core-service/config"
	"hris-core-service/internal/middlewares"
	"hris-core-service/internal/registry"
	"hris-core-service/internal/routes"
	"hris-core-service/internal/shared/observability"
	"hris-core-service/internal/shared/response"
	securityevent "hris-core-service/internal/shared/security"

	"github.com/sirupsen/logrus"
)

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Start the API server",
	RunE: func(cmd *cobra.Command, args []string) error {

		// 🔥 Registry (DI)
		reg, err := registry.NewRegistry()
		if err != nil {
			return fmt.Errorf("failed to initialize registry: %w", err)
		}

		suppressGinRouteRegistrationOutput()
		router := gin.New()
		if err := router.SetTrustedProxies(config.AppConfig.RateLimiter.TrustedProxies); err != nil {
			_ = reg.Close()
			return fmt.Errorf("trusted proxy configuration was validated during startup: %w", err)
		}
		router.HandleMethodNotAllowed = true
		router.Use(
			observability.RequestIDMiddleware(),
			observability.RequestCompletionLogger(reg.Logger),
		)
		if reg.Metrics.Enabled() {
			router.Use(middlewares.HTTPMetrics(reg.Metrics, config.AppConfig.Metrics.Path))
			router.Use(observability.PanicRecovery(reg.Logger, reg.Metrics))
		} else {
			router.Use(observability.PanicRecovery(reg.Logger))
		}
		router.Use(httpBoundaryMiddleware(config.AppConfig.CORS, config.AppConfig.HTTP.MaxRequestBodyBytes)...)

		// 🔥 Routes (pass semua handler)
		routes.SetupRoutes(
			router,
			reg.OperationalHandler,
		)
		routes.SetupAPIDocsRoutes(router, config.AppConfig.App.Env)
		routes.SetupAuthenticationRoutes(router, reg.AuthenticationHandler, middlewares.BearerAuthentication(reg.Authentication, securityevent.NewLogrusRecorder(reg.Logger)))
		if reg.Metrics.Enabled() {
			routes.SetupMetricsRoute(router, config.AppConfig.Metrics.Path, reg.Metrics.Handler())
		}
		router.NoRoute(func(c *gin.Context) {
			response.ErrorResponseWithCode(c, http.StatusNotFound, "ROUTE_NOT_FOUND", "The requested endpoint was not found.", nil)
		})
		router.NoMethod(func(c *gin.Context) {
			response.ErrorResponseWithCode(c, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "The HTTP method is not allowed for this endpoint.", nil)
		})

		port := config.AppConfig.App.Port
		addr := fmt.Sprintf(":%d", port)
		server := &http.Server{
			Addr:              addr,
			Handler:           router,
			ReadHeaderTimeout: time.Duration(config.AppConfig.HTTP.ReadHeaderTimeoutSeconds) * time.Second,
			ReadTimeout:       time.Duration(config.AppConfig.HTTP.ReadTimeoutSeconds) * time.Second,
			WriteTimeout:      time.Duration(config.AppConfig.HTTP.WriteTimeoutSeconds) * time.Second,
			IdleTimeout:       time.Duration(config.AppConfig.HTTP.IdleTimeoutSeconds) * time.Second,
			MaxHeaderBytes:    config.AppConfig.HTTP.MaxHeaderBytes,
		}

		serveErrors := make(chan error, 1)
		signals := make(chan os.Signal, 1)
		signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
		defer signal.Stop(signals)

		listener, err := net.Listen("tcp", addr)
		if err != nil {
			_ = reg.Close()
			return fmt.Errorf("listen for HTTP requests: %w", err)
		}
		reg.Logger.WithFields(logrus.Fields{
			"event_code":     "application.server.started",
			"event_category": "application",
			"listen_address": addr,
		}).Info("HTTP server started")
		go func() { serveErrors <- server.Serve(listener) }()

		select {
		case err := <-serveErrors:
			if !errors.Is(err, http.ErrServerClosed) {
				reg.Logger.WithFields(logrus.Fields{
					"event_code":     "application.server.start_failed",
					"event_category": "application",
					"listen_address": addr,
				}).WithError(err).Error("HTTP server failed")
				_ = reg.Close()
				return fmt.Errorf("server failed: %w", err)
			}
			return reg.Close()
		case <-signals:
		}

		reg.LifecycleState.BeginShutdown()
		reg.Logger.WithFields(logrus.Fields{
			"event_code":     "application.server.shutdown_started",
			"event_category": "application",
			"listen_address": addr,
		}).Info("HTTP server shutdown started")

		shutdownContext, cancel := context.WithTimeout(context.Background(), time.Duration(config.AppConfig.HTTP.ShutdownTimeoutSeconds)*time.Second)
		defer cancel()
		shutdownErr := server.Shutdown(shutdownContext)
		if shutdownErr != nil {
			reg.Logger.WithFields(logrus.Fields{
				"event_code":     "application.server.shutdown_forced",
				"event_category": "application",
				"listen_address": addr,
			}).WithError(shutdownErr).Warn("HTTP server shutdown forced")
			if closeErr := server.Close(); closeErr != nil && !errors.Is(closeErr, http.ErrServerClosed) {
				shutdownErr = fmt.Errorf("shutdown timed out: %w; forced close failed: %v", shutdownErr, closeErr)
			}
		}
		if closeErr := reg.Close(); closeErr != nil {
			return fmt.Errorf("close application resources: %w", closeErr)
		}
		reg.Logger.WithFields(logrus.Fields{
			"event_code":     "application.server.shutdown_completed",
			"event_category": "application",
			"listen_address": addr,
		}).Info("HTTP server shutdown completed")
		return shutdownErr
	},
}

func multipartUploadPaths() []string {
	return []string{
		"/v1/me/attendance/face-enrollment",
		"/v1/me/attendance/check-in",
		"/v1/me/attendance/check-out",
		"/v1/student-imports",
		"/v1/questions/import/validate",
		"/v1/questions/import",
		"/v1/ptks/import/validate",
		"/v1/ptks/import",
	}
}

func httpBoundaryMiddleware(cors config.CORSConfig, maxBytes int64) []gin.HandlerFunc {
	return []gin.HandlerFunc{
		middlewares.SecurityHeaders(),
		middlewares.CORSMiddleware(cors),
		middlewares.RequestBodyLimit(maxBytes, multipartUploadPaths()...),
	}
}

func suppressGinRouteRegistrationOutput() {
	gin.DebugPrintRouteFunc = func(string, string, string, int) {}
}

// 🔥 register command hanya di sini
func init() {
	RootCmd.AddCommand(serverCmd)
}
