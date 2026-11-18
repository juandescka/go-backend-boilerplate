package migration

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database"
	"github.com/jackc/pgx/v5"
	"hris-core-service/config"
)

// Opt-in only: creates and drops exactly one randomly named, test-owned DB.
// Never applies migrations or metadata changes to the configured development DB.
func TestPostgresMigrationIntegration(t *testing.T) {
	if os.Getenv("M4B_INTEGRATION") != "1" {
		t.Skip("set M4B_INTEGRATION=1 for disposable PostgreSQL migration verification")
	}
	t.Chdir("../../..")
	previous := config.AppConfig
	t.Cleanup(func() { config.AppConfig = previous })
	if err := config.LoadConfig(); err != nil {
		t.Fatal("unable to load local configuration")
	}
	base := config.AppConfig.Database
	token := make([]byte, 8)
	if _, err := rand.Read(token); err != nil {
		t.Fatal(err)
	}
	dbName := "m4b_verify_" + hex.EncodeToString(token)
	if dbName == base.Name {
		t.Fatal("disposable database name equals configured database")
	}
	adminCfg := base
	adminCfg.Name = "postgres"
	connect := func(c config.DatabaseConfig, name string) *pgx.Conn {
		parsed, err := pgx.ParseConfig(migrationDatabaseURL(c))
		if err != nil {
			t.Fatal("invalid test connection configuration")
		}
		parsed.RuntimeParams["application_name"] = name
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		conn, err := pgx.ConnectConfig(ctx, parsed)
		if err != nil {
			t.Fatal("test PostgreSQL connection failed (details withheld)")
		}
		return conn
	}
	admin := connect(adminCfg, "m4b-test-admin")
	var observer *pgx.Conn
	owned := false
	var oid uint32
	var runners []*Runner
	t.Cleanup(func() {
		for _, r := range runners {
			if err := r.Close(); err != nil {
				t.Error("migration cleanup failed")
			}
		}
		if observer != nil {
			_ = observer.Close(context.Background())
		}
		if owned {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			var matches int
			if err := admin.QueryRow(ctx, "SELECT count(*) FROM pg_database WHERE datname=$1 AND oid=$2 AND datdba=(SELECT oid FROM pg_roles WHERE rolname=current_user)", dbName, oid).Scan(&matches); err != nil || matches != 1 {
				t.Errorf("database identity or ownership changed; refusing cleanup of %s", dbName)
				_ = admin.Close(context.Background())
				return
			}
			if _, err := admin.Exec(ctx, "DROP DATABASE "+pgx.Identifier{dbName}.Sanitize()); err != nil {
				t.Errorf("could not remove test-owned database %s; details withheld", dbName)
			} else {
				if err := admin.QueryRow(ctx, "SELECT count(*) FROM pg_database WHERE datname=$1", dbName).Scan(&matches); err != nil || matches != 0 {
					t.Error("disposable database removal could not be verified")
				} else {
					t.Logf("Removed ownership/OID-checked disposable database %s", dbName)
				}
			}
		}
		_ = admin.Close(context.Background())
	})
	var permitted bool
	if err := admin.QueryRow(context.Background(), "SELECT rolcreatedb OR rolsuper FROM pg_roles WHERE rolname=current_user").Scan(&permitted); err != nil || !permitted {
		t.Fatal("safe disposable database creation is unavailable")
	}
	if _, err := admin.Exec(context.Background(), "CREATE DATABASE "+pgx.Identifier{dbName}.Sanitize()); err != nil {
		t.Fatal("unable to create unique disposable database (no existing database is reused)")
	}
	if err := admin.QueryRow(context.Background(), "SELECT oid FROM pg_database WHERE datname=$1 AND datdba=(SELECT oid FROM pg_roles WHERE rolname=current_user)", dbName).Scan(&oid); err != nil {
		t.Fatalf("created database ownership unavailable; inspect %s manually", dbName)
	}
	owned = true
	base.Name = dbName
	observer = connect(base, "m4b-test-observer")
	t.Log("Created isolated test database; configured development database remains untouched")
	queryInt := func(query string, args ...any) int {
		var n int
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := observer.QueryRow(ctx, query, args...).Scan(&n); err != nil {
			t.Fatal("disposable verification query failed (details withheld)")
		}
		return n
	}
	noSessions := func() {
		for i := 0; i < 30; i++ {
			if queryInt("SELECT count(*) FROM pg_stat_activity WHERE datname=$1 AND application_name='hris-core-service-migrate'", dbName) == 0 {
				return
			}
			time.Sleep(20 * time.Millisecond)
		}
		t.Fatal("migration connections remain after cleanup")
	}
	sourceDir := func(files map[string]string) string {
		dir := t.TempDir()
		for name, sql := range files {
			if err := os.WriteFile(filepath.Join(dir, name), []byte(sql), 0600); err != nil {
				t.Fatal(err)
			}
		}
		return dir
	}
	runnerFor := func(dir string, c config.DatabaseConfig) *Runner {
		r, err := NewRunner(Config{SourceURL: migrationSourceURL(dir), DatabaseURL: migrationDatabaseURL(c)})
		if err != nil {
			t.Fatalf("runner initialization: %s", err)
		}
		runners = append(runners, r)
		return r
	}
	closeRunner := func(r *Runner) {
		if r.Close() != nil || r.Close() != nil {
			t.Fatal("repeated Close failed")
		}
		noSessions()
	}
	state := func(r *Runner, want int, dirty bool) {
		v, d, e := r.Version()
		if e != nil || v != want || d != dirty {
			t.Fatalf("unexpected state: got version=%d dirty=%t; expected version=%d dirty=%t", v, d, want, dirty)
		}
	}
	must := func(err error) {
		if err != nil {
			t.Fatalf("migration operation: %s", err)
		}
	}
	empty := sourceDir(map[string]string{".gitkeep": "", "unparseable.sql": "SELECT bad_filename_is_ignored;"})
	// Source failures must not create schema_migrations or open a connection.
	if _, err := NewRunner(Config{SourceURL: migrationSourceURL(filepath.Join(empty, "missing")), DatabaseURL: migrationDatabaseURL(base)}); err == nil {
		t.Fatal("missing source accepted")
	}
	if queryInt("SELECT count(*) FROM information_schema.tables WHERE table_name='schema_migrations'") != 0 {
		t.Fatal("source failure initialized metadata")
	}
	noSessions()
	r := runnerFor(empty, base)
	state(r, -1, false)
	if queryInt("SELECT count(*) FROM information_schema.tables WHERE table_name='schema_migrations'") != 1 {
		t.Fatal("metadata initialization not observed")
	}
	if err := r.Up(); err == nil || !strings.Contains(err.Error(), "contains no versions") {
		t.Fatal("fresh empty Up must report missing first version")
	}
	if err := r.Down(); err == nil || !strings.Contains(err.Error(), "no applied migration") {
		t.Fatal("fresh Down must report no applied version")
	}
	closeRunner(r)
	t.Log("PASS empty source, ignored invalid filename, fresh version, metadata side effects, missing-source preflight, repeated Close")

	goodFiles := map[string]string{
		"000001_first.up.sql":    "CREATE TABLE m4b_fixture_one (id integer PRIMARY KEY); INSERT INTO m4b_fixture_one VALUES (1);",
		"000001_first.down.sql":  "DROP TABLE m4b_fixture_one;",
		"000002_second.up.sql":   "ALTER TABLE m4b_fixture_one ADD COLUMN note text; CREATE TABLE m4b_fixture_two (id integer);",
		"000002_second.down.sql": "DROP TABLE m4b_fixture_two; ALTER TABLE m4b_fixture_one DROP COLUMN note;",
	}
	good := sourceDir(goodFiles)
	r = runnerFor(good, base)
	must(r.Up())
	state(r, 2, false)
	if queryInt("SELECT count(*) FROM m4b_fixture_one") != 1 || queryInt("SELECT count(*) FROM information_schema.tables WHERE table_name='m4b_fixture_two'") != 1 {
		t.Fatal("ordered/compound migration failed")
	}
	if !errors.Is(r.Up(), migrate.ErrNoChange) {
		t.Fatal("repeated Up must report ErrNoChange")
	}
	must(r.Down())
	state(r, 1, false)
	if queryInt("SELECT count(*) FROM information_schema.tables WHERE table_name='m4b_fixture_one'") != 1 || queryInt("SELECT count(*) FROM information_schema.tables WHERE table_name='m4b_fixture_two'") != 0 {
		t.Fatal("Down rolled back beyond one version")
	}
	must(r.Down())
	state(r, -1, false)
	if queryInt("SELECT count(*) FROM information_schema.tables WHERE table_name='m4b_fixture_one'") != 0 {
		t.Fatal("first rollback did not remove its fixture")
	}
	must(r.Force(42))
	if e := r.Up(); e == nil || errors.Is(e, migrate.ErrNoChange) {
		t.Fatal("forced version absent from source must fail, not report no change")
	}
	if e := r.Down(); e == nil {
		t.Fatal("rollback from a version absent from source must fail")
	}
	must(r.Force(-1))
	closeRunner(r)
	t.Log("PASS ordered two-version compound SQL, repeat Up no-change, exact one-step Down, rollback to none")
	// v4.18.3 permits a missing direction as a metadata-only migration. Record
	// that limitation; complete up/down pairs remain the repository convention.
	unpaired := sourceDir(map[string]string{"000001_unpaired.up.sql": "CREATE TABLE m4b_unpaired_fixture (id integer);"})
	r = runnerFor(unpaired, base)
	must(r.Up())
	must(r.Down())
	state(r, -1, false)
	if queryInt("SELECT count(*) FROM information_schema.tables WHERE table_name='m4b_unpaired_fixture'") != 1 {
		t.Fatal("unpaired-direction library behavior changed")
	}
	// The inspected table still exists: reconcile metadata explicitly and add
	// its real rollback before cleanup, only in this test-owned disposable DB.
	must(r.Force(1))
	closeRunner(r)
	must(os.WriteFile(filepath.Join(unpaired, "000001_unpaired.down.sql"), []byte("DROP TABLE m4b_unpaired_fixture;"), 0600))
	r = runnerFor(unpaired, base)
	must(r.Down())
	closeRunner(r)
	t.Log("PASS absent forced-version errors and verified missing-direction limitation; no false no-change")

	bad := sourceDir(map[string]string{
		"000001_dirty.up.sql":   "CREATE TABLE m4b_dirty_fixture (id integer); SELECT private_sql_sentinel FROM m4b_missing_fixture;",
		"000001_dirty.down.sql": "SELECT private_down_sentinel FROM m4b_missing_fixture;",
	})
	r = runnerFor(bad, base)
	err := r.Up()
	if err == nil || strings.Contains(err.Error(), "private_sql_sentinel") || !strings.Contains(err.Error(), "SQLSTATE") {
		t.Fatal("SQL failure was not safely diagnosed")
	}
	state(r, 1, true)
	if queryInt("SELECT count(*) FROM information_schema.tables WHERE table_name='m4b_dirty_fixture'") != 0 {
		t.Fatal("compound failing SQL was partially applied")
	}
	if e := r.Up(); e == nil || !strings.Contains(e.Error(), "dirty at version 1") {
		t.Fatal("dirty database did not block Up")
	}
	if e := r.Down(); e == nil || !strings.Contains(e.Error(), "dirty at version 1") {
		t.Fatal("dirty database did not block Down")
	}
	state(r, 1, true)
	// Explicit recovery is safe only after verifying the failed compound SQL rolled back.
	must(r.Force(-1))
	state(r, -1, false)
	closeRunner(r)
	must(os.WriteFile(filepath.Join(bad, "000001_dirty.up.sql"), []byte("CREATE TABLE m4b_dirty_fixture (id integer);"), 0600))
	r = runnerFor(bad, base)
	must(r.Up())
	state(r, 1, false)
	if e := r.Down(); e == nil || strings.Contains(e.Error(), "private_down_sentinel") {
		t.Fatal("controlled down failure was not sanitized")
	}
	state(r, -1, true)
	if e := r.Up(); e == nil || !strings.Contains(e.Error(), "dirty at version -1") {
		t.Fatal("dirty nil version did not block Up")
	}
	if queryInt("SELECT count(*) FROM information_schema.tables WHERE table_name='m4b_dirty_fixture'") != 1 {
		t.Fatal("failed first rollback changed actual table")
	}
	t.Log("PASS dirty Up refusal, no automatic force, atomic compound failure, failed first Down retains signed dirty nil state")

	// Real Cobra binary checks use an isolated config and temporary source.
	// Secret values are passed in the subprocess environment, never persisted/printed.
	binary := os.Getenv("M4B_CLI_BINARY")
	cli := func(sourcePath string, args ...string) string {
		t.Helper()
		if binary == "" {
			t.Fatal("set M4B_CLI_BINARY to the current built backend for CLI verification")
		}
		cwd := t.TempDir()
		cfg := config.AppConfig
		cfg.Security.JWTAuthSecretKey = ""
		cfg.Database.Password = ""
		content, e := json.Marshal(cfg)
		must(e)
		must(os.WriteFile(filepath.Join(cwd, "config.json"), content, 0600))
		must(os.Mkdir(filepath.Join(cwd, "migrations"), 0700))
		must(os.Symlink(sourcePath, filepath.Join(cwd, "migrations", "postgres")))
		overrides := map[string]string{"DB_NAME": dbName, "DB_PASSWORD": base.Password, "JWT_AUTH_SECRET_KEY": config.AppConfig.Security.JWTAuthSecretKey}
		env := []string{}
		for _, entry := range os.Environ() {
			key, _, _ := strings.Cut(entry, "=")
			if _, set := overrides[key]; !set {
				env = append(env, entry)
			}
		}
		for key, value := range overrides {
			env = append(env, key+"="+value)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, binary, args...)
		cmd.Dir = cwd
		cmd.Env = env
		out, e := cmd.CombinedOutput()
		for _, secret := range []string{base.Password, config.AppConfig.Security.JWTAuthSecretKey} {
			if secret != "" && strings.Contains(string(out), secret) {
				t.Fatal("secret exposed by Cobra output")
			}
		}
		if e != nil {
			t.Fatal("Cobra command failed (output withheld)")
		}
		return string(out)
	}
	if out := cli(bad, "migrate", "version"); !strings.Contains(out, "none dirty: true") {
		t.Fatal("CLI lost dirty nil version")
	}
	// After inspecting failed down left the table intact, reconcile metadata to 1.
	must(r.Force(1))
	closeRunner(r)
	must(os.WriteFile(filepath.Join(bad, "000001_dirty.down.sql"), []byte("DROP TABLE m4b_dirty_fixture;"), 0600))
	r = runnerFor(bad, base)
	must(r.Down())
	state(r, -1, false)
	closeRunner(r)
	if out := cli(good, "migrate", "up"); !strings.Contains(out, "applied successfully") {
		t.Fatal("CLI Up failed")
	}
	if out := cli(good, "migrate", "up"); !strings.Contains(out, "No pending migrations") {
		t.Fatal("CLI no-change output incorrect")
	}
	if out := cli(good, "migrate", "version"); !strings.Contains(out, "version: 2 dirty: false") {
		t.Fatal("CLI version incorrect")
	}
	_ = cli(good, "migrate", "down")
	if queryInt("SELECT count(*) FROM information_schema.tables WHERE table_name='m4b_fixture_one'") != 1 || queryInt("SELECT count(*) FROM information_schema.tables WHERE table_name='m4b_fixture_two'") != 0 {
		t.Fatal("CLI Down is not one step")
	}
	_ = cli(good, "migrate", "force", "0")
	if out := cli(good, "migrate", "version"); !strings.Contains(out, "version: 0 dirty: false") {
		t.Fatal("CLI conflated version zero and nil")
	}
	if queryInt("SELECT count(*) FROM m4b_fixture_one") != 1 {
		t.Fatal("force executed SQL migration")
	}
	_ = cli(good, "migrate", "force", "--", "-1")
	if out := cli(good, "migrate", "version"); !strings.Contains(out, "version: none") {
		t.Fatal("CLI nil state incorrect")
	}
	_ = cli(good, "migrate", "force", "1")
	_ = cli(good, "migrate", "down")
	noSessions()
	t.Log("PASS real Cobra up/repeat-up/down/version/force0/force-- -1, dirty nil display, force changes metadata only")

	short := base
	short.QueryTimeoutSeconds = 1
	slow := sourceDir(map[string]string{"000001_slow.up.sql": "SELECT pg_sleep(3);", "000001_slow.down.sql": "SELECT 1;"})
	r = runnerFor(slow, short)
	start := time.Now()
	err = r.Up()
	if err == nil || !strings.Contains(err.Error(), "timeout") || time.Since(start) > 3*time.Second {
		t.Fatal("statement timeout failed")
	}
	// Cancellation may discard the reserved pgx connection. Each CLI command
	// starts a new runner; inspect durable dirty state through that fresh session.
	closeRunner(r)
	r = runnerFor(slow, short)
	state(r, 1, true)
	must(r.Force(-1))
	closeRunner(r)
	// Hold exactly the official driver's advisory key in this disposable DB.
	locker := connect(base, "m4b-test-locker")
	t.Cleanup(func() { _ = locker.Close(context.Background()) })
	lockID, e := database.GenerateAdvisoryLockId(dbName, "public", "schema_migrations")
	must(e)
	if _, e = locker.Exec(context.Background(), "SELECT pg_advisory_lock($1)", lockID); e != nil {
		t.Fatal("unable to acquire test advisory lock")
	}
	start = time.Now()
	blocked, e := NewRunner(Config{SourceURL: migrationSourceURL(good), DatabaseURL: migrationDatabaseURL(short)})
	if blocked != nil {
		runners = append(runners, blocked)
	}
	if e == nil || !strings.Contains(e.Error(), "timeout") || time.Since(start) > 3*time.Second {
		_ = locker.Close(context.Background())
		t.Fatal("initial metadata lock wait not bounded")
	}
	_ = locker.Close(context.Background())
	noSessions()
	t.Log("PASS configured SQL timeout, bounded initialization advisory-lock wait and failure cleanup")

	// Concurrent migrators serialize using the official PostgreSQL advisory lock.
	concurrent := sourceDir(map[string]string{"000001_lock.up.sql": "SELECT pg_sleep(0.35); CREATE TABLE m4b_lock_fixture (id integer);", "000001_lock.down.sql": "DROP TABLE m4b_lock_fixture;"})
	one := runnerFor(concurrent, base)
	two := runnerFor(concurrent, base)
	var wg sync.WaitGroup
	wg.Add(2)
	results := make(chan error, 2)
	go func() { defer wg.Done(); results <- one.Up() }()
	for i := 0; i < 100; i++ {
		if queryInt("SELECT count(*) FROM pg_locks WHERE locktype='advisory' AND granted AND pid IN (SELECT pid FROM pg_stat_activity WHERE datname=$1)", dbName) > 0 {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	go func() { defer wg.Done(); results <- two.Up() }()
	wg.Wait()
	close(results)
	applied, unchanged := 0, 0
	for e := range results {
		if e == nil {
			applied++
		} else if errors.Is(e, migrate.ErrNoChange) {
			unchanged++
		} else {
			t.Fatal("concurrent migration failed")
		}
	}
	if applied != 1 || unchanged != 1 {
		t.Fatal("advisory lock did not serialize one application")
	}
	must(one.Down())
	must(one.Close())
	must(two.Close())
	noSessions()
	t.Log("PASS concurrent migrators apply once under official advisory locking")

	// Connection errors use the same source/config without any real DB mutation.
	negative := []struct {
		name   string
		cfg    config.DatabaseConfig
		reason string
	}{}
	wrong := base
	wrong.Password = "m4b_intentionally_wrong_password"
	negative = append(negative, struct {
		name   string
		cfg    config.DatabaseConfig
		reason string
	}{"wrong password", wrong, "authentication failed"})
	absent := base
	absent.Name = dbName + "_absent"
	negative = append(negative, struct {
		name   string
		cfg    config.DatabaseConfig
		reason string
	}{"missing database", absent, "database not found"})
	listener, e := net.Listen("tcp", "127.0.0.1:0")
	must(e)
	port := listener.Addr().(*net.TCPAddr).Port
	must(listener.Close())
	refused := base
	refused.Port = port
	negative = append(negative, struct {
		name   string
		cfg    config.DatabaseConfig
		reason string
	}{"unreachable", refused, "connection refused"})
	for _, tc := range negative {
		candidate, e := NewRunner(Config{SourceURL: migrationSourceURL(empty), DatabaseURL: migrationDatabaseURL(tc.cfg)})
		if candidate != nil {
			runners = append(runners, candidate)
		}
		if e == nil || !strings.Contains(e.Error(), tc.reason) || strings.Contains(e.Error(), tc.cfg.Password) {
			t.Fatalf("unsafe/incorrect %s error", tc.name)
		}
	}
	// A listener that never answers PostgreSQL proves the connect timeout.
	stall, e := net.Listen("tcp", "127.0.0.1:0")
	must(e)
	accepted := make(chan net.Conn, 1)
	go func() {
		c, e := stall.Accept()
		if e == nil {
			accepted <- c
		}
	}()
	stalled := short
	stalled.Port = stall.Addr().(*net.TCPAddr).Port
	start = time.Now()
	candidate, e := NewRunner(Config{SourceURL: migrationSourceURL(empty), DatabaseURL: migrationDatabaseURL(stalled)})
	if candidate != nil {
		runners = append(runners, candidate)
	}
	_ = stall.Close()
	select {
	case c := <-accepted:
		_ = c.Close()
	default:
	}
	if e == nil || !strings.Contains(e.Error(), "timeout") || time.Since(start) > 3*time.Second {
		t.Fatal("migration connection timeout not bounded")
	}
	noSessions()
	t.Log("PASS invalid credentials, missing database, refused connection, handshake timeout and no leaked migration sessions")
}
