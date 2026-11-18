// Package seeding implements explicit local database provisioning.
package seeding

import (
	"context"
	"database/sql"
	"errors"
	"net"
	"net/netip"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/stdlib"

	"hris-core-service/config"
	"hris-core-service/internal/infrastructure"
)

type Target struct {
	pool    *sql.DB
	conn    *sql.Conn
	timeout time.Duration
}

// Open verifies the operator's target before connecting and pins the connection
// so subsequent writes use the peer and database we inspected.
func Open(ctx context.Context, expected string) (*Target, error) {
	if err := config.LoadConfig(); err != nil {
		return nil, errors.New("invalid local configuration; check config.json and environment overrides")
	}
	cfg := config.AppConfig.Database
	ip, err := validateTarget(config.AppConfig.App.Env, cfg, expected)
	if err != nil {
		return nil, err
	}
	_, pool, err := infrastructure.ConnectDB()
	if err != nil {
		return nil, err
	}
	target := &Target{pool: pool, timeout: time.Duration(cfg.TransactionTimeoutSeconds) * time.Second}
	ctx, cancel := context.WithTimeout(ctx, time.Duration(cfg.QueryTimeoutSeconds)*time.Second)
	defer cancel()
	target.conn, err = pool.Conn(ctx)
	if err == nil {
		err = target.conn.Raw(func(driver any) error {
			conn, ok := driver.(*stdlib.Conn)
			if !ok {
				return errors.New("unexpected database driver")
			}
			peer, ok := conn.Conn().PgConn().Conn().RemoteAddr().(*net.TCPAddr)
			if !ok || peer.Port != cfg.Port || !peer.IP.Equal(ip.AsSlice()) {
				return errors.New("database peer does not match the confirmed loopback target")
			}
			return nil
		})
	}
	if err != nil {
		target.Close()
		return nil, errors.New("could not verify the database connection peer")
	}
	var database, role string
	if err := target.conn.QueryRowContext(ctx, "SELECT pg_catalog.current_database(), current_user").Scan(&database, &role); err != nil || database != cfg.Name || role != cfg.Username {
		target.Close()
		return nil, errors.New("connected database or role does not match the confirmed configuration")
	}
	return target, nil
}

func validateTarget(environment string, cfg config.DatabaseConfig, expected string) (netip.Addr, error) {
	if environment != "development" {
		return netip.Addr{}, errors.New("provisioning requires the development environment")
	}
	ip, err := netip.ParseAddr(cfg.Host)
	if err != nil || !ip.IsLoopback() || ip.Zone() != "" {
		return netip.Addr{}, errors.New("provisioning requires a numeric loopback PostgreSQL host")
	}
	if expected != net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port))+"/"+cfg.Name {
		return netip.Addr{}, errors.New("--target must match the effective PostgreSQL host:port/database")
	}
	return ip, nil
}

func (t *Target) Close() {
	if t.conn != nil {
		_ = t.conn.Close()
	}
	_ = t.pool.Close()
}

// Write checks the migration state and executes a single dataset atomically.
func (t *Target) Write(ctx context.Context, run func(context.Context, *sql.Tx) error) error {
	ctx, cancel := context.WithTimeout(ctx, t.timeout)
	defer cancel()
	tx, err := t.conn.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return errors.New("could not start database transaction")
	}
	defer tx.Rollback()
	// The migrator changes this table before applying SQL. Hold its write
	// boundary while a dataset depends on the verified schema version.
	if _, err := tx.ExecContext(ctx, "LOCK TABLE public.schema_migrations IN SHARE MODE"); err != nil {
		return errors.New("could not verify migration state; apply migration 000015 or later")
	}
	var ready bool
	if err := tx.QueryRowContext(ctx, "SELECT count(*) = 1 AND coalesce(bool_and(version >= 15 AND NOT dirty), false) FROM public.schema_migrations").Scan(&ready); err != nil || !ready {
		return errors.New("provisioning requires a clean migration version of at least 15")
	}
	if err := run(ctx, tx); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return errors.New("database commit could not be confirmed; inspect the target before retrying")
	}
	return nil
}
