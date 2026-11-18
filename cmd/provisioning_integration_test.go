package cmd

import (
	"context"
	"crypto/rand"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"hris-core-service/config"
	authstore "hris-core-service/internal/infrastructure/authentication"
	accessstore "hris-core-service/internal/infrastructure/authorization"
	"hris-core-service/internal/infrastructure/migration"
	"hris-core-service/internal/shared/password"
	auth "hris-core-service/internal/usecases/authentication"
	access "hris-core-service/internal/usecases/authorization"
)

//go:embed testdata/bootstrap_pty.py
var bootstrapPTY string

func TestProvisioningPostgresIntegration(t *testing.T) {
	if os.Getenv("M7S_INTEGRATION") != "1" {
		t.Skip("set M7S_INTEGRATION=1 and M7S_CLI_BINARY for disposable PostgreSQL and terminal checks")
	}
	binary := os.Getenv("M7S_CLI_BINARY")
	if !filepath.IsAbs(binary) {
		t.Fatal("M7S_CLI_BINARY must be an absolute path to the current CLI")
	}
	python, err := exec.LookPath("python3")
	if err != nil {
		t.Fatal("python3 is required to exercise real terminal password input")
	}
	t.Chdir("..")
	if err := config.LoadConfig(); err != nil {
		t.Fatal("local verification configuration unavailable")
	}
	base := config.AppConfig.Database
	host := net.ParseIP(base.Host)
	if host == nil || !host.IsLoopback() {
		t.Fatal("verification requires a numeric loopback PostgreSQL host")
	}
	baseURL := migration.NewConfig().DatabaseURL
	ctx, cancel := context.WithTimeout(context.Background(), 75*time.Second)
	defer cancel()
	connect := func(database string) *pgx.Conn {
		t.Helper()
		u, err := url.Parse(baseURL)
		if err != nil {
			t.Fatal("verification connection configuration invalid")
		}
		u.Path, u.RawPath = "/"+database, ""
		cfg, err := pgx.ParseConfig(u.String())
		if err != nil {
			t.Fatal("verification connection configuration invalid")
		}
		cfg.RuntimeParams["statement_timeout"] = "8000"
		conn, err := pgx.ConnectConfig(ctx, cfg)
		if err != nil {
			t.Fatal("verification connection failed (details withheld)")
		}
		return conn
	}
	anonymous := make([]byte, 16)
	if _, err := rand.Read(anonymous); err != nil {
		t.Fatal(err)
	}
	database := "m7s_verify_" + hex.EncodeToString(anonymous)
	if database == base.Name {
		t.Fatal("disposable database must differ from the configured development database")
	}
	admin := connect("postgres")
	defer admin.Close(context.Background())
	identifier := pgx.Identifier{database}.Sanitize()
	if _, err := admin.Exec(ctx, "CREATE DATABASE "+identifier); err != nil {
		t.Fatal("disposable database creation failed (details withheld)")
	}
	var oid uint32
	const ownerQuery = "SELECT oid FROM pg_database WHERE datname=$1 AND datdba=(SELECT oid FROM pg_roles WHERE rolname=current_user)"
	if err := admin.QueryRow(ctx, ownerQuery, database).Scan(&oid); err != nil {
		t.Fatal("cannot verify disposable database ownership")
	}
	defer func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()
		var observed uint32
		if err := admin.QueryRow(cleanup, ownerQuery, database).Scan(&observed); err != nil || observed != oid {
			t.Error("disposable database identity changed; refusing cleanup")
			return
		}
		if _, err := admin.Exec(cleanup, "DROP DATABASE "+identifier); err != nil {
			t.Error("disposable database cleanup failed (details withheld)")
		} else {
			t.Logf("Removed verified disposable database %s", database)
		}
	}()
	db := connect(database)
	defer db.Close(context.Background())
	count := func(query string, args ...any) int {
		t.Helper()
		var n int
		if db.QueryRow(ctx, query, args...).Scan(&n) != nil {
			t.Fatal("disposable verification query failed (details withheld)")
		}
		return n
	}
	fixture := func(query string, args ...any) {
		t.Helper()
		if _, err := db.Exec(ctx, query, args...); err != nil {
			t.Fatal("disposable fixture SQL failed (details withheld)")
		}
	}
	if count("SELECT current_setting('server_version_num')::int / 10000") != 17 ||
		count("SELECT count(*) FROM pg_tables WHERE schemaname='public'") != 0 {
		t.Fatal("a fresh PostgreSQL 17 database is required")
	}
	target := net.JoinHostPort(base.Host, strconv.Itoa(base.Port)) + "/" + database
	env := append(os.Environ(), "DB_NAME="+database, "APP_ENV=development")
	command := func(dir string, extraEnv []string, want string, success bool, args ...string) string {
		t.Helper()
		c := exec.CommandContext(ctx, binary, args...)
		c.Dir, c.Env = dir, append(append([]string{}, env...), extraEnv...)
		out, err := c.CombinedOutput()
		if (err == nil) != success || !strings.Contains(string(out), want) || strings.Contains(string(out), base.Password) {
			t.Fatalf("CLI did not return expected status/message for %v (output withheld)", args)
		}
		return string(out)
	}
	emptyDir := t.TempDir()
	listed := command(emptyDir, nil, "dev-users", true, "seed", "list")
	for _, name := range []string{"dev-users", "demo-tenant", "demo-organization", "demo-memberships"} {
		if !strings.Contains(listed, name) {
			t.Fatalf("registered seeder missing: %s", name)
		}
	}
	command(emptyDir, nil, "unknown seeder", false, "seed", "run", "unknown")
	command(emptyDir, nil, "Available Commands", true, "seed", "--help")
	command("", nil, "requires an interactive terminal", false, "bootstrap", "super-admin", "--target", target)
	command("", []string{"APP_ENV=production", "AUTH_REFRESH_COOKIE_SECURE=true"}, "requires the development environment", false, "seed", "run", "dev-users", "--target", target)
	command("", []string{"DB_HOST=192.0.2.1"}, "requires a numeric loopback", false, "seed", "run", "dev-users", "--target", target)
	command("", nil, "--target must match", false, "seed", "run", "dev-users", "--target", target+"-incorrect")
	command("", nil, "verify migration state", false, "seed", "run", "dev-users", "--target", target)
	command("", nil, "Migrations applied successfully", true, "migrate", "up")
	command("", nil, "Migration version: 15 dirty: false", true, "migrate", "version")

	seed := func(name, want string, success bool) {
		t.Helper()
		command("", nil, want, success, "seed", "run", name, "--target", target)
	}
	fixture("UPDATE public.schema_migrations SET dirty=true")
	seed("dev-users", "requires a clean migration version", false)
	fixture("UPDATE public.schema_migrations SET dirty=false")
	seed("demo-organization", "run seed run demo-tenant", false)
	seed("demo-memberships", "run seed run demo-tenant", false)
	fixture("INSERT INTO public.tenants(code,slug,name,timezone) VALUES ('MAPATU_DEMO','unrelated','Unrelated','UTC')")
	seed("demo-tenant", "identity conflicts", false)
	fixture("DELETE FROM public.tenants WHERE slug='unrelated'")
	seed("demo-tenant", "Created: 1", true)
	seed("demo-memberships", "run seed run dev-users", false)
	fixture("INSERT INTO public.users(email,display_name,is_active) VALUES (' MAPATU-DEMO-USER@example.invalid ','Unrelated',true)")
	seed("dev-users", "identity conflicts", false)
	fixture("DELETE FROM public.users WHERE display_name='Unrelated'")
	seed("dev-users", "Created: 1", true)
	seed("demo-organization", "Created: 1", true)
	seed("demo-memberships", "Created: 1", true)
	fixture("UPDATE public.tenants SET name='Local edit',is_active=false")
	fixture("UPDATE public.organizations SET name='Local org edit'")
	fixture("UPDATE public.tenant_memberships SET is_active=false")
	for _, name := range []string{"dev-users", "demo-tenant", "demo-organization", "demo-memberships"} {
		seed(name, "Already exists: 1. Created: 0", true)
	}
	if count("SELECT count(*) FROM public.users WHERE email='mapatu-demo-user@example.invalid' AND NOT is_active AND password_hash IS NULL") != 1 ||
		count("SELECT count(*) FROM public.tenants WHERE name='Local edit' AND NOT is_active") != 1 ||
		count("SELECT count(*) FROM public.organizations o JOIN public.tenants t ON t.id=o.tenant_id WHERE o.name='Local org edit' AND t.code='MAPATU_DEMO'") != 1 ||
		count("SELECT count(*) FROM public.tenant_memberships m JOIN public.users u ON u.id=m.user_id JOIN public.tenants t ON t.id=m.tenant_id WHERE NOT m.is_active AND t.code='MAPATU_DEMO' AND u.email='mapatu-demo-user@example.invalid'") != 1 ||
		count("SELECT count(*) FROM public.role_assignments") != 0 {
		t.Fatal("demo fixtures or preserved state mismatch")
	}
	t.Log("Verified CLI target and migration checks, seed dependencies, ownership collisions and safe reruns")

	secretBytes := make([]byte, 32)
	if _, err := rand.Read(secretBytes); err != nil {
		t.Fatal(err)
	}
	secret := hex.EncodeToString(secretBytes)
	clear(secretBytes)
	type outcome struct {
		Code   int
		Result string
	}
	runBootstrap := func(email, mode, confirmedTarget string, overrides ...string) outcome {
		inputSecret := secret
		if mode == "invalid-password" {
			inputSecret = strings.Repeat("x", 14)
		}
		account, _ := json.Marshal(map[string]string{"email": email, "password": inputSecret, "mode": mode})
		c := exec.CommandContext(ctx, python, "-c", bootstrapPTY, binary, "bootstrap", "super-admin", "--target", confirmedTarget)
		c.Env = append(append([]string{}, env...), overrides...)
		c.Stdin = strings.NewReader(string(account))
		out, err := c.Output()
		var result outcome
		if err != nil || json.Unmarshal(out, &result) != nil {
			return outcome{-1, "terminal verification failed"}
		}
		return result
	}
	bootstrap := func(email string) outcome { return runBootstrap(email, "", target) }
	expect := func(label string, got outcome, result string, success bool) {
		t.Helper()
		if got.Result != result || (got.Code == 0) != success {
			t.Fatalf("%s: got %v, expected %s (credentials withheld)", label, got, result)
		}
	}
	expect("interactive wrong target", runBootstrap("wrong@example.invalid", "reject", target+"-incorrect"), "target", false)
	expect("interactive production environment", runBootstrap("wrong@example.invalid", "reject", target,
		"APP_ENV=production", "AUTH_REFRESH_COOKIE_SECURE=true"), "environment", false)
	expect("password prompt interruption", runBootstrap("interrupted@example.invalid", "interrupt", target), "interrupted", false)
	if count("SELECT count(*) FROM public.users") != 1 {
		t.Fatal("refused or interrupted bootstrap wrote an account")
	}
	expect("short bootstrap password", runBootstrap("short@example.invalid", "invalid-password", target), "password-policy", false)
	if count("SELECT count(*) FROM public.users") != 1 {
		t.Fatal("short password wrote an account")
	}
	expect("existing normalized account", bootstrap("MAPATU-DEMO-USER@example.invalid"), "existing", false)
	if count("SELECT count(*) FROM public.role_assignments") != 0 {
		t.Fatal("bootstrap elevated existing account")
	}
	fixture(`INSERT INTO public.role_assignments (user_id,role_id,scope_type,is_active)
		SELECT u.id,r.id,'PLATFORM',false FROM public.users u CROSS JOIN public.roles r
		WHERE u.email='mapatu-demo-user@example.invalid' AND r.code='SUPER_ADMIN'`)
	expect("pre-existing inactive platform assignment", bootstrap("prior-inactive@example.invalid"), "already", false)
	fixture("DELETE FROM public.role_assignments")
	// Induce an assignment insert failure inside the transaction after user creation.
	fixture("ALTER TABLE public.role_assignments ADD CONSTRAINT test_reject_platform CHECK (scope_type <> 'PLATFORM')")
	expect("assignment insert rollback", bootstrap("rollback@example.invalid"), "rollback", false)
	fixture("ALTER TABLE public.role_assignments DROP CONSTRAINT test_reject_platform")
	if count("SELECT count(*) FROM public.users WHERE email='rollback@example.invalid'") != 0 {
		t.Fatal("bootstrap left a partially created account")
	}
	results := make(chan outcome, 2)
	go func() { results <- bootstrap("admin-one@example.invalid") }()
	go func() { results <- bootstrap("admin-two@example.invalid") }()
	first, second := <-results, <-results
	if first.Result == "already" {
		first, second = second, first
	}
	expect("concurrent first bootstrap", first, "created", true)
	expect("concurrent second bootstrap", second, "already", false)

	var email, hash string
	if db.QueryRow(ctx, `SELECT u.email, u.password_hash FROM public.users u
		JOIN public.role_assignments a ON a.user_id=u.id JOIN public.roles r ON r.id=a.role_id
		WHERE u.is_active AND r.code='SUPER_ADMIN' AND a.is_active AND a.scope_type='PLATFORM'
		  AND a.tenant_id IS NULL AND a.organization_id IS NULL AND a.branch_id IS NULL`).Scan(&email, &hash) != nil ||
		hash == secret || !password.Verify(hash, secret) || count("SELECT count(*) FROM public.role_assignments") != 1 ||
		count("SELECT count(*) FROM public.users") != 2 || count("SELECT count(*) FROM public.tenant_memberships") != 1 {
		t.Fatal("bootstrap user or platform assignment violated the contract")
	}
	pool := stdlib.OpenDB(*db.Config())
	defer pool.Close()
	gormDB, err := gorm.Open(postgres.New(postgres.Config{Conn: pool}), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal("disposable authentication observer unavailable")
	}
	service := auth.NewService(authstore.NewStore(gormDB, time.Second), time.Minute)
	login, err := service.Login(ctx, email, secret, auth.Metadata{})
	if err != nil {
		t.Fatal("bootstrapped administrator could not sign in")
	}
	identity, err := service.Authenticate(ctx, login.AccessToken)
	if err != nil {
		t.Fatal("bootstrapped administrator session could not be authenticated")
	}
	authCtx := auth.WithIdentity(ctx, identity)
	authorizer := access.NewService(accessstore.NewStore(pool, time.Second))
	if _, err := authorizer.Authorize(authCtx, "platform.manage", access.Scope{Type: "PLATFORM"}); err != nil {
		t.Fatal("bootstrapped administrator lacks the platform grant")
	}
	var demoTenant uuid.UUID
	if err := db.QueryRow(ctx, "SELECT id FROM public.tenants WHERE code='MAPATU_DEMO'").Scan(&demoTenant); err != nil {
		t.Fatal("demo tenant unavailable")
	}
	if _, err := authorizer.Authorize(authCtx, "tenant.read", access.Scope{Type: "TENANT", TenantID: demoTenant}); !errors.Is(err, access.ErrForbidden) {
		t.Fatal("platform login granted tenant data")
	}
	fixture("UPDATE public.role_assignments SET is_active=false")
	expect("inactive assignment", bootstrap("inactive-retry@example.invalid"), "already", false)
	fixture("DELETE FROM public.role_assignments")
	expect("reserved bootstrap user", bootstrap("removed-assignment-retry@example.invalid"), "already", false)
	for query, want := range map[string]int{
		"SELECT count(*) FROM public.roles":            6,
		"SELECT count(*) FROM public.permissions":      8,
		"SELECT count(*) FROM public.role_permissions": 19,
		"SELECT count(*) FROM public.users":            2,
	} {
		if count(query) != want {
			t.Fatalf("unexpected row count for %s", query)
		}
	}
	t.Log("Verified hidden terminal input and interruption restoration, Argon2id, duplicate-account refusal, atomic rollback, concurrent and inactive guards, authenticated platform grant and reference counts")
}
