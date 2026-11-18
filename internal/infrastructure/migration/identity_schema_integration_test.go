package migration

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"hris-core-service/config"
)

// Explicit opt-in: actual repository SQL and Cobra CLI, disposable database only.
// No migration runner (including version) ever targets the configured database.
func TestIdentitySessionSchemaIntegration(t *testing.T) {
	if os.Getenv("M5A_INTEGRATION") != "1" {
		t.Skip("set M5A_INTEGRATION=1 and M5A_CLI_BINARY for disposable schema verification")
	}
	binary, err := filepath.Abs(os.Getenv("M5A_CLI_BINARY"))
	if err != nil || os.Getenv("M5A_CLI_BINARY") == "" {
		t.Fatal("M5A_CLI_BINARY must identify the current built backend")
	}
	t.Chdir("../../..")
	root, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	previous := config.AppConfig
	t.Cleanup(func() { config.AppConfig = previous })
	if config.LoadConfig() != nil {
		t.Fatal("unable to load effective backend configuration (details withheld)")
	}
	base := config.AppConfig.Database
	t.Logf("Effective configured target: host=%s port=%d database=%s user=%s ssl=%s timeout=%ds; password withheld", base.Host, base.Port, base.Name, base.Username, base.SSLMode, base.QueryTimeoutSeconds)
	connect := func(c config.DatabaseConfig, label string) *pgx.Conn {
		t.Helper()
		parsed, e := pgx.ParseConfig(migrationDatabaseURL(c))
		if e != nil {
			t.Fatal("invalid verification configuration")
		}
		parsed.RuntimeParams["application_name"] = label
		parsed.RuntimeParams["statement_timeout"] = "5000"
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		conn, e := pgx.ConnectConfig(ctx, parsed)
		if e != nil {
			t.Fatal("verification connection failed (details withheld)")
		}
		return conn
	}
	query := func(conn *pgx.Conn, sql string, args ...any) pgx.Row {
		// Every query has a server statement bound and connections have a connect bound.
		return conn.QueryRow(context.Background(), sql, args...)
	}
	intValue := func(conn *pgx.Conn, sql string, args ...any) int {
		t.Helper()
		var value int
		if query(conn, sql, args...).Scan(&value) != nil {
			t.Fatal("catalog/integrity query failed (details withheld)")
		}
		return value
	}
	execSQL := func(conn *pgx.Conn, sql string, args ...any) {
		t.Helper()
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if _, e := conn.Exec(ctx, sql, args...); e != nil {
			t.Fatal("verification SQL failed (details withheld)")
		}
	}
	// The only development-target connection performs catalog SELECTs in a
	// read-only transaction. Compare before/after without opening the migrator.
	development := connect(base, "m5a-read-only-baseline")
	t.Cleanup(func() { _ = development.Close(context.Background()) })
	developmentSnapshot := func() string {
		t.Helper()
		execSQL(development, "BEGIN READ ONLY")
		var snapshot string
		e := query(development, `SELECT COALESCE(string_agg(n.nspname || '.' || c.relname || ':' || c.relkind::text, ',' ORDER BY n.nspname, c.relname), '') FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='public'`).Scan(&snapshot)
		execSQL(development, "ROLLBACK")
		if e != nil {
			t.Fatal("read-only development catalog check failed")
		}
		return snapshot
	}
	before := developmentSnapshot()
	t.Cleanup(func() {
		if after := developmentSnapshot(); after != before {
			t.Error("configured development catalog changed during verification")
		} else {
			t.Log("PASS configured development public catalog unchanged; no migrator opened on it")
		}
	})
	adminCfg := base
	adminCfg.Name = "postgres"
	admin := connect(adminCfg, "m5a-test-admin")
	t.Cleanup(func() { _ = admin.Close(context.Background()) })
	if intValue(admin, "SELECT count(*) FROM pg_roles WHERE rolname=current_user AND (rolcreatedb OR rolsuper)") != 1 {
		t.Fatal("disposable database creation is unavailable")
	}
	marker := make([]byte, 16)
	if _, err := rand.Read(marker); err != nil {
		t.Fatal(err)
	}
	ownership := "m5a-owned-" + hex.EncodeToString(marker)
	dbName := "m5a_verify_" + hex.EncodeToString(marker[:8])
	if dbName == base.Name {
		t.Fatal("disposable name unexpectedly matches configured database")
	}
	// CREATE must succeed; never reuse an existing database, even on collision.
	execSQL(admin, "CREATE DATABASE "+pgx.Identifier{dbName}.Sanitize())
	var dbOID uint32
	if query(admin, "SELECT oid FROM pg_database WHERE datname=$1 AND datdba=(SELECT oid FROM pg_roles WHERE rolname=current_user)", dbName).Scan(&dbOID) != nil {
		t.Fatal("created database ownership cannot be verified; manual inspection required")
	}
	var observer *pgx.Conn
	t.Cleanup(func() {
		if observer != nil {
			_ = observer.Close(context.Background())
		}
		// Verify creation identity and owner immediately before exact-name cleanup.
		if intValue(admin, "SELECT count(*) FROM pg_database WHERE datname=$1 AND oid=$2 AND datdba=(SELECT oid FROM pg_roles WHERE rolname=current_user)", dbName, dbOID) != 1 {
			t.Error("disposable identity/owner changed; refusing cleanup")
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if _, e := admin.Exec(ctx, "DROP DATABASE "+pgx.Identifier{dbName}.Sanitize()); e != nil {
			t.Errorf("cleanup of test-owned database %s failed (details withheld)", dbName)
			return
		}
		if intValue(admin, "SELECT count(*) FROM pg_database WHERE datname=$1", dbName) != 0 {
			t.Error("disposable database still present after cleanup")
		}
		t.Logf("PASS ownership-checked cleanup: %s removed", dbName)
	})
	base.Name = dbName
	observer = connect(base, "m5a-test-observer")
	if intValue(observer, "SELECT current_setting('server_version_num')::int / 10000") != 17 {
		t.Fatal("verification requires PostgreSQL 17")
	}
	if intValue(observer, "SELECT count(*) FROM information_schema.tables WHERE table_schema='public'") != 0 {
		t.Fatal("new disposable database is not empty")
	}
	// An unrelated object must survive both down migrations and reapply.
	execSQL(observer, "CREATE TABLE m5a_unrelated_guard (marker TEXT NOT NULL)")
	execSQL(observer, "INSERT INTO m5a_unrelated_guard VALUES ($1)", ownership)
	t.Logf("Created fresh owned PostgreSQL 17 disposable database %s", dbName)

	cwd := t.TempDir()
	cfg := config.AppConfig
	cfg.Database.Name = dbName
	cfg.Database.Password = ""
	cfg.Security.JWTAuthSecretKey = ""
	content, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	// Preserve the original M5A two-version lifecycle as later milestones add SQL.
	source := t.TempDir()
	for _, name := range []string{"000001_create_users.up.sql", "000001_create_users.down.sql", "000002_create_auth_sessions.up.sql", "000002_create_auth_sessions.down.sql"} {
		data, err := os.ReadFile(filepath.Join(root, DefaultMigrationsPath, name))
		if err != nil || os.WriteFile(filepath.Join(source, name), data, 0600) != nil {
			t.Fatal("unable to prepare M5A source")
		}
	}
	if os.WriteFile(filepath.Join(cwd, "config.json"), content, 0600) != nil || os.Mkdir(filepath.Join(cwd, "migrations"), 0700) != nil || os.Symlink(source, filepath.Join(cwd, "migrations", "postgres")) != nil {
		t.Fatal("unable to prepare isolated CLI workspace")
	}
	overrides := map[string]string{"DB_NAME": dbName, "DB_PASSWORD": base.Password, "JWT_AUTH_SECRET_KEY": config.AppConfig.Security.JWTAuthSecretKey}
	env := []string{}
	for _, entry := range os.Environ() {
		key, _, _ := strings.Cut(entry, "=")
		if _, replaced := overrides[key]; !replaced {
			env = append(env, entry)
		}
	}
	for key, value := range overrides {
		env = append(env, key+"="+value)
	}
	cli := func(command, expected string) {
		t.Helper()
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		process := exec.CommandContext(ctx, binary, "migrate", command)
		process.Dir, process.Env = cwd, env
		output, e := process.CombinedOutput()
		for _, secret := range []string{base.Password, config.AppConfig.Security.JWTAuthSecretKey} {
			if secret != "" && strings.Contains(string(output), secret) {
				t.Fatal("CLI output exposed a configured secret")
			}
		}
		if e != nil || !strings.Contains(string(output), expected) {
			t.Fatalf("CLI migrate %s did not return expected success (output withheld)", command)
		}
	}
	cli("version", "Migration version: none")
	cli("up", "Migrations applied successfully")
	cli("version", "Migration version: 2 dirty: false")
	cli("up", "No pending migrations")
	t.Log("PASS actual Cobra fresh version, up, version 2 clean, repeat-up no change")

	// Exact column contracts, including sizes/nullability, catch schema drift.
	contracts := map[string][]string{
		"users":         {"id:uuid:NO", "email:character varying(255):NO", "password_hash:text:YES", "display_name:character varying(150):NO", "email_verified_at:timestamp with time zone:YES", "last_login_at:timestamp with time zone:YES", "is_active:boolean:NO", "created_at:timestamp with time zone:NO", "created_by:uuid:YES", "updated_at:timestamp with time zone:NO", "updated_by:uuid:YES"},
		"auth_sessions": {"id:uuid:NO", "user_id:uuid:NO", "token_hash:text:NO", "device_name:character varying(150):YES", "ip_address:inet:YES", "user_agent:text:YES", "expires_at:timestamp with time zone:NO", "revoked_at:timestamp with time zone:YES", "last_used_at:timestamp with time zone:YES", "created_at:timestamp with time zone:NO", "created_by:uuid:YES", "updated_at:timestamp with time zone:NO", "updated_by:uuid:YES"},
	}
	for table, expected := range contracts {
		var actual []string
		if query(observer, `SELECT array_agg(column_name || ':' || data_type || CASE WHEN character_maximum_length IS NOT NULL THEN '(' || character_maximum_length || ')' ELSE '' END || ':' || is_nullable ORDER BY ordinal_position) FROM information_schema.columns WHERE table_schema='public' AND table_name=$1`, table).Scan(&actual) != nil || strings.Join(actual, ",") != strings.Join(expected, ",") {
			t.Fatalf("%s column contract mismatch: %v", table, actual)
		}
		if intValue(observer, `SELECT count(*) FROM pg_constraint WHERE conrelid=('public.' || $1)::regclass AND contype='p' AND pg_get_constraintdef(oid)='PRIMARY KEY (id)'`, table) != 1 {
			t.Fatalf("%s UUID primary key missing", table)
		}
		if intValue(observer, `SELECT count(*) FROM information_schema.columns WHERE table_schema='public' AND table_name=$1 AND ((column_name='id' AND column_default='gen_random_uuid()') OR (column_name IN ('created_at','updated_at') AND column_default='CURRENT_TIMESTAMP'))`, table) != 3 {
			t.Fatalf("%s UUID/timestamp defaults incorrect", table)
		}
	}
	if intValue(observer, `SELECT count(*) FROM information_schema.columns WHERE table_schema='public' AND table_name='users' AND column_name='is_active' AND column_default IS NULL`) != 1 {
		t.Fatal("account activation must be explicit")
	}
	if intValue(observer, `SELECT count(*) FROM pg_constraint WHERE conrelid IN ('public.users'::regclass,'public.auth_sessions'::regclass) AND contype='f' AND confrelid='public.users'::regclass AND confdeltype='a' AND confupdtype='a' AND NOT condeferrable`) != 5 {
		t.Fatal("five conservative users FKs required")
	}
	if intValue(observer, `SELECT count(*) FROM pg_indexes WHERE schemaname='public' AND tablename IN ('users','auth_sessions')`) != 4 || intValue(observer, `SELECT count(*) FROM pg_indexes WHERE schemaname='public' AND indexname='users_email_normalized_uidx' AND indexdef LIKE 'CREATE UNIQUE INDEX%lower(btrim(%email%'`) != 1 || intValue(observer, `SELECT count(*) FROM pg_indexes WHERE schemaname='public' AND indexname='auth_sessions_user_id_idx' AND indexdef LIKE '%USING btree (user_id)'`) != 1 {
		t.Fatal("expected two PK indexes, normalized email uniqueness and user-session lookup index")
	}
	t.Log("PASS exact 11/13 column contracts, UUID PK/defaults, TIMESTAMPTZ, INET, five NO ACTION FKs and four minimum indexes")

	var userID, otherID string
	if query(observer, `INSERT INTO users(email,display_name,is_active) VALUES (' Example@Mapatu.test ', 'Schema verification', true) RETURNING id::text`).Scan(&userID) != nil || query(observer, `INSERT INTO users(email,display_name,is_active,created_by,updated_by) VALUES ('other@mapatu.test','Actor verification',false,$1,$1) RETURNING id::text`, userID).Scan(&otherID) != nil {
		t.Fatal("valid users, nullable password/audit actors or valid actor FKs rejected")
	}
	wantState := func(sql, state string, args ...any) {
		t.Helper()
		_, e := observer.Exec(context.Background(), sql, args...)
		var pgError *pgconn.PgError
		if !errors.As(e, &pgError) || pgError.Code != state {
			t.Fatalf("expected SQLSTATE %s (driver details withheld)", state)
		}
	}
	for _, email := range []string{" Example@Mapatu.test ", "example@mapatu.test", "EXAMPLE@MAPATU.TEST", "  example@mapatu.test  "} {
		wantState(`INSERT INTO users(email,display_name,is_active) VALUES ($1,'Duplicate check',true)`, "23505", email)
	}
	wantState(`UPDATE users SET email='EXAMPLE@MAPATU.TEST' WHERE id=$1`, "23505", otherID)
	wantState(`INSERT INTO users(id,email,display_name,is_active) VALUES ($1,'pk@mapatu.test','PK check',true)`, "23505", userID)
	for _, column := range []string{"id", "email", "display_name", "is_active", "created_at", "updated_at"} {
		wantState("UPDATE users SET "+pgx.Identifier{column}.Sanitize()+"=NULL WHERE id=$1", "23502", otherID)
	}
	for _, column := range []string{"created_by", "updated_by"} {
		wantState("UPDATE users SET "+pgx.Identifier{column}.Sanitize()+"='00000000-0000-4000-8000-000000000000' WHERE id=$1", "23503", otherID)
	}
	if intValue(observer, `SELECT count(*) FROM users WHERE id=$1 AND password_hash IS NULL AND created_by IS NULL AND updated_by IS NULL AND email_verified_at IS NULL AND last_login_at IS NULL AND created_at IS NOT NULL AND updated_at IS NOT NULL`, userID) != 1 {
		t.Fatal("nullable bootstrap user contract incorrect")
	}
	wantState(`DELETE FROM users WHERE id=$1`, "23503", userID)
	t.Log("PASS global normalized duplicate insert/update rejection, required user fields, nullable password/actors and valid/invalid self FKs")

	var sessionID string
	if query(observer, `INSERT INTO auth_sessions(user_id,token_hash,expires_at) VALUES ($1,$2,CURRENT_TIMESTAMP + interval '1 hour') RETURNING id::text`, userID, strings.Repeat("a", 64)).Scan(&sessionID) != nil {
		t.Fatal("valid session with nullable device/audit fields rejected")
	}
	execSQL(observer, `INSERT INTO auth_sessions(user_id,token_hash,expires_at,created_by,updated_by,device_name,ip_address,user_agent,revoked_at,last_used_at) VALUES ($1,$2,CURRENT_TIMESTAMP + interval '1 hour',$3,$3,'Schema check','2001:db8::1','Verification',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`, userID, strings.Repeat("b", 64), otherID)
	if intValue(observer, "SELECT count(*) FROM auth_sessions WHERE user_id=$1", userID) != 2 {
		t.Fatal("multiple sessions per user must be supported")
	}
	execSQL(observer, "UPDATE auth_sessions SET ip_address='192.0.2.1' WHERE id=$1", sessionID)
	execSQL(observer, "UPDATE auth_sessions SET ip_address=NULL WHERE id=$1", sessionID)
	if intValue(observer, `SELECT count(*) FROM auth_sessions WHERE id=$1 AND device_name IS NULL AND ip_address IS NULL AND user_agent IS NULL AND revoked_at IS NULL AND last_used_at IS NULL AND created_by IS NULL AND updated_by IS NULL`, sessionID) != 1 {
		t.Fatal("session nullable fields incorrect")
	}
	for _, column := range []string{"id", "user_id", "token_hash", "expires_at", "created_at", "updated_at"} {
		wantState("UPDATE auth_sessions SET "+pgx.Identifier{column}.Sanitize()+"=NULL WHERE id=$1", "23502", sessionID)
	}
	for _, column := range []string{"user_id", "created_by", "updated_by"} {
		wantState("UPDATE auth_sessions SET "+pgx.Identifier{column}.Sanitize()+"='00000000-0000-4000-8000-000000000000' WHERE id=$1", "23503", sessionID)
	}
	wantState(`INSERT INTO auth_sessions(id,user_id,token_hash,expires_at) VALUES ($1,$2,$3,CURRENT_TIMESTAMP)`, "23505", sessionID, userID, strings.Repeat("c", 64))
	// Remove only the disposable actor edge to isolate session-retention behavior.
	execSQL(observer, "UPDATE users SET created_by=NULL,updated_by=NULL WHERE id=$1", otherID)
	wantState(`DELETE FROM users WHERE id=$1`, "23503", userID)
	if intValue(observer, "SELECT count(*) FROM auth_sessions WHERE user_id=$1", userID) != 2 {
		t.Fatal("user deletion cascaded into session history")
	}
	// Without an update trigger, updated_at must remain the writer's responsibility.
	execSQL(observer, "UPDATE users SET updated_at='2000-01-01Z',display_name='Changed' WHERE id=$1", otherID)
	execSQL(observer, "UPDATE users SET display_name='Changed again' WHERE id=$1", otherID)
	if intValue(observer, "SELECT count(*) FROM users WHERE id=$1 AND updated_at='2000-01-01Z'", otherID) != 1 {
		t.Fatal("unexpected automatic updated_at behavior")
	}
	t.Log("PASS many sessions per user, hash required, nullable metadata, IPv4/IPv6, session PK/FKs, noncascading delete and writer-maintained updated_at")

	cli("down", "Migration rolled back successfully")
	cli("version", "Migration version: 1 dirty: false")
	if intValue(observer, "SELECT count(*) FROM information_schema.tables WHERE table_schema='public' AND table_name='auth_sessions'") != 0 || intValue(observer, "SELECT count(*) FROM users") != 2 {
		t.Fatal("first down did not preserve users/remove only sessions")
	}
	cli("down", "Migration rolled back successfully")
	cli("version", "Migration version: none")
	if intValue(observer, "SELECT count(*) FROM information_schema.tables WHERE table_schema='public' AND table_name IN ('users','auth_sessions')") != 0 || intValue(observer, "SELECT count(*) FROM m5a_unrelated_guard WHERE marker=$1", ownership) != 1 {
		t.Fatal("second down or unrelated object preservation failed")
	}
	cli("up", "Migrations applied successfully")
	cli("version", "Migration version: 2 dirty: false")
	if intValue(observer, "SELECT count(*) FROM users") != 0 || intValue(observer, "SELECT count(*) FROM auth_sessions") != 0 || intValue(observer, "SELECT count(*) FROM m5a_unrelated_guard WHERE marker=$1", ownership) != 1 || intValue(observer, "SELECT count(*) FROM information_schema.tables WHERE table_schema='public'") != 4 {
		t.Fatal("reapply created unexpected tables/seed data or lost unrelated object")
	}
	if intValue(observer, "SELECT count(*) FROM pg_extension WHERE extname IN ('pgcrypto','uuid-ossp','citext')") != 0 {
		t.Fatal("identity schema must not require extra extensions")
	}
	if intValue(admin, "SELECT count(*) FROM pg_stat_activity WHERE datname=$1 AND application_name='hris-core-service-migrate'", dbName) != 0 {
		t.Fatal("CLI migration connections leaked")
	}
	t.Log("PASS exact one-step down 2→1→none, unrelated object preserved, reapply to clean 2, no seed/extra schema/extensions, CLI connections closed")
}
