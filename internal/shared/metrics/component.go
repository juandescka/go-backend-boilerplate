package metrics

import (
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var httpDurationBuckets = []float64{0.005, 0.010, 0.025, 0.050, 0.100, 0.250, 0.500, 1, 2.5, 5}

// DBStatsProvider is implemented by the application's shared *sql.DB.
type DBStatsProvider interface {
	Stats() sql.DBStats
}

// Component owns one dedicated Prometheus registry and all application collectors.
type Component struct {
	enabled   bool
	namespace string
	registry  *prometheus.Registry
	handler   http.Handler

	httpRequests  *prometheus.CounterVec
	httpDuration  *prometheus.HistogramVec
	httpInFlight  prometheus.Gauge
	httpPanics    prometheus.Counter
	cacheOps      *prometheus.CounterVec
	login         *prometheus.CounterVec
	refresh       *prometheus.CounterVec
	authContext   *prometheus.CounterVec
	authorization *prometheus.CounterVec
	rateLimit     *prometheus.CounterVec
	dbTimeouts    *prometheus.CounterVec
}

// NewComponent creates an isolated registry when enabled and a safe no-op component otherwise.
func NewComponent(enabled bool, namespace string) (*Component, error) {
	component := &Component{enabled: enabled, namespace: namespace}
	if !enabled {
		return component, nil
	}

	component.registry = prometheus.NewRegistry()
	component.httpRequests = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace, Name: "http_requests_total", Help: "Total completed application HTTP requests.",
	}, []string{"method", "route", "status"})
	component.httpDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: namespace, Name: "http_request_duration_seconds", Help: "Application HTTP request duration in seconds.", Buckets: httpDurationBuckets,
	}, []string{"method", "route", "status"})
	component.httpInFlight = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace, Name: "http_requests_in_flight", Help: "Application HTTP requests currently in flight.",
	})
	component.httpPanics = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: namespace, Name: "http_panics_total", Help: "Total HTTP panics caught by application recovery.",
	})
	component.cacheOps = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace, Name: "user_cache_operations_total", Help: "Total Redis-backed User cache operations by bounded outcome.",
	}, []string{"operation", "result"})
	component.login = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace, Name: "auth_login_total", Help: "Total Login operations by final result.",
	}, []string{"result"})
	component.refresh = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace, Name: "auth_refresh_total", Help: "Total Refresh operations by final result.",
	}, []string{"result"})
	component.authContext = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace, Name: "auth_context_total", Help: "Total database-backed Authentication Context validations by result.",
	}, []string{"result"})
	component.authorization = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace, Name: "authorization_total", Help: "Total Permission authorization decisions by result.",
	}, []string{"result"})
	component.rateLimit = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace, Name: "rate_limit_blocked_total", Help: "Total requests blocked by a bounded rate-limit policy.",
	}, []string{"policy"})
	component.dbTimeouts = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace, Name: "database_timeouts_total", Help: "Total centrally classified database timeouts by type.",
	}, []string{"type"})

	registered := []prometheus.Collector{
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
		component.httpRequests,
		component.httpDuration,
		component.httpInFlight,
		component.httpPanics,
		component.cacheOps,
		component.login,
		component.refresh,
		component.authContext,
		component.authorization,
		component.rateLimit,
		component.dbTimeouts,
	}
	for _, collector := range registered {
		if err := component.registry.Register(collector); err != nil {
			return nil, fmt.Errorf("register application metrics collector: %w", err)
		}
	}
	component.handler = promhttp.HandlerFor(component.registry, promhttp.HandlerOpts{})
	return component, nil
}

func (c *Component) Enabled() bool {
	return c != nil && c.enabled
}

func (c *Component) Handler() http.Handler {
	if c == nil {
		return nil
	}
	return c.handler
}

// Gatherer exposes the dedicated gatherer for focused endpoint wiring and tests.
func (c *Component) Gatherer() prometheus.Gatherer {
	if c == nil || c.registry == nil {
		return nil
	}
	return c.registry
}

// RegisterDBStats registers scrape-time functions against the existing shared pool.
func (c *Component) RegisterDBStats(provider DBStatsProvider) error {
	if !c.Enabled() {
		return nil
	}
	if provider == nil {
		return fmt.Errorf("database stats provider is nil")
	}

	collectorsToRegister := []prometheus.Collector{
		prometheus.NewGaugeFunc(prometheus.GaugeOpts{Namespace: c.namespace, Name: "db_connections_max_open", Help: "Maximum open PostgreSQL connections."}, func() float64 {
			return float64(provider.Stats().MaxOpenConnections)
		}),
		prometheus.NewGaugeFunc(prometheus.GaugeOpts{Namespace: c.namespace, Name: "db_connections_open", Help: "Current open PostgreSQL connections."}, func() float64 {
			return float64(provider.Stats().OpenConnections)
		}),
		prometheus.NewGaugeFunc(prometheus.GaugeOpts{Namespace: c.namespace, Name: "db_connections_in_use", Help: "Current in-use PostgreSQL connections."}, func() float64 {
			return float64(provider.Stats().InUse)
		}),
		prometheus.NewGaugeFunc(prometheus.GaugeOpts{Namespace: c.namespace, Name: "db_connections_idle", Help: "Current idle PostgreSQL connections."}, func() float64 {
			return float64(provider.Stats().Idle)
		}),
		prometheus.NewCounterFunc(prometheus.CounterOpts{Namespace: c.namespace, Name: "db_connections_wait_total", Help: "Total waits for a PostgreSQL connection."}, func() float64 {
			return float64(provider.Stats().WaitCount)
		}),
		prometheus.NewCounterFunc(prometheus.CounterOpts{Namespace: c.namespace, Name: "db_connections_wait_duration_seconds", Help: "Total time spent waiting for PostgreSQL connections in seconds."}, func() float64 {
			return provider.Stats().WaitDuration.Seconds()
		}),
	}
	for _, collector := range collectorsToRegister {
		if err := c.registry.Register(collector); err != nil {
			return fmt.Errorf("register database pool metrics collector: %w", err)
		}
	}
	return nil
}

func (c *Component) RequestStarted() {
	if c.Enabled() {
		c.httpInFlight.Inc()
	}
}

func (c *Component) RequestFinished(method, route, status string, duration time.Duration) {
	if !c.Enabled() {
		return
	}
	if route == "" {
		route = RouteUnmatched
	}
	c.httpRequests.WithLabelValues(method, route, status).Inc()
	c.httpDuration.WithLabelValues(method, route, status).Observe(duration.Seconds())
	c.httpInFlight.Dec()
}

func (c *Component) RecordPanic() {
	if c.Enabled() {
		c.httpPanics.Inc()
	}
}

func (c *Component) RecordUserCacheOperation(operation, result string) {
	if c.Enabled() && validCacheResult(operation, result) {
		c.cacheOps.WithLabelValues(operation, result).Inc()
	}
}

func (c *Component) RecordLogin(result string) {
	if c.Enabled() && oneOf(result, LoginResultSuccess, LoginResultInvalidCredentials, LoginResultRateLimited, LoginResultError) {
		c.login.WithLabelValues(result).Inc()
	}
}

func (c *Component) RecordRefresh(result string) {
	if c.Enabled() && oneOf(result, RefreshResultSuccess, RefreshResultInvalid, RefreshResultRateLimited, RefreshResultError) {
		c.refresh.WithLabelValues(result).Inc()
	}
}

func (c *Component) RecordAuthenticationContext(result string) {
	if c.Enabled() && oneOf(result, ContextResultValid, ContextResultInvalid, ContextResultError) {
		c.authContext.WithLabelValues(result).Inc()
	}
}

func (c *Component) RecordAuthorization(result string) {
	if c.Enabled() && oneOf(result, AuthorizationResultAllowed, AuthorizationResultDenied, AuthorizationResultError) {
		c.authorization.WithLabelValues(result).Inc()
	}
}

func (c *Component) RecordRateLimitBlocked(policy string) {
	if c.Enabled() && oneOf(policy, RateLimitPolicyLoginIP, RateLimitPolicyLoginIdentifier, RateLimitPolicyRefreshIP) {
		c.rateLimit.WithLabelValues(policy).Inc()
	}
}

func (c *Component) RecordDatabaseTimeout(timeoutType string) {
	if c.Enabled() && oneOf(timeoutType, DatabaseTimeoutQuery, DatabaseTimeoutTransaction) {
		c.dbTimeouts.WithLabelValues(timeoutType).Inc()
	}
}

func validCacheResult(operation, result string) bool {
	switch operation {
	case CacheOperationRead:
		return oneOf(result, CacheResultHit, CacheResultMiss, CacheResultError, CacheResultInvalid)
	case CacheOperationWrite, CacheOperationInvalidate:
		return oneOf(result, CacheResultSuccess, CacheResultError)
	default:
		return false
	}
}

func oneOf(value string, allowed ...string) bool {
	for _, candidate := range allowed {
		if value == candidate {
			return true
		}
	}
	return false
}
