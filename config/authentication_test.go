package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"encoding/json"
)

func TestAuthenticationLifetimeConfiguration(t *testing.T) {
	previous := AppConfig
	t.Cleanup(func() { AppConfig = previous })
	dir := t.TempDir()
	t.Chdir(dir)
	fixture := defaultConfig()
	fixture.App.Port = 8080
	fixture.Security.JWTAuthSecretKey = "test-only-secret"
	fixture.Security.JWTAuthExpirationTimeInMinutes = 15
	fixture.Security.JWTRefreshExpirationTimeInMinutes = 10080
	fixture.Database = DatabaseConfig{Driver: "postgres", Host: "127.0.0.1", Port: 5432, Name: "test", Username: "test", Password: "test-only", SSLMode: "disable", MaxOpenConnections: 2, MaxIdleConnections: 1, MaxLifetimeSeconds: 1800, MaxIdleTime: 300, QueryTimeoutSeconds: 5, TransactionTimeoutSeconds: 10, SlowQueryThresholdMilliseconds: 500}
	// Clear only supported overrides that may alter this fixture; no real config is read.
	for _, entry := range os.Environ() {
		key, _, _ := strings.Cut(entry, "=")
		if strings.HasPrefix(key, "DB_") || strings.HasPrefix(key, "AUTH_") || key == "APP_ENV" || key == "JWT_AUTH_SECRET_KEY" || key == AccessTokenExpirationEnv {
			t.Setenv(key, os.Getenv(key))
			_ = os.Unsetenv(key)
		}
	}
	write := func() {
		data, _ := json.Marshal(fixture)
		if os.WriteFile(filepath.Join(dir, "config.json"), data, 0600) != nil {
			t.Fatal("fixture write failed")
		}
	}
	write()
	if err := LoadConfig(); err != nil || AppConfig.Authentication.SessionLifetimeMinutes != 15 {
		t.Fatalf("default lifetime failed: %v", err)
	}
	fixture.Authentication.SessionLifetimeMinutes = 30
	write()
	if LoadConfig() != nil || AppConfig.Authentication.SessionLifetimeMinutes != 30 {
		t.Fatal("JSON lifetime failed")
	}
	t.Setenv("AUTH_SESSION_LIFETIME_MINUTES", "45")
	if LoadConfig() != nil || AppConfig.Authentication.SessionLifetimeMinutes != 45 {
		t.Fatal("environment precedence failed")
	}
	for _, value := range []string{"", "0", "-1", "1441", "bad"} {
		t.Setenv("AUTH_SESSION_LIFETIME_MINUTES", value)
		if LoadConfig() == nil {
			t.Fatal("invalid lifetime accepted")
		}
	}
}
