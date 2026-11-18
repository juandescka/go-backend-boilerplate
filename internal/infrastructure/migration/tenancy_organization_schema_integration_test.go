package migration

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
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
func TestTenancyOrganizationSchemaIntegration(t *testing.T) {
	if os.Getenv("M6_INTEGRATION") != "1" {
		t.Skip("set M6_INTEGRATION=1 and M6_CLI_BINARY for disposable schema verification")
	}
	stage := os.Getenv("M6_STAGE")
	lastVersion := 5
	if stage == "b" || stage == "full" {
		lastVersion = 9
	} else if stage != "a" {
		t.Fatal("M6_STAGE must be a, b or full")
	}
	binary, err := filepath.Abs(os.Getenv("M6_CLI_BINARY"))
	if err != nil || os.Getenv("M6_CLI_BINARY") == "" {
		t.Fatal("M6_CLI_BINARY must identify the current built backend")
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
	development := connect(base, "m6-read-only-baseline")
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
	admin := connect(adminCfg, "m6-test-admin")
	t.Cleanup(func() { _ = admin.Close(context.Background()) })
	if intValue(admin, "SELECT count(*) FROM pg_roles WHERE rolname=current_user AND (rolcreatedb OR rolsuper)") != 1 {
		t.Fatal("disposable database creation is unavailable")
	}
	marker := make([]byte, 16)
	if _, err := rand.Read(marker); err != nil {
		t.Fatal(err)
	}
	ownership := "m6-owned-" + hex.EncodeToString(marker)
	dbName := "m6_verify_" + hex.EncodeToString(marker[:8])
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
	observer = connect(base, "m6-test-observer")
	if intValue(observer, "SELECT current_setting('server_version_num')::int / 10000") != 17 {
		t.Fatal("verification requires PostgreSQL 17")
	}
	if intValue(observer, "SELECT count(*) FROM information_schema.tables WHERE table_schema='public'") != 0 {
		t.Fatal("new disposable database is not empty")
	}
	// An unrelated object must survive both down migrations and reapply.
	execSQL(observer, "CREATE TABLE m6_unrelated_guard (marker TEXT NOT NULL)")
	execSQL(observer, "INSERT INTO m6_unrelated_guard VALUES ($1)", ownership)
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
	// Stage A verifies only its dependencies; later stages use all Module A SQL.
	source := t.TempDir()
	entries, err := os.ReadDir(filepath.Join(root, DefaultMigrationsPath))
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		name := entry.Name()
		var version int
		if _, err := fmt.Sscanf(name, "%06d_", &version); err != nil || !strings.HasSuffix(name, ".sql") || version > lastVersion {
			continue
		}
		data, err := os.ReadFile(filepath.Join(root, DefaultMigrationsPath, name))
		if err != nil || os.WriteFile(filepath.Join(source, name), data, 0600) != nil {
			t.Fatal("unable to prepare stage migration source")
		}
	}
	if os.WriteFile(filepath.Join(cwd, "config.json"), content, 0600) != nil || os.Mkdir(filepath.Join(cwd, "migrations"), 0700) != nil || os.Symlink(source, filepath.Join(cwd, "migrations", "postgres")) != nil {
		t.Fatal("unable to prepare isolated CLI workspace")
	}
	overrides := map[string]string{"DB_DRIVER": "postgres", "DB_HOST": base.Host, "DB_PORT": fmt.Sprint(base.Port), "DB_NAME": dbName, "DB_USERNAME": base.Username, "DB_PASSWORD": base.Password, "DB_SSL_MODE": base.SSLMode, "JWT_AUTH_SECRET_KEY": config.AppConfig.Security.JWTAuthSecretKey}
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
	cli("version", fmt.Sprintf("Migration version: %d dirty: false", lastVersion))
	cli("up", "No pending migrations")
	t.Logf("PASS actual Cobra CLI fresh up, clean version %d and repeated up", lastVersion)

	contracts := map[string]string{
		"tenants":       "code:character varying(30):NO|name:character varying(150):NO|slug:character varying(100):NO|timezone:character varying(60):NO|deactivation_reason:text:YES|deactivated_at:timestamp with time zone:YES",
		"organizations": "tenant_id:uuid:NO|parent_organization_id:uuid:YES|code:character varying(30):NO|name:character varying(150):NO|organization_type:character varying(30):NO|legal_name:character varying(200):YES|registration_number:character varying(60):YES|email:character varying(150):YES|phone:character varying(30):YES|address:text:YES",
	}
	if lastVersion == 9 {
		contracts["branches"] = "tenant_id:uuid:NO|organization_id:uuid:NO|code:character varying(30):NO|name:character varying(150):NO|description:text:YES|is_head_office:boolean:NO|address:text:YES|latitude:numeric(10,7):YES|longitude:numeric(10,7):YES|timezone:character varying(60):NO"
		contracts["departments"] = "tenant_id:uuid:NO|organization_id:uuid:NO|parent_department_id:uuid:YES|code:character varying(30):NO|name:character varying(150):NO|description:text:YES"
		contracts["positions"] = "tenant_id:uuid:NO|organization_id:uuid:NO|code:character varying(30):NO|name:character varying(150):NO|description:text:YES|hierarchy_level:smallint:YES"
		contracts["work_locations"] = "tenant_id:uuid:NO|organization_id:uuid:NO|branch_id:uuid:YES|code:character varying(30):NO|name:character varying(150):NO|address:text:YES|latitude:numeric(10,7):YES|longitude:numeric(10,7):YES|geofence_radius_meters:integer:YES"
	}
	for table, columns := range contracts {
		columns += "|id:uuid:NO|is_active:boolean:NO|created_at:timestamp with time zone:NO|created_by:uuid:YES|updated_at:timestamp with time zone:NO|updated_by:uuid:YES"
		expected := map[string]string{}
		for _, column := range strings.Split(columns, "|") {
			name, contract, _ := strings.Cut(column, ":")
			expected[name] = contract
		}
		rows, err := observer.Query(context.Background(), `SELECT a.attname, format_type(a.atttypid,a.atttypmod) || ':' || CASE WHEN a.attnotnull THEN 'NO' ELSE 'YES' END FROM pg_attribute a WHERE a.attrelid=('public.' || $1)::regclass AND a.attnum>0 AND NOT a.attisdropped`, table)
		if err != nil {
			t.Fatal("column inspection failed")
		}
		count := 0
		for rows.Next() {
			var name, actual string
			if rows.Scan(&name, &actual) != nil || expected[name] != actual {
				t.Fatalf("%s column contract mismatch: %s %s", table, name, actual)
			}
			count++
		}
		rows.Close()
		if rows.Err() != nil || count != len(expected) {
			t.Fatalf("%s exact column count mismatch", table)
		}
		if intValue(observer, `SELECT count(*) FROM pg_constraint WHERE conrelid=('public.' || $1)::regclass AND contype='p' AND pg_get_constraintdef(oid)='PRIMARY KEY (id)'`, table) != 1 {
			t.Fatal("UUID primary key missing")
		}
		if intValue(observer, `SELECT count(*) FROM information_schema.columns WHERE table_schema='public' AND table_name=$1 AND ((column_name='id' AND column_default='gen_random_uuid()') OR (column_name='is_active' AND column_default='true') OR (column_name IN ('created_at','updated_at') AND column_default='CURRENT_TIMESTAMP'))`, table) != 4 {
			t.Fatalf("%s UUID/activation/audit defaults incorrect", table)
		}
		if intValue(observer, `SELECT count(*) FROM pg_constraint WHERE conrelid=('public.' || $1)::regclass AND contype='f' AND confrelid='users'::regclass`, table) != 2 {
			t.Fatal("audit actor FKs missing")
		}
		if intValue(observer, `SELECT count(*) FROM pg_constraint WHERE conrelid=('public.' || $1)::regclass AND contype='f' AND (confupdtype<>'a' OR confdeltype<>'a' OR condeferrable)`, table) != 0 {
			t.Fatal("unexpected cascading or deferred FK")
		}
	}
	if stage == "full" {
		// Contract indexes: six PKs, ten scoped/global unique keys and three optional-edge indexes.
		if intValue(observer, `SELECT count(*) FROM pg_indexes WHERE schemaname='public' AND tablename IN ('tenants','organizations','branches','departments','positions','work_locations')`) != 19 {
			t.Fatal("unexpected redundant or missing Module A indexes")
		}
		if intValue(observer, `SELECT count(*) FROM pg_constraint WHERE conrelid IN ('tenants'::regclass,'organizations'::regclass,'branches'::regclass,'departments'::regclass,'positions'::regclass,'work_locations'::regclass) AND contype='f'`) != 24 {
			t.Fatal("missing or extra Module A foreign keys")
		}
		if intValue(observer, `SELECT count(*) FROM information_schema.tables WHERE table_schema='public'`) != 10 {
			t.Fatal("unexpected business or metadata table")
		}
		if intValue(observer, `SELECT count(*) FROM pg_trigger WHERE NOT tgisinternal AND tgrelid IN ('tenants'::regclass,'organizations'::regclass,'branches'::regclass,'departments'::regclass,'positions'::regclass,'work_locations'::regclass)`) != 0 {
			t.Fatal("unexpected application triggers")
		}
		t.Log("PASS integrated catalog: exactly six Module A tables, 19 purposeful indexes, 24 conservative FKs and no user triggers")
	}
	wantState := func(sql, state string, args ...any) {
		t.Helper()
		_, e := observer.Exec(context.Background(), sql, args...)
		var pgError *pgconn.PgError
		if !errors.As(e, &pgError) || pgError.Code != state {
			t.Fatalf("expected SQLSTATE %s (details withheld)", state)
		}
	}
	id := func(sql string, args ...any) string {
		t.Helper()
		var value string
		if query(observer, sql, args...).Scan(&value) != nil {
			t.Fatal("valid fixture insert rejected (details withheld)")
		}
		return value
	}
	tenantA := id(`INSERT INTO tenants(code,name,slug,timezone) VALUES ('A','Shared name','a','Asia/Makassar') RETURNING id::text`)
	tenantB := id(`INSERT INTO tenants(code,name,slug,timezone) VALUES ('B','Shared name','b','UTC') RETURNING id::text`)
	orgA := id(`INSERT INTO organizations(tenant_id,code,name,organization_type) VALUES ($1,'SHARED','A','COMPANY') RETURNING id::text`, tenantA)
	orgB := id(`INSERT INTO organizations(tenant_id,code,name,organization_type) VALUES ($1,'SHARED','B','SCHOOL') RETURNING id::text`, tenantB)
	orgChild := id(`INSERT INTO organizations(tenant_id,parent_organization_id,code,name,organization_type) VALUES ($1,$2,'CHILD','Child','OTHER') RETURNING id::text`, tenantA, orgA)
	for _, kind := range []string{"GOVERNMENT", "FOUNDATION"} {
		execSQL(observer, `INSERT INTO organizations(tenant_id,code,name,organization_type) VALUES ($1,$2,'Type',$2)`, tenantB, kind)
	}
	wantState(`INSERT INTO tenants(code,name,slug,timezone) VALUES ('A','Duplicate','new','UTC')`, "23505")
	wantState(`INSERT INTO tenants(code,name,slug,timezone) VALUES ('NEW','Duplicate','a','UTC')`, "23505")
	wantState(`UPDATE tenants SET code='A' WHERE id=$1`, "23505", tenantB)
	wantState(`UPDATE tenants SET slug='a' WHERE id=$1`, "23505", tenantB)
	// No unapproved lower/trim normalization or global organization-code uniqueness.
	execSQL(observer, `INSERT INTO tenants(code,name,slug,timezone) VALUES ('a','Shared name','A','UTC')`)
	wantState(`INSERT INTO organizations(tenant_id,code,name,organization_type) VALUES ($1,'SHARED','Duplicate','OTHER')`, "23505", tenantA)
	wantState(`UPDATE organizations SET code='SHARED' WHERE id=$1`, "23505", orgChild)
	wantState(`UPDATE organizations SET organization_type='UNKNOWN' WHERE id=$1`, "23514", orgA)
	wantState(`INSERT INTO organizations(tenant_id,parent_organization_id,code,name,organization_type) VALUES ($1,$2,'CROSS','Cross','OTHER')`, "23503", tenantA, orgB)
	wantState(`UPDATE organizations SET parent_organization_id=$1 WHERE id=$2`, "23503", orgB, orgChild)
	wantState(`UPDATE organizations SET parent_organization_id=id WHERE id=$1`, "23514", orgA)
	wantState(`UPDATE organizations SET tenant_id=$1 WHERE id=$2`, "23503", tenantB, orgChild)
	wantState(`UPDATE organizations SET tenant_id=$1,code='MOVED' WHERE id=$2`, "23503", tenantB, orgA)
	wantState(`DELETE FROM tenants WHERE id=$1`, "23503", tenantA)
	wantState(`DELETE FROM organizations WHERE id=$1`, "23503", orgA)
	if lastVersion == 9 {
		idsA, idsB, idsOther := map[string]string{}, map[string]string{}, map[string]string{}
		for _, table := range []string{"branches", "departments", "positions", "work_locations"} {
			quoted := pgx.Identifier{table}.Sanitize()
			extraColumns, extraValues := "", ""
			if table == "branches" {
				extraColumns, extraValues = ",timezone", ",'UTC'"
			}
			insert := "INSERT INTO " + quoted + "(tenant_id,organization_id,code,name" + extraColumns + ") VALUES ($1,$2,$3,'Fixture'" + extraValues + ")"
			idsA[table] = id(insert+" RETURNING id::text", tenantA, orgA, "SHARED")
			idsB[table] = id(insert+" RETURNING id::text", tenantB, orgB, "SHARED")
			idsOther[table] = id(insert+" RETURNING id::text", tenantA, orgChild, "SHARED")
			wantState(insert, "23505", tenantA, orgA, "SHARED")
			wantState(insert, "23503", tenantA, orgB, "CROSS")
			wantState("UPDATE "+quoted+" SET tenant_id=$1 WHERE id=$2", "23503", tenantB, idsA[table])
			wantState("UPDATE "+quoted+" SET organization_id=$1 WHERE id=$2", "23503", orgB, idsA[table])
			wantState("UPDATE "+quoted+" SET organization_id=$1 WHERE id=$2", "23505", orgA, idsOther[table])
		}
		branchA, branchB, branchOther := idsA["branches"], idsB["branches"], idsOther["branches"]
		if intValue(observer, `SELECT count(*) FROM branches WHERE is_head_office`) != 0 {
			t.Fatal("head-office default must be false")
		}
		execSQL(observer, `UPDATE branches SET is_head_office=true WHERE id=$1`, branchA)
		execSQL(observer, `INSERT INTO branches(tenant_id,organization_id,code,name,timezone,is_head_office) VALUES ($1,$2,'HEAD2','Regional head','UTC',true)`, tenantA, orgA)
		if intValue(observer, `SELECT count(*) FROM branches WHERE tenant_id=$1 AND organization_id=$2 AND is_head_office`, tenantA, orgA) != 2 {
			t.Fatal("multiple head offices must be allowed")
		}
		deptChild := id(`INSERT INTO departments(tenant_id,organization_id,parent_department_id,code,name) VALUES ($1,$2,$3,'CHILD','Child') RETURNING id::text`, tenantA, orgA, idsA["departments"])
		for _, parent := range []string{idsB["departments"], idsOther["departments"]} {
			wantState(`INSERT INTO departments(tenant_id,organization_id,parent_department_id,code,name) VALUES ($1,$2,$3,'CROSS','Cross')`, "23503", tenantA, orgA, parent)
			wantState(`UPDATE departments SET parent_department_id=$1 WHERE id=$2`, "23503", parent, deptChild)
		}
		wantState(`UPDATE departments SET parent_department_id=id WHERE id=$1`, "23514", deptChild)
		wantState(`UPDATE departments SET organization_id=$1 WHERE id=$2`, "23503", orgChild, deptChild)
		wantState(`DELETE FROM departments WHERE id=$1`, "23503", idsA["departments"])
		if intValue(observer, `SELECT count(*) FROM work_locations WHERE id=$1 AND branch_id IS NULL`, idsA["work_locations"]) != 1 {
			t.Fatal("organization-level location must allow null branch")
		}
		execSQL(observer, `UPDATE work_locations SET branch_id=$1 WHERE id=$2`, branchA, idsA["work_locations"])
		execSQL(observer, `UPDATE work_locations SET branch_id=$1 WHERE id=$2`, branchB, idsB["work_locations"])
		for _, branch := range []string{branchB, branchOther} {
			wantState(`INSERT INTO work_locations(tenant_id,organization_id,branch_id,code,name) VALUES ($1,$2,$3,'CROSS','Cross')`, "23503", tenantA, orgA, branch)
			wantState(`UPDATE work_locations SET branch_id=$1 WHERE id=$2`, "23503", branch, idsA["work_locations"])
		}
		wantState(`UPDATE branches SET organization_id=$1,code='MOVED' WHERE id=$2`, "23503", orgChild, branchA)
		wantState(`DELETE FROM branches WHERE id=$1`, "23503", branchA)
		wantState(`UPDATE positions SET hierarchy_level=0`, "23514")
		wantState(`UPDATE positions SET hierarchy_level=-1`, "23514")
		execSQL(observer, `UPDATE positions SET hierarchy_level=1`)
		execSQL(observer, `UPDATE positions SET hierarchy_level=NULL`)
		for _, table := range []string{"branches", "work_locations"} {
			quoted := pgx.Identifier{table}.Sanitize()
			execSQL(observer, "UPDATE "+quoted+" SET latitude=-90,longitude=-180")
			execSQL(observer, "UPDATE "+quoted+" SET latitude=90,longitude=180")
			for _, invalid := range []string{"latitude=-90.0000001", "latitude=90.0000001", "longitude=-180.0000001", "longitude=180.0000001", "latitude='NaN'", "longitude='NaN'"} {
				wantState("UPDATE "+quoted+" SET "+invalid, "23514")
			}
			execSQL(observer, "UPDATE "+quoted+" SET latitude=NULL,longitude=0")
			execSQL(observer, "UPDATE "+quoted+" SET longitude=NULL")
		}
		wantState(`UPDATE work_locations SET geofence_radius_meters=0`, "23514")
		wantState(`UPDATE work_locations SET geofence_radius_meters=-1`, "23514")
		execSQL(observer, `UPDATE work_locations SET geofence_radius_meters=1`)
		execSQL(observer, `UPDATE work_locations SET geofence_radius_meters=NULL`)
		// Self-parent checks do not prove recursive acyclicity; expose this limit explicitly.
		execSQL(observer, `UPDATE departments SET parent_department_id=$1 WHERE id=$2`, deptChild, idsA["departments"])
		execSQL(observer, `UPDATE departments SET parent_department_id=NULL WHERE id=$1`, idsA["departments"])
		t.Log("PASS four unit tables across two tenants/organizations, scoped duplicates, invalid cross-scope insert/update/ownership rejection, optional branch, two head offices, department parents, hierarchy and coordinate/radius bounds")
		t.Log("LIMIT verified: longer department cycles are not database-enforced; restored fixture immediately")
	}
	actor := id(`INSERT INTO users(email,display_name,is_active) VALUES ('m6@mapatu.test','M6 actor',true) RETURNING id::text`)
	session := id(`INSERT INTO auth_sessions(user_id,token_hash,expires_at) VALUES ($1,'m6-session-hash',CURRENT_TIMESTAMP+interval '1 hour') RETURNING id::text`, actor)
	for table := range contracts {
		quoted := pgx.Identifier{table}.Sanitize()
		for _, column := range []string{"created_by", "updated_by"} {
			wantState("UPDATE "+quoted+" SET "+column+"='00000000-0000-4000-8000-000000000000'", "23503")
			execSQL(observer, "UPDATE "+quoted+" SET "+column+"=$1", actor)
		}
		for _, column := range []string{"id", "is_active", "created_at", "updated_at", "code", "name"} {
			wantState("UPDATE "+quoted+" SET "+column+"=NULL", "23502")
		}
		if intValue(observer, "SELECT count(*) FROM "+quoted+" WHERE NOT is_active OR created_at IS NULL OR updated_at IS NULL") != 0 {
			t.Fatal("master activation/audit defaults not applied")
		}
	}
	wantState(`DELETE FROM users WHERE id=$1`, "23503", actor)
	execSQL(observer, `UPDATE tenants SET is_active=false,deactivation_reason='Test',deactivated_at=CURRENT_TIMESTAMP,updated_at='2000-01-01Z' WHERE id=$1`, tenantA)
	execSQL(observer, `UPDATE tenants SET name='Updated' WHERE id=$1`, tenantA)
	if intValue(observer, `SELECT count(*) FROM tenants WHERE id=$1 AND updated_at='2000-01-01Z' AND NOT is_active AND deactivation_reason='Test' AND deactivated_at IS NOT NULL`, tenantA) != 1 {
		t.Fatal("deactivation or writer-maintained timestamp contract failed")
	}
	t.Log("PASS exact schema/defaults, two tenants, tenant code/slug uniqueness, scoped organization codes, allowed types, parent tenant/self/ownership rejection, audit actors and noncascading deletes")

	// Known boundary: longer hierarchy cycles require future concurrent-safe write validation.
	execSQL(observer, `UPDATE organizations SET parent_organization_id=$1 WHERE id=$2`, orgChild, orgA)
	execSQL(observer, `UPDATE organizations SET parent_organization_id=NULL WHERE id=$1`, orgA)
	t.Log("LIMIT verified: longer organization cycles are not database-enforced; restored fixture immediately")

	for version := lastVersion; version >= 4; version-- {
		cli("down", "Migration rolled back successfully")
		cli("version", fmt.Sprintf("Migration version: %d dirty: false", version-1))
		if stage == "full" {
			dropped := map[int]string{9: "work_locations", 8: "positions", 7: "departments", 6: "branches", 5: "organizations", 4: "tenants"}
			if intValue(observer, `SELECT count(*) FROM information_schema.tables WHERE table_schema='public' AND table_name=$1`, dropped[version]) != 0 || intValue(observer, `SELECT count(*) FROM information_schema.tables WHERE table_schema='public' AND table_name IN ('tenants','organizations','branches','departments','positions','work_locations')`) != version-4 {
				t.Fatal("one-step down removed unexpected tables")
			}
		}

		if intValue(observer, `SELECT count(*) FROM users WHERE id=$1`, actor) != 1 || intValue(observer, `SELECT count(*) FROM auth_sessions WHERE id=$1`, session) != 1 || intValue(observer, `SELECT count(*) FROM m6_unrelated_guard WHERE marker=$1`, ownership) != 1 {
			t.Fatal("Module B data or unrelated guard lost on rollback")
		}
	}
	if intValue(observer, `SELECT count(*) FROM information_schema.tables WHERE table_schema='public' AND table_name IN ('tenants','organizations','branches','departments','positions','work_locations')`) != 0 {
		t.Fatal("Module A tables remain after rollback to version 3")
	}
	cli("up", "Migrations applied successfully")
	cli("version", fmt.Sprintf("Migration version: %d dirty: false", lastVersion))
	for table := range contracts {
		if intValue(observer, "SELECT count(*) FROM "+pgx.Identifier{table}.Sanitize()) != 0 {
			t.Fatal("unexpected seed data after reapply")
		}
	}
	if intValue(observer, `SELECT count(*) FROM m6_unrelated_guard WHERE marker=$1`, ownership) != 1 || intValue(observer, `SELECT count(*) FROM users WHERE id=$1`, actor) != 1 || intValue(observer, `SELECT count(*) FROM auth_sessions WHERE id=$1`, session) != 1 {
		t.Fatal("reapply changed unrelated or Module B data")
	}
	if intValue(observer, `SELECT count(*) FROM pg_indexes WHERE schemaname='public' AND indexname='auth_sessions_token_hash_uidx'`) != 1 {
		t.Fatal("Module B token hash index lost")
	}
	if intValue(observer, `SELECT count(*) FROM pg_extension WHERE extname IN ('pgcrypto','uuid-ossp','citext')`) != 0 {
		t.Fatal("unexpected extension requirement")
	}
	if intValue(admin, `SELECT count(*) FROM pg_stat_activity WHERE datname=$1 AND application_name='hris-core-service-migrate'`, dbName) != 0 {
		t.Fatal("migration connections leaked")
	}
	t.Log("PASS one-step rollback through Module A only, Module B rows/index and unrelated guard preserved, clean reapply, no seed/extensions or leaked migration connections")
}
