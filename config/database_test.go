package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPostgresConfigurationPrecedenceAndValidation(t *testing.T) {
	content, err := os.ReadFile("../config.example.json")
	if err != nil {
		t.Fatal(err)
	}
	fixture := defaultConfig()
	if err := json.Unmarshal(content, &fixture); err != nil {
		t.Fatal(err)
	}
	fixture.Security.JWTAuthSecretKey = "test-only-validator-secret"
	fixture.Database.Password = "test-only-json-password"
	content, err = json.Marshal(fixture)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "config.json"), content, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte("DB_HOST=dotenv-host\nDB_PORT=5433\nDB_NAME=dotenv_database\nDB_USERNAME=dotenv_user\nDB_PASSWORD=dotenv_password\nDB_SSL_MODE=disable\nDB_MAX_OPEN_CONNECTIONS=19\nPOSTGRES_PORT=5999\n"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)
	old := AppConfig
	t.Cleanup(func() { AppConfig = old })
	for _, key := range []string{"DB_DRIVER", "DB_HOST", "DB_PORT", "DB_NAME", "DB_USERNAME", "DB_PASSWORD", "DB_SSL_MODE", "DB_MAX_OPEN_CONNECTIONS", "POSTGRES_PORT"} {
		t.Setenv(key, "")
		if err := os.Unsetenv(key); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("DB_HOST", "process-host")
	if err := LoadConfig(); err != nil {
		t.Fatal(err)
	}
	if AppConfig.Database.Host != "process-host" || AppConfig.Database.Port != 5433 || AppConfig.Database.Name != "dotenv_database" || AppConfig.Database.Password != "dotenv_password" || AppConfig.Database.Driver != "postgres" || AppConfig.Database.MaxOpenConnections != 19 || AppConfig.Database.QueryTimeoutSeconds != 5 {
		t.Fatal("process > dotenv > JSON > defaults or POSTGRES isolation violated")
	}
	valid := AppConfig.Database
	for _, sslMode := range []string{"disable", "allow", "prefer", "require", "verify-ca", "verify-full"} {
		cfg := valid
		cfg.SSLMode = sslMode
		if err := validateDatabaseConfig(cfg); err != nil {
			t.Fatalf("valid SSL mode rejected: %s", sslMode)
		}
	}
	for _, tc := range []struct {
		name   string
		change func(*DatabaseConfig)
	}{
		{"mysql", func(c *DatabaseConfig) { c.Driver = "mysql" }},
		{"empty_host", func(c *DatabaseConfig) { c.Host = " " }},
		{"empty_database", func(c *DatabaseConfig) { c.Name = "" }},
		{"empty_user", func(c *DatabaseConfig) { c.Username = "" }},
		{"empty_password", func(c *DatabaseConfig) { c.Password = "" }},
		{"zero_port", func(c *DatabaseConfig) { c.Port = 0 }},
		{"oversize_port", func(c *DatabaseConfig) { c.Port = 65536 }},
		{"empty_ssl", func(c *DatabaseConfig) { c.SSLMode = "" }},
		{"invalid_ssl", func(c *DatabaseConfig) { c.SSLMode = "unsafe" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := valid
			tc.change(&cfg)
			if validateDatabaseConfig(cfg) == nil {
				t.Fatal("invalid config accepted")
			}
		})
	}
	t.Setenv("DB_PORT", "invalid-secret-sentinel")
	if err := LoadConfig(); err == nil || strings.Contains(err.Error(), "invalid-secret-sentinel") {
		t.Fatal("invalid port must fail without its value")
	}
	t.Setenv("DB_PORT", "5433")
	t.Setenv("DB_PASSWORD", "")
	if err := LoadConfig(); err == nil {
		t.Fatal("explicit empty process password must not fall back")
	}
}
