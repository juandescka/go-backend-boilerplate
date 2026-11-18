package infrastructure

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"syscall"
	"time"

	"hris-core-service/config"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/stdlib"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// ConnectDB creates the application's single pool and checks connectivity before
// returning ownership to the registry. It never runs migrations or business SQL.
func ConnectDB() (*gorm.DB, *sql.DB, error) {
	cfg := config.AppConfig.Database
	driverConfig, err := postgresConnectionConfig(cfg)
	if err != nil {
		return nil, nil, err
	}
	pool := stdlib.OpenDB(*driverConfig)
	pool.SetMaxOpenConns(cfg.MaxOpenConnections)
	pool.SetMaxIdleConns(cfg.MaxIdleConnections)
	pool.SetConnMaxLifetime(time.Duration(cfg.MaxLifetimeSeconds) * time.Second)
	pool.SetConnMaxIdleTime(time.Duration(cfg.MaxIdleTime) * time.Second)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.QueryTimeoutSeconds)*time.Second)
	defer cancel()
	if err := pool.PingContext(ctx); err != nil {
		_ = pool.Close()
		if ctx.Err() != nil {
			err = ctx.Err()
		}
		return nil, nil, databaseStartupError(err)
	}
	db, err := gorm.Open(postgres.New(postgres.Config{Conn: pool}), &gorm.Config{
		DisableAutomaticPing: true,
		Logger:               logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		_ = pool.Close()
		return nil, nil, errors.New("initialize database ORM failed")
	}
	return db, pool, nil
}

func postgresConnectionConfig(cfg config.DatabaseConfig) (*pgx.ConnConfig, error) {
	// A service file must not introduce an alternative application DB identity.
	// Do not mutate process-wide environment to configure the driver.
	if os.Getenv("PGSERVICE") != "" {
		return nil, errors.New("initialize database connector: PGSERVICE is unsupported; use database configuration")
	}
	connectionURL := url.URL{
		Scheme:  "postgres",
		User:    url.UserPassword(cfg.Username, cfg.Password),
		Host:    net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port)),
		Path:    "/" + cfg.Name,
		RawPath: "/" + url.PathEscape(cfg.Name),
	}
	params := url.Values{
		"sslmode":              {cfg.SSLMode},
		"connect_timeout":      {strconv.Itoa(cfg.QueryTimeoutSeconds)},
		"application_name":     {"hris-core-service"},
		"timezone":             {"UTC"},
		"client_encoding":      {"UTF8"},
		"target_session_attrs": {"any"},
	}
	connectionURL.RawQuery = params.Encode()
	driverConfig, err := pgx.ParseConfig(connectionURL.String())
	if err != nil {
		// ParseConfigError and connection errors may contain a DSN or password.
		return nil, errors.New("initialize database connector: invalid PostgreSQL configuration")
	}
	return driverConfig, nil
}

func databaseStartupError(err error) error {
	reason := "connection failed"
	var pgError *pgconn.PgError
	var timeoutError net.Error
	var dnsError *net.DNSError
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		reason = "timeout"
	case errors.Is(err, context.Canceled):
		reason = "canceled"
	case errors.As(err, &pgError):
		switch pgError.Code {
		case "28P01", "28000":
			reason = "authentication failed"
		case "3D000":
			reason = "database not found"
		default:
			reason = "database server rejected connection"
		}
	case errors.Is(err, syscall.ECONNREFUSED):
		reason = "connection refused"
	case errors.As(err, &timeoutError) && timeoutError.Timeout():
		reason = "timeout"
	case errors.As(err, &dnsError):
		reason = "host resolution failed"
	}
	return fmt.Errorf("database startup connectivity check failed: %s", reason)
}
