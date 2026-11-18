package migration

import (
	"context"
	"errors"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"hris-core-service/config"
)

func TestMigrationURLPreservesConfiguredIdentity(t *testing.T) {
	t.Setenv("PGSERVICE", "")
	for _, host := range []string{"127.0.0.1", "::1"} {
		cfg := config.DatabaseConfig{Host: host, Port: 5433, Name: "database /?#%", Username: "role :@/'\\", Password: "secret :@/'\\?%=&+", SSLMode: "disable", QueryTimeoutSeconds: 3}
		c, err := pgx.ParseConfig(migrationDatabaseURL(cfg))
		if err != nil || c.Host != host || c.Port != 5433 || c.User != cfg.Username || c.Password != cfg.Password || c.Database != cfg.Name || c.ConnectTimeout.Seconds() != 3 || c.TLSConfig != nil {
			t.Fatal("migration URL changed configured identity")
		}
	}
	for _, mode := range []string{"disable", "allow", "prefer", "require", "verify-ca", "verify-full"} {
		u, err := url.Parse(migrationDatabaseURL(config.DatabaseConfig{SSLMode: mode}))
		if err != nil || u.Query().Get("sslmode") != mode {
			t.Fatal("migration SSL mode must come from backend configuration")
		}
	}
	path := filepath.Join(t.TempDir(), "migration source # ? %")
	if err := os.Mkdir(path, 0700); err != nil {
		t.Fatal(err)
	}
	if _, err := NewRunner(Config{SourceURL: migrationSourceURL(path), DatabaseURL: "invalid"}); err == nil || !strings.Contains(err.Error(), "PostgreSQL configuration") {
		t.Fatal("encoded file source should open before invalid database configuration")
	}
}

func TestMigrationSourceErrorsPrecedeDatabaseAccess(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing")
	if _, err := NewRunner(Config{SourceURL: migrationSourceURL(missing), DatabaseURL: "secret-sentinel"}); err == nil || !strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "secret-sentinel") {
		t.Fatal("missing source must fail before database access without leaking URL")
	}
	dir := t.TempDir()
	for _, name := range []string{"000001_one.up.sql", "000001_two.up.sql"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("SELECT 1;"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := NewRunner(Config{SourceURL: migrationSourceURL(dir), DatabaseURL: "secret-sentinel"}); err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatal("duplicate version must fail before database access")
	}
}

func TestMigrationDiagnosticsNeverExposeSQLOrCredentials(t *testing.T) {
	for _, tc := range []struct {
		err  error
		part string
	}{
		{migrate.ErrDirty{Version: 2}, "dirty at version 2"},
		{migrate.ErrDirty{Version: -1}, "dirty at version -1"},
		{database.Error{OrigErr: &pgconn.PgError{Code: "42601", Message: "secret-sentinel"}, Query: []byte("secret-sentinel")}, "SQLSTATE 42601"},
		{&database.Error{OrigErr: &pgconn.PgError{Code: "55P03", Message: "secret-sentinel"}}, "lock timeout"},
		{&pgconn.PgError{Code: "28P01", Message: "secret-sentinel"}, "authentication failed"},
		{&pgconn.PgError{Code: "3D000", Message: "secret-sentinel"}, "database not found"},
		{&pgconn.PgError{Code: "57014", Message: "secret-sentinel"}, "timeout"},
		{&pgconn.PgError{Code: "42501", Message: "secret-sentinel"}, "privileges"},
		{&pgconn.PgError{Code: "secret-sentinel"}, "operation failed"},
		{context.DeadlineExceeded, "timeout"}, {syscall.ECONNREFUSED, "connection refused"},
		{fs.ErrNotExist, "not found"}, {errors.New("secret-sentinel"), "operation failed"},
		{errors.Join(migrate.ErrNoChange, errors.New("secret-sentinel")), "operation failed"},
	} {
		err := migrationError("test migration", tc.err)
		if err == nil || !strings.Contains(err.Error(), tc.part) || strings.Contains(err.Error(), "secret-sentinel") {
			t.Fatal("unsafe or uninformative migration diagnostic")
		}
	}
	if err := (&Runner{}).Force(-2); err == nil {
		t.Fatal("invalid version must fail before driver invocation")
	}
}
