package infrastructure

import (
	"context"
	"errors"
	"fmt"
	"syscall"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"hris-core-service/config"
)

func TestPostgresConnectionEscapingAndExplicitSettings(t *testing.T) {
	for _, key := range []string{"PGHOST", "PGPORT", "PGUSER", "PGPASSWORD", "PGDATABASE", "PGSSLMODE", "PGCONNECT_TIMEOUT", "PGAPPNAME", "PGTARGETSESSIONATTRS"} {
		t.Setenv(key, "ignored-invalid-value")
	}
	t.Setenv("PGSERVICE", "")
	for _, host := range []string{"127.0.0.1", "::1"} {
		cfg := config.DatabaseConfig{Host: host, Port: 5433, Name: "database /?#%", Username: "user :@/'\\", Password: "password :@/'\\?%=&+", SSLMode: "disable", QueryTimeoutSeconds: 3}
		parsed, err := postgresConnectionConfig(cfg)
		if err != nil {
			t.Fatal("escaped configuration rejected")
		}
		if parsed.Host != host || parsed.Port != 5433 || parsed.Database != cfg.Name || parsed.User != cfg.Username || parsed.Password != cfg.Password || parsed.TLSConfig != nil || parsed.ConnectTimeout != 3*time.Second || parsed.RuntimeParams["timezone"] != "UTC" || parsed.RuntimeParams["application_name"] != "hris-core-service" {
			t.Fatal("explicit connection fields changed during parsing")
		}
	}
	t.Setenv("PGSERVICE", "unused-service")
	if _, err := postgresConnectionConfig(config.DatabaseConfig{}); err == nil {
		t.Fatal("service identity override accepted")
	}
}

func TestDatabaseStartupDiagnosticsDoNotExposeDriverDetails(t *testing.T) {
	for _, tc := range []struct {
		err    error
		reason string
	}{
		{context.DeadlineExceeded, "timeout"}, {context.Canceled, "canceled"},
		{&pgconn.PgError{Code: "28P01", Message: "secret-sentinel"}, "authentication failed"},
		{&pgconn.PgError{Code: "3D000", Message: "secret-sentinel"}, "database not found"},
		{&pgconn.PgError{Code: "08006", Message: "secret-sentinel"}, "database server rejected connection"},
		{fmt.Errorf("secret-sentinel: %w", syscall.ECONNREFUSED), "connection refused"},
		{errors.New("secret-sentinel"), "connection failed"},
	} {
		if got := databaseStartupError(tc.err).Error(); got != "database startup connectivity check failed: "+tc.reason {
			t.Fatal("unexpected or unsafe diagnostic")
		}
	}
}
