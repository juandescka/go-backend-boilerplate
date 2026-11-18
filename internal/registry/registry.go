package registry

import (
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"time"

	"hris-core-service/config"
	// "hris-core-service/internal/domain/repositories"
	// domainStorage "hris-core-service/internal/domain/storage"
	"hris-core-service/internal/handlers"
	"hris-core-service/internal/infrastructure"
	authstore "hris-core-service/internal/infrastructure/authentication"
	accessstore "hris-core-service/internal/infrastructure/authorization"
	"hris-core-service/internal/shared/logger"
	appmetrics "hris-core-service/internal/shared/metrics"
	securityevent "hris-core-service/internal/shared/security"
	auth "hris-core-service/internal/usecases/authentication"
	access "hris-core-service/internal/usecases/authorization"
	"hris-core-service/internal/usecases/health"

	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"

	"gorm.io/gorm"
)

type Registry struct {
	DB        *gorm.DB
	Logger    *logrus.Logger
	Metrics   *appmetrics.Component
	sqlDB     *sql.DB
	redis     *redis.Client
	closeOnce sync.Once
	closeErr  error

	// hanya expose yang dipakai luar
	OperationalHandler    *handlers.OperationalHandler
	LifecycleState        *health.LifecycleState
	Authentication        *auth.Service
	AuthenticationHandler *handlers.AuthenticationHandler
	Authorization         *access.Service
}

func NewRegistry() (*Registry, error) {
	if err := config.LoadConfig(); err != nil {
		return nil, err
	}

	applicationLogger := logger.NewLogger(config.AppConfig.App.Env)
	metricsComponent, err := appmetrics.NewComponent(
		config.AppConfig.Metrics.Enabled,
		config.AppConfig.Metrics.Namespace,
	)
	if err != nil {
		return nil, fmt.Errorf("initialize application metrics: %w", err)
	}
	db, sqlDB, err := infrastructure.ConnectDB()
	if err != nil {
		return nil, err
	}
	if err := metricsComponent.RegisterDBStats(sqlDB); err != nil {
		_ = sqlDB.Close()
		return nil, err
	}

	limiter, err := securityevent.NewLoginLimiter(config.AppConfig.RateLimiter.MaxEntries,
		securityevent.LoginPolicy{Limit: config.AppConfig.RateLimiter.AuthLoginIPLimit, Window: time.Duration(config.AppConfig.RateLimiter.AuthLoginIPWindowSeconds) * time.Second},
		securityevent.LoginPolicy{Limit: config.AppConfig.RateLimiter.AuthLoginIdentifierFailureLimit, Window: time.Duration(config.AppConfig.RateLimiter.AuthLoginIdentifierFailureWindowSeconds) * time.Second})
	if err != nil {
		_ = sqlDB.Close()
		return nil, err
	}

	// =========================
	// Unit of Work
	// =========================
	// uow := mysql.NewUnitOfWorkWithMetrics(db, time.Duration(config.AppConfig.Database.TransactionTimeoutSeconds)*time.Second, metricsComponent)

	// Operational and global authentication dependencies; no authorization grants.
	lifecycleState := health.NewLifecycleState()
	operationalHandler := handlers.NewOperationalHandler(infrastructure.NewDatabaseReadinessChecker(sqlDB), applicationLogger, lifecycleState)
	authentication := auth.NewService(authstore.NewStore(db, time.Duration(config.AppConfig.Database.QueryTimeoutSeconds)*time.Second), time.Duration(config.AppConfig.Authentication.SessionLifetimeMinutes)*time.Minute)

	return &Registry{
		DB:      db,
		Logger:  applicationLogger,
		Metrics: metricsComponent,
		sqlDB:   sqlDB,

		OperationalHandler:    operationalHandler,
		LifecycleState:        lifecycleState,
		Authentication:        authentication,
		Authorization:         access.NewService(accessstore.NewStore(sqlDB, time.Duration(config.AppConfig.Database.QueryTimeoutSeconds)*time.Second)),
		AuthenticationHandler: handlers.NewAuthenticationHandler(authentication, securityevent.NewLogrusRecorder(applicationLogger), limiter),
	}, nil
}

// Close releases process-scoped dependency clients exactly once during shutdown.
func (r *Registry) Close() error {
	if r == nil {
		return nil
	}
	r.closeOnce.Do(func() {
		var closeErrors []error
		if r.redis != nil {
			if err := r.redis.Close(); err != nil && !errors.Is(err, redis.ErrClosed) {
				if r.Logger != nil {
					r.Logger.WithFields(logrus.Fields{
						"event_code":     "redis.client.close_failed",
						"event_category": "application",
						"dependency":     "redis",
					}).WithError(err).Warn("redis.client.close_failed")
				}
			} else if r.Logger != nil {
				r.Logger.WithFields(logrus.Fields{
					"event_code":     "redis.client.closed",
					"event_category": "application",
					"dependency":     "redis",
				}).Info("redis.client.closed")
			}
		}
		if r.sqlDB != nil {
			if err := r.sqlDB.Close(); err != nil {
				closeErrors = append(closeErrors, fmt.Errorf("close database pool: %w", err))
			}
		}
		r.closeErr = errors.Join(closeErrors...)
	})
	return r.closeErr
}
