package migration

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"net/url"
	"os"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
)

type Runner struct {
	migrate     *migrate.Migrate
	database    database.Driver
	source      source.Driver
	connection  *sql.Conn
	pool        *sql.DB
	emptySource bool
	closeOnce   sync.Once
	closeErr    error
}

func NewRunner(cfg Config) (_ *Runner, err error) {
	// Validate/open the source before any database metadata initialization.
	src, err := source.Open(cfg.SourceURL)
	if err != nil {
		return nil, migrationError("open migration source", err)
	}
	var pool *sql.DB
	var conn *sql.Conn
	var driver database.Driver
	ready := false
	defer func() {
		if ready {
			return
		}
		err = errors.Join(err, closeConnection(conn))
		if pool != nil {
			err = errors.Join(err, closeError(pool.Close()))
		}
		err = errors.Join(err, closeError(src.Close()))
	}()
	_, firstErr := src.First()
	empty := errors.Is(firstErr, fs.ErrNotExist)
	if firstErr != nil && !empty {
		return nil, migrationError("read migration source", firstErr)
	}
	if os.Getenv("PGSERVICE") != "" {
		return nil, errors.New("initialize migration: PGSERVICE is unsupported; use database configuration")
	}
	u, parseErr := url.Parse(cfg.DatabaseURL)
	if parseErr != nil || u.Scheme != "postgres" {
		return nil, errors.New("initialize migration: invalid PostgreSQL configuration")
	}
	pgConfig, parseErr := pgx.ParseConfig(cfg.DatabaseURL)
	if parseErr != nil || pgConfig.ConnectTimeout <= 0 {
		return nil, errors.New("initialize migration: invalid PostgreSQL configuration or connection timeout")
	}
	timeout := pgConfig.ConnectTimeout
	// The official driver's metadata/advisory-lock SQL uses background contexts.
	// Bound these server-side before WithConnection; apply the same limit to SQL.
	timeoutMS := strconv.FormatInt(timeout.Milliseconds(), 10)
	pgConfig.RuntimeParams["statement_timeout"] = timeoutMS
	pgConfig.RuntimeParams["lock_timeout"] = timeoutMS
	pool = stdlib.OpenDB(*pgConfig)
	pool.SetMaxOpenConns(1)
	pool.SetMaxIdleConns(1)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	conn, err = pool.Conn(ctx)
	if err != nil {
		return nil, migrationError("connect migration database", err)
	}
	driver, err = postgres.WithConnection(ctx, conn, &postgres.Config{StatementTimeout: timeout})
	if err != nil {
		return nil, migrationError("initialize migration metadata", err)
	}
	engine, err := migrate.NewWithInstance("file", src, "postgres", driver)
	if err != nil {
		return nil, migrationError("initialize migration runner", err)
	}
	// Let PostgreSQL finish its bounded lock query before the library timeout.
	engine.LockTimeout = timeout + time.Second
	ready = true
	return &Runner{migrate: engine, database: driver, source: src, connection: conn, pool: pool, emptySource: empty}, nil
}

func (r *Runner) Close() error {
	r.closeOnce.Do(func() {
		// WithConnection owns only this reserved connection (no driver-owned DB).
		// Close our resources directly: postgres.Close loses ErrConnDone's type
		// when pgx has already discarded a connection after cancellation.
		r.closeErr = errors.Join(closeError(r.source.Close()), closeConnection(r.connection), closeError(r.pool.Close()))
	})
	return r.closeErr
}

func closeError(err error) error {
	if err == nil {
		return nil
	}
	return errors.New("close migration resources failed")
}

func closeConnection(conn *sql.Conn) error {
	if conn == nil {
		return nil
	}
	err := conn.Close()
	if errors.Is(err, sql.ErrConnDone) {
		return nil
	}
	return closeError(err)
}

func (r *Runner) Up() error {
	err := r.migrate.Up()
	// A composite error may include ErrNoChange plus an unlock failure.
	if err == migrate.ErrNoChange {
		return migrate.ErrNoChange
	}
	if errors.Is(err, fs.ErrNotExist) && r.emptySource {
		return errors.New("run up migrations: migration source contains no versions")
	}
	return migrationError("run up migrations", err)
}

// Down rolls back exactly one migration; never call the library's rollback-all.
func (r *Runner) Down() error {
	err := r.migrate.Steps(-1)
	if err == migrate.ErrNoChange {
		return migrate.ErrNoChange
	}
	if errors.Is(err, fs.ErrNotExist) {
		version, dirty, stateErr := r.database.Version()
		if stateErr == nil && version == database.NilVersion && !dirty {
			return errors.New("run down migration: no applied migration to roll back")
		}
	}
	return migrationError("run down migration", err)
}

// Version uses the official driver's signed state to preserve dirty nil (-1)
// after a failed first rollback; migrate.Version() discards that dirty flag.
func (r *Runner) Version() (int, bool, error) {
	version, dirty, err := r.database.Version()
	return version, dirty, migrationError("get migration version", err)
}

func (r *Runner) Force(version int) error {
	if version < database.NilVersion {
		return errors.New("invalid migration version: expected an integer >= -1")
	}
	return migrationError("force migration version", r.migrate.Force(version))
}

func migrationError(operation string, err error) error {
	if err == nil {
		return nil
	}
	reason := "operation failed"
	var dirty migrate.ErrDirty
	var duplicate source.ErrDuplicateMigration
	var dbValue database.Error
	var dbPointer *database.Error
	switch {
	case errors.As(err, &dirty):
		return fmt.Errorf("%s: database is dirty at version %d; inspect state before manual recovery", operation, dirty.Version)
	case errors.As(err, &duplicate):
		return fmt.Errorf("%s: duplicate migration version and direction", operation)
	case errors.Is(err, fs.ErrNotExist):
		return fmt.Errorf("%s: migration directory, file or required version not found", operation)
	}
	// database.Error has no Unwrap and includes SQL/raw details in Error().
	if errors.As(err, &dbValue) {
		err = dbValue.OrigErr
	} else if errors.As(err, &dbPointer) {
		err = dbPointer.OrigErr
	}
	var state interface{ SQLState() string }
	var netErr net.Error
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		reason = "timeout"
	case errors.Is(err, context.Canceled):
		reason = "canceled"
	case errors.Is(err, syscall.ECONNREFUSED):
		reason = "connection refused"
	case errors.As(err, &netErr) && netErr.Timeout():
		reason = "timeout"
	case errors.As(err, &state):
		code := state.SQLState()
		switch code {
		case "28P01", "28000":
			reason = "authentication failed"
		case "3D000":
			reason = "database not found"
		case "55P03":
			reason = "database lock timeout"
		case "57014":
			reason = "statement timeout or cancellation"
		case "42501":
			reason = "insufficient database privileges"
		default:
			valid := len(code) == 5
			for _, c := range code {
				if !(c >= '0' && c <= '9' || c >= 'A' && c <= 'Z') {
					valid = false
				}
			}
			if valid {
				reason = "PostgreSQL rejected operation (SQLSTATE " + code + "); inspect migration file and state"
			}
		}
	}
	return fmt.Errorf("%s: %s", operation, reason)
}
