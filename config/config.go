package config

import (
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

const (
	// AccessTokenExpirationEnv controls the retained JWT duration, not opaque sessions.
	AccessTokenExpirationEnv  = "JWT_AUTH_EXPIRATION_TIME_IN_MINUTES"
	maxTokenDurationMinutes   = 525600
	maxRateLimit              = 1000000
	maxRateLimitWindowSeconds = 86400
	maxHTTPTimeoutSeconds     = 300
	minHeaderBytes            = 8 * 1024
	maxHeaderBytes            = 1024 * 1024
	minRequestBodyBytes       = 1024
	maxRequestBodyBytes       = 50 * 1024 * 1024
)

// Adjust like config.json
type Config struct {
	App struct {
		Name string `json:"name"`
		Env  string `json:"env"`
		Port int    `json:"port"`
	} `json:"app"`

	HTTP           HTTPServerConfig     `json:"http"`
	CORS           CORSConfig           `json:"cors"`
	Authentication AuthenticationConfig `json:"authentication"`

	Security struct {
		JWTAuthSecretKey                  string              `json:"JWTAuthKey"`
		JWTAuthExpirationTimeInMinutes    int                 `json:"JWTAuthExpirationTimeInMinutes"`
		JWTRefreshExpirationTimeInMinutes int                 `json:"JWTRefreshExpirationTimeInMinutes"`
		RefreshCookie                     RefreshCookieConfig `json:"refreshCookie"`
	} `json:"security"`

	Database DatabaseConfig `json:"database"`

	Redis RedisConfig `json:"redis"`

	Metrics MetricsConfig `json:"metrics"`

	RateLimiter struct {
		AuthLoginIPLimit                        int      `json:"authLoginIPLimit"`
		AuthLoginIPWindowSeconds                int      `json:"authLoginIPWindowSeconds"`
		AuthLoginIdentifierFailureLimit         int      `json:"authLoginIdentifierFailureLimit"`
		AuthLoginIdentifierFailureWindowSeconds int      `json:"authLoginIdentifierFailureWindowSeconds"`
		AuthRefreshIPLimit                      int      `json:"authRefreshIPLimit"`
		AuthRefreshIPWindowSeconds              int      `json:"authRefreshIPWindowSeconds"`
		MaxEntries                              int      `json:"maxEntries"`
		TrustedProxies                          []string `json:"trustedProxies"`
	} `json:"rateLimiter"`
}

// HTTPServerConfig defines process-wide HTTP boundary limits in seconds and bytes.
type HTTPServerConfig struct {
	ReadHeaderTimeoutSeconds int   `json:"readHeaderTimeoutSeconds"`
	ReadTimeoutSeconds       int   `json:"readTimeoutSeconds"`
	WriteTimeoutSeconds      int   `json:"writeTimeoutSeconds"`
	IdleTimeoutSeconds       int   `json:"idleTimeoutSeconds"`
	ShutdownTimeoutSeconds   int   `json:"shutdownTimeoutSeconds"`
	MaxHeaderBytes           int   `json:"maxHeaderBytes"`
	MaxRequestBodyBytes      int64 `json:"maxRequestBodyBytes"`
}

// CORSConfig contains the browser-origin policy applied at the HTTP boundary.
type CORSConfig struct {
	AllowedOrigins   []string `json:"allowedOrigins"`
	AllowCredentials bool     `json:"allowCredentials"`
}

// AuthenticationConfig controls the opaque session lifetime, independently of
// retained JWT settings (no JWT or refresh-token authentication is implemented).
type AuthenticationConfig struct {
	SessionLifetimeMinutes int `json:"sessionLifetimeMinutes"`
}

// RefreshCookieConfig contains the configurable attributes for the opaque
// Refresh Token transport. HttpOnly is intentionally not configurable.
type RefreshCookieConfig struct {
	HRISAdminName   string `json:"hrisAdminName"`
	HRISStudentName string `json:"hrisStudentName"`
	Secure          bool   `json:"secure"`
	SameSite        string `json:"sameSite"`
	Path            string `json:"path"`
}

type DatabaseConfig struct {
	Driver                         string `json:"driver"`
	Host                           string `json:"host"`
	Port                           int    `json:"port"`
	Name                           string `json:"name"`
	Username                       string `json:"username"`
	Password                       string `json:"password"`
	SSLMode                        string `json:"sslMode"`
	MaxOpenConnections             int    `json:"maxOpenConnections"`
	MaxIdleConnections             int    `json:"maxIdleConnections"`
	MaxLifetimeSeconds             int    `json:"maxLifetimeSeconds"`
	MaxIdleTime                    int    `json:"maxIdleTime"`
	QueryTimeoutSeconds            int    `json:"queryTimeoutSeconds"`
	TransactionTimeoutSeconds      int    `json:"transactionTimeoutSeconds"`
	SlowQueryThresholdMilliseconds int    `json:"slowQueryThresholdMilliseconds"`
	DBDebug                        bool   `json:"dbDebug"`
}

// RedisConfig defines the optional process-scoped Redis client and caches.
type RedisConfig struct {
	Enabled                               bool   `json:"enabled"`
	Address                               string `json:"address"`
	Username                              string `json:"username"`
	Password                              string `json:"password"`
	Database                              int    `json:"database"`
	KeyPrefix                             string `json:"keyPrefix"`
	DialTimeoutMilliseconds               int    `json:"dialTimeoutMilliseconds"`
	ReadTimeoutMilliseconds               int    `json:"readTimeoutMilliseconds"`
	WriteTimeoutMilliseconds              int    `json:"writeTimeoutMilliseconds"`
	UserCacheTTLSeconds                   int    `json:"userCacheTTLSeconds"`
	ExamSnapshotCacheTTLSeconds           int    `json:"examSnapshotCacheTTLSeconds"`
	ExamExecutionQuestionsCacheTTLSeconds int    `json:"examExecutionQuestionsCacheTTLSeconds"`
}

// MetricsConfig defines the optional Prometheus-compatible application endpoint.
type MetricsConfig struct {
	Enabled   bool   `json:"enabled"`
	Path      string `json:"path"`
	Namespace string `json:"namespace"`
}

var AppConfig Config

func LoadConfig() error {
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("load .env: invalid or unreadable environment file")
	}
	AppConfig = defaultConfig()

	// config.json is the required local configuration source. Defaults fill any
	// optional fields omitted from the file before explicit environment overrides.
	file, err := os.Open("config.json")
	if err != nil {
		return fmt.Errorf("open config.json: %w", err)
	}
	defer file.Close()

	if err := json.NewDecoder(file).Decode(&AppConfig); err != nil {
		return err
	}

	// Override application and security configuration from environment.
	if err := OverrideIntFromEnv("AUTH_SESSION_LIFETIME_MINUTES", &AppConfig.Authentication.SessionLifetimeMinutes); err != nil {
		return err
	}
	OverrideStringFromEnv("APP_ENV", &AppConfig.App.Env)
	OverrideStringFromEnv("JWT_AUTH_SECRET_KEY", &AppConfig.Security.JWTAuthSecretKey)
	if err := OverrideIntFromEnv(AccessTokenExpirationEnv, &AppConfig.Security.JWTAuthExpirationTimeInMinutes); err != nil {
		return err
	}
	OverrideStringFromEnv("AUTH_HRIS_ADMIN_REFRESH_COOKIE_NAME", &AppConfig.Security.RefreshCookie.HRISAdminName)
	OverrideStringFromEnv("AUTH_HRIS_STUDENT_REFRESH_COOKIE_NAME", &AppConfig.Security.RefreshCookie.HRISStudentName)
	if err := OverrideBoolFromEnvStrict("AUTH_REFRESH_COOKIE_SECURE", &AppConfig.Security.RefreshCookie.Secure); err != nil {
		return err
	}
	OverrideStringFromEnv("AUTH_REFRESH_COOKIE_SAMESITE", &AppConfig.Security.RefreshCookie.SameSite)
	OverrideStringFromEnv("AUTH_REFRESH_COOKIE_PATH", &AppConfig.Security.RefreshCookie.Path)
	if err := OverrideIntFromEnv("AUTH_LOGIN_IP_RATE_LIMIT", &AppConfig.RateLimiter.AuthLoginIPLimit); err != nil {
		return err
	}
	if err := OverrideIntFromEnv("AUTH_LOGIN_IP_RATE_WINDOW_SECONDS", &AppConfig.RateLimiter.AuthLoginIPWindowSeconds); err != nil {
		return err
	}
	if err := OverrideIntFromEnv("AUTH_LOGIN_IDENTIFIER_FAILURE_LIMIT", &AppConfig.RateLimiter.AuthLoginIdentifierFailureLimit); err != nil {
		return err
	}
	if err := OverrideIntFromEnv("AUTH_LOGIN_IDENTIFIER_FAILURE_WINDOW_SECONDS", &AppConfig.RateLimiter.AuthLoginIdentifierFailureWindowSeconds); err != nil {
		return err
	}
	if err := OverrideIntFromEnv("AUTH_REFRESH_IP_RATE_LIMIT", &AppConfig.RateLimiter.AuthRefreshIPLimit); err != nil {
		return err
	}
	if err := OverrideIntFromEnv("AUTH_REFRESH_IP_RATE_WINDOW_SECONDS", &AppConfig.RateLimiter.AuthRefreshIPWindowSeconds); err != nil {
		return err
	}
	if err := OverrideIntFromEnv("AUTH_RATE_LIMIT_MAX_ENTRIES", &AppConfig.RateLimiter.MaxEntries); err != nil {
		return err
	}
	OverrideStringSliceFromEnv("TRUSTED_PROXIES", &AppConfig.RateLimiter.TrustedProxies)
	if err := OverrideIntFromEnv("HTTP_READ_HEADER_TIMEOUT_SECONDS", &AppConfig.HTTP.ReadHeaderTimeoutSeconds); err != nil {
		return err
	}
	if err := OverrideIntFromEnv("HTTP_READ_TIMEOUT_SECONDS", &AppConfig.HTTP.ReadTimeoutSeconds); err != nil {
		return err
	}
	if err := OverrideIntFromEnv("HTTP_WRITE_TIMEOUT_SECONDS", &AppConfig.HTTP.WriteTimeoutSeconds); err != nil {
		return err
	}
	if err := OverrideIntFromEnv("HTTP_IDLE_TIMEOUT_SECONDS", &AppConfig.HTTP.IdleTimeoutSeconds); err != nil {
		return err
	}
	if err := OverrideIntFromEnv("HTTP_SHUTDOWN_TIMEOUT_SECONDS", &AppConfig.HTTP.ShutdownTimeoutSeconds); err != nil {
		return err
	}
	if err := OverrideIntFromEnv("HTTP_MAX_HEADER_BYTES", &AppConfig.HTTP.MaxHeaderBytes); err != nil {
		return err
	}
	if err := OverrideInt64FromEnv("HTTP_MAX_REQUEST_BODY_BYTES", &AppConfig.HTTP.MaxRequestBodyBytes); err != nil {
		return err
	}
	OverrideStringSliceFromEnv("CORS_ALLOWED_ORIGINS", &AppConfig.CORS.AllowedOrigins)
	if err := OverrideBoolFromEnvStrict("CORS_ALLOW_CREDENTIALS", &AppConfig.CORS.AllowCredentials); err != nil {
		return err
	}

	// Override Database From Env
	OverrideStringFromEnv("DB_DRIVER", &AppConfig.Database.Driver)
	OverrideStringFromEnv("DB_HOST", &AppConfig.Database.Host)
	if err := OverrideIntFromEnv("DB_PORT", &AppConfig.Database.Port); err != nil {
		return err
	}
	OverrideStringFromEnv("DB_SSL_MODE", &AppConfig.Database.SSLMode)
	OverrideStringFromEnv("DB_NAME", &AppConfig.Database.Name)
	OverrideStringFromEnv("DB_USERNAME", &AppConfig.Database.Username)
	OverrideStringFromEnv("DB_PASSWORD", &AppConfig.Database.Password)
	if err := OverrideIntFromEnv("DB_MAX_OPEN_CONNECTIONS", &AppConfig.Database.MaxOpenConnections); err != nil {
		return err
	}
	if err := OverrideIntFromEnv("DB_MAX_IDLE_CONNECTIONS", &AppConfig.Database.MaxIdleConnections); err != nil {
		return err
	}
	if err := OverrideIntFromEnv("DB_CONNECTION_MAX_LIFETIME_MINUTES", &AppConfig.Database.MaxLifetimeSeconds); err != nil {
		return err
	}
	if _, exists := os.LookupEnv("DB_CONNECTION_MAX_LIFETIME_MINUTES"); exists {
		AppConfig.Database.MaxLifetimeSeconds *= 60
	}
	if err := OverrideIntFromEnv("DB_CONNECTION_MAX_IDLE_TIME_MINUTES", &AppConfig.Database.MaxIdleTime); err != nil {
		return err
	}
	if _, exists := os.LookupEnv("DB_CONNECTION_MAX_IDLE_TIME_MINUTES"); exists {
		AppConfig.Database.MaxIdleTime *= 60
	}
	if err := OverrideIntFromEnv("DB_QUERY_TIMEOUT_SECONDS", &AppConfig.Database.QueryTimeoutSeconds); err != nil {
		return err
	}
	if err := OverrideIntFromEnv("DB_TRANSACTION_TIMEOUT_SECONDS", &AppConfig.Database.TransactionTimeoutSeconds); err != nil {
		return err
	}
	if err := OverrideIntFromEnv("DB_SLOW_QUERY_THRESHOLD_MILLISECONDS", &AppConfig.Database.SlowQueryThresholdMilliseconds); err != nil {
		return err
	}
	if err := OverrideBoolFromEnvStrict("DB_DEBUG", &AppConfig.Database.DBDebug); err != nil {
		return err
	}

	// Override Redis From Env
	if err := OverrideBoolFromEnvStrict("REDIS_ENABLED", &AppConfig.Redis.Enabled); err != nil {
		return err
	}
	OverrideStringFromEnv("REDIS_ADDRESS", &AppConfig.Redis.Address)
	OverrideStringFromEnv("REDIS_USERNAME", &AppConfig.Redis.Username)
	OverrideStringFromEnv("REDIS_PASSWORD", &AppConfig.Redis.Password)
	if err := OverrideIntFromEnv("REDIS_DATABASE", &AppConfig.Redis.Database); err != nil {
		return err
	}
	OverrideStringFromEnv("REDIS_KEY_PREFIX", &AppConfig.Redis.KeyPrefix)
	if err := OverrideIntFromEnv("REDIS_DIAL_TIMEOUT_MILLISECONDS", &AppConfig.Redis.DialTimeoutMilliseconds); err != nil {
		return err
	}
	if err := OverrideIntFromEnv("REDIS_READ_TIMEOUT_MILLISECONDS", &AppConfig.Redis.ReadTimeoutMilliseconds); err != nil {
		return err
	}
	if err := OverrideIntFromEnv("REDIS_WRITE_TIMEOUT_MILLISECONDS", &AppConfig.Redis.WriteTimeoutMilliseconds); err != nil {
		return err
	}
	if err := OverrideIntFromEnv("REDIS_USER_CACHE_TTL_SECONDS", &AppConfig.Redis.UserCacheTTLSeconds); err != nil {
		return err
	}
	if err := OverrideIntFromEnv("REDIS_EXAM_SNAPSHOT_CACHE_TTL_SECONDS", &AppConfig.Redis.ExamSnapshotCacheTTLSeconds); err != nil {
		return err
	}
	if err := OverrideIntFromEnv("REDIS_EXAM_EXECUTION_QUESTIONS_CACHE_TTL_SECONDS", &AppConfig.Redis.ExamExecutionQuestionsCacheTTLSeconds); err != nil {
		return err
	}

	// Override Metrics From Env
	if err := OverrideBoolFromEnvStrict("METRICS_ENABLED", &AppConfig.Metrics.Enabled); err != nil {
		return err
	}
	OverrideStringFromEnv("METRICS_PATH", &AppConfig.Metrics.Path)
	OverrideStringFromEnv("METRICS_NAMESPACE", &AppConfig.Metrics.Namespace)
	// OverrideStringFromEnv("SUPABASE_URL", &AppConfig.Supabase.URL)
	// OverrideStringFromEnv("SUPABASE_SECRET_KEY", &AppConfig.Supabase.SecretKey)
	// OverrideStringFromEnv("SUPABASE_STORAGE_BUCKET", &AppConfig.Supabase.StorageBucket)

	return Validate()
}

// Validate confirms token durations before services are initialized.
func Validate() error {
	if AppConfig.Authentication.SessionLifetimeMinutes < 1 || AppConfig.Authentication.SessionLifetimeMinutes > 1440 {
		return fmt.Errorf("authentication.sessionLifetimeMinutes must be between 1 and 1440")
	}
	if err := validateAppConfig(AppConfig.App.Port); err != nil {
		return err
	}
	if err := validateSecurityConfig(AppConfig.Security.JWTAuthSecretKey); err != nil {
		return err
	}
	if err := validateTokenDuration("JWTAuthExpirationTimeInMinutes", AppConfig.Security.JWTAuthExpirationTimeInMinutes); err != nil {
		return err
	}

	if err := validateTokenDuration("JWTRefreshExpirationTimeInMinutes", AppConfig.Security.JWTRefreshExpirationTimeInMinutes); err != nil {
		return err
	}
	if err := validateRefreshCookieConfig(AppConfig.App.Env, &AppConfig.Security.RefreshCookie); err != nil {
		return err
	}

	for _, setting := range []struct {
		name  string
		value int
	}{
		{"rateLimiter.authLoginIPLimit", AppConfig.RateLimiter.AuthLoginIPLimit},
		{"rateLimiter.authLoginIdentifierFailureLimit", AppConfig.RateLimiter.AuthLoginIdentifierFailureLimit},
		{"rateLimiter.authRefreshIPLimit", AppConfig.RateLimiter.AuthRefreshIPLimit},
		{"rateLimiter.maxEntries", AppConfig.RateLimiter.MaxEntries},
	} {
		if err := validateRateLimit(setting.name, setting.value); err != nil {
			return err
		}
	}

	for _, setting := range []struct {
		name  string
		value int
	}{
		{"rateLimiter.authLoginIPWindowSeconds", AppConfig.RateLimiter.AuthLoginIPWindowSeconds},
		{"rateLimiter.authLoginIdentifierFailureWindowSeconds", AppConfig.RateLimiter.AuthLoginIdentifierFailureWindowSeconds},
		{"rateLimiter.authRefreshIPWindowSeconds", AppConfig.RateLimiter.AuthRefreshIPWindowSeconds},
	} {
		if setting.value <= 0 || setting.value > maxRateLimitWindowSeconds {
			return fmt.Errorf("%s must be between 1 and %d seconds", setting.name, maxRateLimitWindowSeconds)
		}
	}

	if AppConfig.RateLimiter.MaxEntries < 2 {
		return fmt.Errorf("rateLimiter.maxEntries must be at least 2 for IP and identifier records")
	}

	if err := validateTrustedProxies(AppConfig.RateLimiter.TrustedProxies); err != nil {
		return err
	}
	if err := validateHTTPServerConfig(AppConfig.HTTP); err != nil {
		return err
	}
	if err := validateDatabaseConfig(AppConfig.Database); err != nil {
		return err
	}
	if err := validateRedisConfig(AppConfig.Redis); err != nil {
		return err
	}
	if err := validateMetricsConfig(AppConfig.Metrics); err != nil {
		return err
	}
	// if err := AppConfig.Supabase.Validate(); err != nil {
	// 	return err
	// }
	return validateCORSConfig(&AppConfig.CORS)
}

func validateAppConfig(port int) error {
	if port < 1 || port > 65535 {
		return fmt.Errorf("app.port must be between 1 and 65535")
	}
	return nil
}

func validateSecurityConfig(accessSecret string) error {
	if strings.TrimSpace(accessSecret) == "" {
		return fmt.Errorf("security.JWTAuthKey is required")
	}
	return nil
}

var refreshCookieNamePattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

func validateRefreshCookieConfig(appEnv string, cfg *RefreshCookieConfig) error {
	if cfg == nil {
		return fmt.Errorf("security.refreshCookie is required")
	}
	for name, value := range map[string]string{
		"hrisAdminName":   cfg.HRISAdminName,
		"hrisStudentName": cfg.HRISStudentName,
	} {
		if !refreshCookieNamePattern.MatchString(value) {
			return fmt.Errorf("security.refreshCookie.%s must contain only letters, digits, underscores, or hyphens", name)
		}
	}
	if cfg.HRISAdminName == cfg.HRISStudentName {
		return fmt.Errorf("security.refreshCookie client cookie names must be distinct")
	}

	switch strings.ToLower(strings.TrimSpace(cfg.SameSite)) {
	case "lax":
		cfg.SameSite = "Lax"
	case "strict":
		cfg.SameSite = "Strict"
	case "none":
		cfg.SameSite = "None"
	default:
		return fmt.Errorf("security.refreshCookie.sameSite must be Lax, Strict, or None")
	}
	if cfg.SameSite == "None" && !cfg.Secure {
		return fmt.Errorf("security.refreshCookie.sameSite None requires security.refreshCookie.secure=true")
	}
	if strings.EqualFold(strings.TrimSpace(appEnv), "production") && !cfg.Secure {
		return fmt.Errorf("security.refreshCookie.secure must be true in production")
	}

	if cfg.Path == "" || strings.TrimSpace(cfg.Path) != cfg.Path || !strings.HasPrefix(cfg.Path, "/") || strings.ContainsAny(cfg.Path, "?#;\\") || strings.Contains(cfg.Path, "//") {
		return fmt.Errorf("security.refreshCookie.path must be a normalized absolute cookie path")
	}
	return nil
}

var prometheusNamespacePattern = regexp.MustCompile(`^[a-zA-Z_:][a-zA-Z0-9_:]*$`)

func validateMetricsConfig(cfg MetricsConfig) error {
	if !cfg.Enabled {
		return nil
	}

	path := strings.TrimSpace(cfg.Path)
	if path == "" {
		return fmt.Errorf("metrics.path must not be empty when metrics are enabled")
	}
	if path != cfg.Path {
		return fmt.Errorf("metrics.path must not contain surrounding whitespace")
	}
	if !strings.HasPrefix(path, "/") {
		return fmt.Errorf("metrics.path must start with /")
	}
	if strings.ContainsAny(path, "?#") {
		return fmt.Errorf("metrics.path must not contain a query string or fragment")
	}
	if strings.ContainsAny(path, ":*") {
		return fmt.Errorf("metrics.path must be a static path without route parameters")
	}
	if path == "/v1" || strings.HasPrefix(path, "/v1/") {
		return fmt.Errorf("metrics.path must not be placed under /v1")
	}
	if path == "/" || path == "/health" || path == "/ready" || path == "/docs" || strings.HasPrefix(path, "/docs/") || path == "/swagger" || strings.HasPrefix(path, "/swagger/") {
		return fmt.Errorf("metrics.path conflicts with an existing route")
	}

	namespace := strings.TrimSpace(cfg.Namespace)
	if namespace == "" {
		return fmt.Errorf("metrics.namespace must not be empty when metrics are enabled")
	}
	if namespace != cfg.Namespace {
		return fmt.Errorf("metrics.namespace must not contain surrounding whitespace")
	}
	if !prometheusNamespacePattern.MatchString(namespace) {
		return fmt.Errorf("metrics.namespace must be a valid Prometheus metric namespace")
	}
	return nil
}

func validateRedisConfig(cfg RedisConfig) error {
	if !cfg.Enabled {
		return nil
	}
	if strings.TrimSpace(cfg.Address) == "" {
		return fmt.Errorf("redis.address must not be empty when Redis is enabled")
	}
	if cfg.Database < 0 {
		return fmt.Errorf("redis.database must be non-negative")
	}
	if strings.Trim(strings.TrimSpace(cfg.KeyPrefix), ":") == "" {
		return fmt.Errorf("redis.keyPrefix must not be empty when Redis is enabled")
	}
	for _, setting := range []struct {
		name  string
		value int
	}{
		{"redis.dialTimeoutMilliseconds", cfg.DialTimeoutMilliseconds},
		{"redis.readTimeoutMilliseconds", cfg.ReadTimeoutMilliseconds},
		{"redis.writeTimeoutMilliseconds", cfg.WriteTimeoutMilliseconds},
		{"redis.userCacheTTLSeconds", cfg.UserCacheTTLSeconds},
		{"redis.examSnapshotCacheTTLSeconds", cfg.ExamSnapshotCacheTTLSeconds},
		{"redis.examExecutionQuestionsCacheTTLSeconds", cfg.ExamExecutionQuestionsCacheTTLSeconds},
	} {
		if setting.value <= 0 {
			return fmt.Errorf("%s must be positive", setting.name)
		}
	}
	return nil
}

func validateDatabaseConfig(cfg DatabaseConfig) error {
	if cfg.Driver != "postgres" {
		return fmt.Errorf("database.driver must be postgres")
	}
	if strings.TrimSpace(cfg.Host) == "" {
		return fmt.Errorf("database.host is required")
	}
	if cfg.Port < 1 || cfg.Port > 65535 {
		return fmt.Errorf("database.port must be between 1 and 65535")
	}
	if strings.TrimSpace(cfg.Name) == "" {
		return fmt.Errorf("database.name is required")
	}
	if strings.TrimSpace(cfg.Username) == "" {
		return fmt.Errorf("database.username is required")
	}
	if cfg.Password == "" {
		return fmt.Errorf("database.password is required")
	}
	switch cfg.SSLMode {
	case "disable", "allow", "prefer", "require", "verify-ca", "verify-full":
	default:
		return fmt.Errorf("database.sslMode must be disable, allow, prefer, require, verify-ca or verify-full")
	}
	if cfg.MaxOpenConnections < 1 || cfg.MaxOpenConnections > 1000 {
		return fmt.Errorf("database.maxOpenConnections must be between 1 and 1000")
	}
	if cfg.MaxIdleConnections < 0 || cfg.MaxIdleConnections > cfg.MaxOpenConnections {
		return fmt.Errorf("database.maxIdleConnections must be between 0 and database.maxOpenConnections")
	}
	const maxConnectionDurationSeconds = 24 * 60 * 60
	if cfg.MaxLifetimeSeconds < 1 || cfg.MaxLifetimeSeconds > maxConnectionDurationSeconds {
		return fmt.Errorf("database.maxLifetimeSeconds must be between 1 and %d", maxConnectionDurationSeconds)
	}
	if cfg.MaxIdleTime < 1 || cfg.MaxIdleTime > cfg.MaxLifetimeSeconds {
		return fmt.Errorf("database.maxIdleTime must be between 1 and database.maxLifetimeSeconds")
	}
	if cfg.QueryTimeoutSeconds < 1 || cfg.QueryTimeoutSeconds > 60 {
		return fmt.Errorf("database.queryTimeoutSeconds must be between 1 and 60")
	}
	if cfg.TransactionTimeoutSeconds < 1 || cfg.TransactionTimeoutSeconds > 120 {
		return fmt.Errorf("database.transactionTimeoutSeconds must be between 1 and 120")
	}
	if cfg.SlowQueryThresholdMilliseconds < 1 || cfg.SlowQueryThresholdMilliseconds > 60000 {
		return fmt.Errorf("database.slowQueryThresholdMilliseconds must be between 1 and 60000")
	}
	return nil
}

func validateHTTPServerConfig(cfg HTTPServerConfig) error {
	for _, setting := range []struct {
		name  string
		value int
	}{
		{"http.readHeaderTimeoutSeconds", cfg.ReadHeaderTimeoutSeconds},
		{"http.readTimeoutSeconds", cfg.ReadTimeoutSeconds},
		{"http.writeTimeoutSeconds", cfg.WriteTimeoutSeconds},
		{"http.idleTimeoutSeconds", cfg.IdleTimeoutSeconds},
		{"http.shutdownTimeoutSeconds", cfg.ShutdownTimeoutSeconds},
	} {
		if setting.value <= 0 || setting.value > maxHTTPTimeoutSeconds {
			return fmt.Errorf("%s must be between 1 and %d seconds", setting.name, maxHTTPTimeoutSeconds)
		}
	}
	if cfg.WriteTimeoutSeconds <= 2 {
		return fmt.Errorf("http.writeTimeoutSeconds must exceed the 2 second readiness timeout")
	}
	if cfg.MaxHeaderBytes < minHeaderBytes || cfg.MaxHeaderBytes > maxHeaderBytes {
		return fmt.Errorf("http.maxHeaderBytes must be between %d and %d bytes", minHeaderBytes, maxHeaderBytes)
	}
	if cfg.MaxRequestBodyBytes < minRequestBodyBytes || cfg.MaxRequestBodyBytes > maxRequestBodyBytes {
		return fmt.Errorf("http.maxRequestBodyBytes must be between %d and %d bytes", minRequestBodyBytes, maxRequestBodyBytes)
	}
	return nil
}

func validateCORSConfig(cfg *CORSConfig) error {
	if cfg.AllowCredentials && len(cfg.AllowedOrigins) == 0 {
		return fmt.Errorf("cors.allowCredentials requires at least one explicit allowed origin")
	}
	seen := make(map[string]struct{}, len(cfg.AllowedOrigins))
	normalized := make([]string, 0, len(cfg.AllowedOrigins))
	for _, configuredOrigin := range cfg.AllowedOrigins {
		origin := strings.TrimSpace(configuredOrigin)
		if origin == "" {
			return fmt.Errorf("cors.allowedOrigins must not contain empty origins")
		}
		if origin == "*" {
			if cfg.AllowCredentials {
				return fmt.Errorf("cors.allowedOrigins cannot contain wildcard when cors.allowCredentials is true")
			}
			if len(cfg.AllowedOrigins) != 1 {
				return fmt.Errorf("cors.allowedOrigins wildcard must be the only configured origin")
			}
			normalized = append(normalized, origin)
			continue
		}
		if !isValidCORSOrigin(origin) {
			return fmt.Errorf("cors.allowedOrigins contains an invalid origin")
		}
		if _, exists := seen[origin]; !exists {
			seen[origin] = struct{}{}
			normalized = append(normalized, origin)
		}
	}
	cfg.AllowedOrigins = normalized
	return nil
}

func isValidCORSOrigin(origin string) bool {
	parsed, err := url.Parse(origin)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" || parsed.User != nil || parsed.Path != "" || parsed.RawQuery != "" || parsed.Fragment != "" || parsed.Hostname() == "" {
		return false
	}

	if strings.HasPrefix(parsed.Host, "[") {
		if strings.HasSuffix(parsed.Host, "]") {
			return net.ParseIP(strings.Trim(parsed.Host, "[]")) != nil
		}
		_, port, err := net.SplitHostPort(parsed.Host)
		return err == nil && validPort(port)
	}
	if strings.Contains(parsed.Host, ":") {
		_, port, err := net.SplitHostPort(parsed.Host)
		return err == nil && validPort(port)
	}
	return true
}

func validPort(port string) bool {
	parsed, err := strconv.ParseUint(port, 10, 16)
	return err == nil && parsed <= 65535
}

func validateTokenDuration(name string, minutes int) error {
	if minutes <= 0 || minutes > maxTokenDurationMinutes {
		return fmt.Errorf("%s must be between 1 and %d minutes", name, maxTokenDurationMinutes)
	}

	return nil
}

func validateRateLimit(name string, value int) error {
	if value <= 0 || value > maxRateLimit {
		return fmt.Errorf("%s must be between 1 and %d", name, maxRateLimit)
	}
	return nil
}

func validateTrustedProxies(proxies []string) error {
	for _, proxy := range proxies {
		if net.ParseIP(proxy) != nil {
			continue
		}
		if _, _, err := net.ParseCIDR(proxy); err == nil {
			continue
		}
		return fmt.Errorf("trusted proxy %q must be a valid IP address or CIDR", proxy)
	}
	return nil
}

func defaultConfig() Config {
	var cfg Config
	cfg.App.Name = "hris-core-service"
	cfg.HTTP.ReadHeaderTimeoutSeconds = 5
	cfg.HTTP.ReadTimeoutSeconds = 15
	cfg.HTTP.WriteTimeoutSeconds = 30
	cfg.HTTP.IdleTimeoutSeconds = 60
	cfg.HTTP.ShutdownTimeoutSeconds = 15
	cfg.HTTP.MaxHeaderBytes = 65536
	cfg.HTTP.MaxRequestBodyBytes = 1048576
	cfg.CORS.AllowedOrigins = []string{}
	cfg.CORS.AllowCredentials = false
	cfg.Authentication.SessionLifetimeMinutes = 15
	cfg.Security.RefreshCookie.HRISAdminName = "hris_admin_refresh_token"
	cfg.Security.RefreshCookie.HRISStudentName = "hris_student_refresh_token"
	cfg.Security.RefreshCookie.Secure = false
	cfg.Security.RefreshCookie.SameSite = "Lax"
	cfg.Security.RefreshCookie.Path = "/v1/auth"
	cfg.RateLimiter.AuthLoginIPLimit = 10
	cfg.RateLimiter.AuthLoginIPWindowSeconds = 60
	cfg.RateLimiter.AuthLoginIdentifierFailureLimit = 5
	cfg.RateLimiter.AuthLoginIdentifierFailureWindowSeconds = 300
	cfg.RateLimiter.AuthRefreshIPLimit = 30
	cfg.RateLimiter.AuthRefreshIPWindowSeconds = 60
	cfg.RateLimiter.MaxEntries = 100000
	cfg.RateLimiter.TrustedProxies = []string{}
	cfg.Database.MaxOpenConnections = 25
	cfg.Database.MaxIdleConnections = 10
	cfg.Database.MaxLifetimeSeconds = 30 * 60
	cfg.Database.MaxIdleTime = 5 * 60
	cfg.Database.QueryTimeoutSeconds = 5
	cfg.Database.TransactionTimeoutSeconds = 10
	cfg.Database.SlowQueryThresholdMilliseconds = 500
	cfg.Redis.Enabled = false
	cfg.Redis.Address = "127.0.0.1:6379"
	cfg.Redis.Database = 0
	cfg.Redis.KeyPrefix = "hris_core_service:v1"
	cfg.Redis.DialTimeoutMilliseconds = 1000
	cfg.Redis.ReadTimeoutMilliseconds = 500
	cfg.Redis.WriteTimeoutMilliseconds = 500
	cfg.Redis.UserCacheTTLSeconds = 300
	cfg.Redis.ExamSnapshotCacheTTLSeconds = 6 * 60 * 60
	cfg.Redis.ExamExecutionQuestionsCacheTTLSeconds = 6 * 60 * 60
	cfg.Metrics.Enabled = false
	cfg.Metrics.Path = "/metrics"
	cfg.Metrics.Namespace = "hris_core_service"
	return cfg
}

// OverrideStringFromEnv applies an explicitly configured string, including an
// intentionally blank value. It is used where blank has defined semantics or
// must be rejected by validation.
func OverrideStringFromEnv(envKey string, configField *string) {
	if envValue, exists := os.LookupEnv(envKey); exists {
		*configField = envValue
	}
}

func OverrideIntFromEnv(envKey string, configField *int) error {
	envValue, exists := os.LookupEnv(envKey)
	if !exists {
		return nil
	}

	parsed, err := strconv.Atoi(envValue)
	if err != nil {
		return fmt.Errorf("%s must be a valid integer", envKey)
	}

	*configField = parsed
	return nil
}

func OverrideInt64FromEnv(envKey string, configField *int64) error {
	envValue, exists := os.LookupEnv(envKey)
	if !exists {
		return nil
	}

	parsed, err := strconv.ParseInt(envValue, 10, 64)
	if err != nil {
		return fmt.Errorf("%s must be a valid integer", envKey)
	}
	*configField = parsed
	return nil
}

func OverrideStringSliceFromEnv(envKey string, configField *[]string) {
	envValue, exists := os.LookupEnv(envKey)
	if !exists {
		return
	}
	if strings.TrimSpace(envValue) == "" {
		*configField = []string{}
		return
	}

	values := strings.Split(envValue, ",")
	proxies := make([]string, 0, len(values))
	for _, value := range values {
		proxies = append(proxies, strings.TrimSpace(value))
	}
	*configField = proxies
}

func OverrideBoolFromEnvStrict(envKey string, configField *bool) error {
	envValue, exists := os.LookupEnv(envKey)
	if !exists {
		return nil
	}
	parsed, err := strconv.ParseBool(envValue)
	if err != nil {
		return fmt.Errorf("%s must be a valid boolean", envKey)
	}
	*configField = parsed
	return nil
}
