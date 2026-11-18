package config

import "testing"

func TestMetricsPathDoesNotCollideWithAPIDocs(t *testing.T) {
	for _, path := range []string{"/docs", "/docs/", "/docs/nested", "/swagger/doc.json"} {
		if err := validateMetricsConfig(MetricsConfig{Enabled: true, Path: path, Namespace: "hris"}); err == nil {
			t.Fatalf("metrics path %q must not collide with API documentation", path)
		}
	}
	if err := validateMetricsConfig(MetricsConfig{Enabled: true, Path: "/metrics", Namespace: "hris"}); err != nil {
		t.Fatalf("existing metrics path must remain valid: %v", err)
	}
}

func TestSecurityConfigurationGuards(t *testing.T) {
	for _, cfg := range []CORSConfig{
		{AllowedOrigins: []string{"*"}, AllowCredentials: true},
		{AllowedOrigins: []string{"https://client.example.test/path"}},
		{AllowedOrigins: []string{"https://client.example.test", "*"}},
	} {
		if validateCORSConfig(&cfg) == nil {
			t.Fatal("unsafe CORS accepted")
		}
	}
	for _, proxies := range [][]string{{"any"}, {"https://proxy.example.test"}, {"127.0.0.1/999"}} {
		if validateTrustedProxies(proxies) == nil {
			t.Fatal("invalid proxy accepted")
		}
	}
	if validateTrustedProxies([]string{"192.0.2.0/24", "::1"}) != nil {
		t.Fatal("valid proxy rejected")
	}
	previous := AppConfig
	t.Cleanup(func() { AppConfig = previous })
	AppConfig = defaultConfig()
	AppConfig.RateLimiter.MaxEntries = 1
	AppConfig.App.Port = 8080
	AppConfig.Security.JWTAuthSecretKey = "fixture-only"
	AppConfig.Security.JWTAuthExpirationTimeInMinutes = 15
	AppConfig.Security.JWTRefreshExpirationTimeInMinutes = 15
	// Validation before DB settings still rejects an unusable limiter capacity.
	if err := Validate(); err == nil || err.Error() != "rateLimiter.maxEntries must be at least 2 for IP and identifier records" {
		t.Fatalf("unusable limiter capacity guard: %v", err)
	}
	cfg := defaultConfig().HTTP
	if validateHTTPServerConfig(cfg) != nil {
		t.Fatal("valid timeout defaults rejected")
	}
	cfg.ReadHeaderTimeoutSeconds = 0
	if validateHTTPServerConfig(cfg) == nil {
		t.Fatal("unbounded header timeout accepted")
	}
}
