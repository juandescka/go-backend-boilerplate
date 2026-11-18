package migration

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-migrate/migrate/v4"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/stdlib"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"hris-core-service/config"
	"hris-core-service/internal/domain/models"
	authstore "hris-core-service/internal/infrastructure/authentication"
	accessstore "hris-core-service/internal/infrastructure/authorization"
	"hris-core-service/internal/middlewares"
	"hris-core-service/internal/shared/password"
	auth "hris-core-service/internal/usecases/authentication"
	access "hris-core-service/internal/usecases/authorization"
)

// Only a fresh owned database is migrated; the development target is never opened.
func TestTenantRBACPostgresIntegration(t *testing.T) {
	if os.Getenv("M7_INTEGRATION") != "1" {
		t.Skip("set M7_INTEGRATION=1 for disposable PostgreSQL17 verification")
	}
	stage := os.Getenv("M7_STAGE")
	last := 10
	if stage == "full" {
		last = 15
	} else if stage == "b" {
		last = 14
	} else if stage != "a" {
		t.Fatal("M7_STAGE must be a, b or full")
	}
	t.Chdir("../../..")
	previous := config.AppConfig
	t.Cleanup(func() { config.AppConfig = previous })
	if config.LoadConfig() != nil {
		t.Fatal("configuration unavailable; details withheld")
	}
	base := config.AppConfig.Database
	t.Logf("Configured target %s:%d/%s is not migrated; secrets withheld", base.Host, base.Port, base.Name)
	connect := func(c config.DatabaseConfig) *pgx.Conn {
		t.Helper()
		parsed, err := pgx.ParseConfig(migrationDatabaseURL(c))
		if err != nil {
			t.Fatal("invalid database configuration")
		}
		parsed.RuntimeParams["statement_timeout"] = "5000"
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		conn, err := pgx.ConnectConfig(ctx, parsed)
		if err != nil {
			t.Fatal("database connection unavailable; details withheld")
		}
		return conn
	}
	adminConfig := base
	adminConfig.Name = "postgres"
	admin := connect(adminConfig)
	t.Cleanup(func() { _ = admin.Close(context.Background()) })
	name := "m7_verify_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if name == base.Name {
		t.Fatal("target collision")
	}
	if _, err := admin.Exec(context.Background(), "CREATE DATABASE "+pgx.Identifier{name}.Sanitize()); err != nil {
		t.Fatal("disposable creation failed; details withheld")
	}
	var oid uint32
	if admin.QueryRow(context.Background(), "SELECT oid FROM pg_database WHERE datname=$1 AND datdba=(SELECT oid FROM pg_roles WHERE rolname=current_user)", name).Scan(&oid) != nil {
		t.Fatal("created ownership unavailable; inspect manually")
	}
	var conn *pgx.Conn
	var runner *Runner
	t.Cleanup(func() {
		if runner != nil {
			if err := runner.Close(); err != nil {
				t.Error("migrator cleanup failed")
			}
		}
		if conn != nil {
			_ = conn.Close(context.Background())
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		var matches int
		if admin.QueryRow(ctx, "SELECT count(*) FROM pg_database WHERE datname=$1 AND oid=$2 AND datdba=(SELECT oid FROM pg_roles WHERE rolname=current_user)", name, oid).Scan(&matches) != nil || matches != 1 {
			t.Error("ownership changed; refusing cleanup")
			return
		}
		if _, err := admin.Exec(ctx, "DROP DATABASE "+pgx.Identifier{name}.Sanitize()); err != nil {
			t.Errorf("owned disposable %s cleanup failed; details withheld", name)
			return
		}
		t.Logf("PASS ownership/OID-checked disposable cleanup: %s", name)
	})
	base.Name = name
	conn = connect(base)
	scalar := func(query string, args ...any) int {
		t.Helper()
		var n int
		if conn.QueryRow(context.Background(), query, args...).Scan(&n) != nil {
			t.Fatal("verification query failed; details withheld")
		}
		return n
	}
	execSQL := func(query string, args ...any) {
		t.Helper()
		if _, err := conn.Exec(context.Background(), query, args...); err != nil {
			t.Fatal("verification SQL failed; details withheld")
		}
	}
	reject := func(query, state string, args ...any) {
		t.Helper()
		_, err := conn.Exec(context.Background(), query, args...)
		var pgErr *pgconn.PgError
		if !errors.As(err, &pgErr) || pgErr.Code != state {
			t.Fatalf("expected SQLSTATE %s; details withheld", state)
		}
	}
	id := func(query string, args ...any) string {
		t.Helper()
		var value string
		if conn.QueryRow(context.Background(), query, args...).Scan(&value) != nil {
			t.Fatal("fixture insert failed; details withheld")
		}
		return value
	}
	if scalar("SELECT current_setting('server_version_num')::int/10000") != 17 || scalar("SELECT count(*) FROM information_schema.tables WHERE table_schema='public'") != 0 {
		t.Fatal("fresh PostgreSQL17 required")
	}
	source := t.TempDir()
	entries, err := os.ReadDir(DefaultMigrationsPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		var version int
		if _, err := fmt.Sscanf(entry.Name(), "%06d_", &version); err != nil || !strings.HasSuffix(entry.Name(), ".sql") || version > last {
			continue
		}
		data, err := os.ReadFile(filepath.Join(DefaultMigrationsPath, entry.Name()))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(source, entry.Name()), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	runner, err = NewRunner(Config{SourceURL: migrationSourceURL(source), DatabaseURL: migrationDatabaseURL(base)})
	if err != nil {
		t.Fatal("migrator initialization failed")
	}
	if runner.Up() != nil {
		t.Fatal("migration up failed")
	}
	if version, dirty, err := runner.Version(); err != nil || dirty || version != last {
		t.Fatal("unexpected migration version")
	}
	if err := runner.Up(); !errors.Is(err, migrate.ErrNoChange) {
		t.Fatal("repeat up must report no change")
	}
	verifyM7Catalog(t, conn, last)
	if stage == "full" {
		verifyPolicyCatalog(t, conn)
	}
	userA := id("INSERT INTO users(email,display_name,is_active) VALUES ('a@test.invalid','A',true) RETURNING id::text")
	userB := id("INSERT INTO users(email,display_name,is_active) VALUES ('b@test.invalid','B',true) RETURNING id::text")
	tenantA := id("INSERT INTO tenants(code,name,slug,timezone) VALUES ('A','A','a','UTC') RETURNING id::text")
	tenantB := id("INSERT INTO tenants(code,name,slug,timezone) VALUES ('B','B','b','UTC') RETURNING id::text")
	membership := id("INSERT INTO tenant_memberships(tenant_id,user_id,created_by,updated_by) VALUES ($1,$2,$2,$2) RETURNING id::text", tenantA, userA)
	execSQL("INSERT INTO tenant_memberships(tenant_id,user_id) VALUES ($1,$2),($1,$3)", tenantB, userA, userB)
	reject("INSERT INTO tenant_memberships(tenant_id,user_id) VALUES ($1,$2)", "23505", tenantA, userA)
	reject("INSERT INTO tenant_memberships(tenant_id,user_id) VALUES ($1,$2)", "23503", tenantA, uuid.NewString())
	reject("INSERT INTO tenant_memberships(tenant_id,user_id) VALUES ($1,$2)", "23503", uuid.NewString(), userA)
	for _, column := range []string{"created_by", "updated_by"} {
		reject("UPDATE tenant_memberships SET "+pgx.Identifier{column}.Sanitize()+"=$1 WHERE id=$2", "23503", uuid.NewString(), membership)
	}
	execSQL("UPDATE tenant_memberships SET is_active=false WHERE id=$1", membership)
	if scalar("SELECT count(*) FROM tenant_memberships WHERE id=$1 AND NOT is_active AND created_by=$2 AND updated_by=$2 AND created_at IS NOT NULL AND updated_at IS NOT NULL", membership, userA) != 1 || scalar("SELECT count(*) FROM users WHERE id=$1 AND is_active", userA) != 1 {
		t.Fatal("membership deactivation changed identity or lost audit")
	}
	execSQL("UPDATE tenant_memberships SET is_active=true WHERE id=$1", membership)
	reject("DELETE FROM tenants WHERE id=$1", "23503", tenantA)
	reject("DELETE FROM users WHERE id=$1", "23503", userA)
	if scalar("SELECT count(*) FROM information_schema.columns WHERE table_name='tenant_memberships' AND table_schema='public'") != 8 {
		t.Fatal("membership column drift")
	}
	t.Log("PASS multi-tenant memberships, shared tenant, uniqueness, user/tenant/actor FKs, activation independence and retained audit")
	if last >= 14 {
		verifyRBACSchema(t, conn, userA, userB, tenantA, tenantB)
	}
	if stage == "full" {
		// A grant added after 000015 must abort the whole rollback, including
		// the deletion of the original pairs. Run the SQL outside the runner
		// to leave migration metadata available for the successful down.
		down, err := os.ReadFile(filepath.Join(source, "000015_seed_rbac_policy_v1.down.sql"))
		if err != nil {
			t.Fatal(err)
		}
		const extraGrant = `SELECT r.id, p.id FROM roles r, permissions p
			WHERE r.code = 'EMPLOYEE' AND p.code = 'branch.read'`
		execSQL("INSERT INTO role_permissions (role_id, permission_id) " + extraGrant)
		reject(string(down), "23503")
		if scalar("SELECT count(*) FROM permissions") != 9 || scalar("SELECT count(*) FROM role_permissions") != 21 {
			t.Fatal("failed rollback changed reference data")
		}
		execSQL("DELETE FROM role_permissions WHERE (role_id, permission_id) IN (" + extraGrant + ")")
	}
	for version := last; version > 9; version-- {
		assignments := 0
		if version == 15 {
			assignments = scalar("SELECT count(*) FROM role_assignments")
		}
		if runner.Down() != nil {
			t.Fatal("rollback failed")
		}
		if got, dirty, err := runner.Version(); err != nil || dirty || got != version-1 {
			t.Fatal("one-step rollback version mismatch")
		}
		if version == 15 {
			if scalar("SELECT count(*) FROM roles") != 6 ||
				scalar("SELECT count(*) FROM role_assignments") != assignments ||
				scalar("SELECT count(*) FROM permissions") != 1 ||
				scalar("SELECT count(*) FROM role_permissions rp JOIN permissions p ON p.id=rp.permission_id WHERE p.code='fixture.inspect'") != 1 ||
				scalar("SELECT count(*) FROM role_permissions") != 1 {
				t.Fatal("policy rollback removed unrelated data or left seeded grants")
			}
			t.Log("PASS policy rollback preserves roles, assignments and unrelated grants, and refuses later dependent mappings")
		}
	}
	if scalar("SELECT count(*) FROM users WHERE id IN ($1,$2)", userA, userB) != 2 || scalar("SELECT count(*) FROM tenants") != 2 || scalar("SELECT count(*) FROM pg_indexes WHERE indexname='auth_sessions_token_hash_uidx'") != 1 {
		t.Fatal("M7 rollback affected M5/M6")
	}
	if runner.Up() != nil {
		t.Fatal("reapply failed")
	}
	if scalar("SELECT count(*) FROM tenant_memberships") != 0 {
		t.Fatal("membership seed forbidden")
	}
	if last == 14 && (scalar("SELECT count(*) FROM roles") != 6 || scalar("SELECT count(*) FROM permissions") != 0 || scalar("SELECT count(*) FROM role_permissions") != 0 || scalar("SELECT count(*) FROM role_assignments") != 0) {
		t.Fatal("unexpected bootstrap privileges")
	}
	if stage == "full" {
		verifyPolicyCatalog(t, conn)
	}
	t.Logf("PASS migrations up/repeat/down%d-to9/reapply, M5/M6 preserved, no membership or user assignment seeded", last)
}

func verifyRBACSchema(t *testing.T, conn *pgx.Conn, userA, userB, tenantA, tenantB string) {
	t.Helper()
	scalar := func(query string, args ...any) int {
		t.Helper()
		var n int
		if conn.QueryRow(context.Background(), query, args...).Scan(&n) != nil {
			t.Fatal("RBAC query failed; details withheld")
		}
		return n
	}
	execSQL := func(query string, args ...any) {
		t.Helper()
		if _, err := conn.Exec(context.Background(), query, args...); err != nil {
			t.Fatal("RBAC SQL failed; details withheld")
		}
	}
	reject := func(query, state string, args ...any) {
		t.Helper()
		_, err := conn.Exec(context.Background(), query, args...)
		var pgErr *pgconn.PgError
		if !errors.As(err, &pgErr) || pgErr.Code != state {
			t.Fatalf("expected RBAC SQLSTATE %s; details withheld", state)
		}
	}
	id := func(query string, args ...any) string {
		t.Helper()
		var value string
		if conn.QueryRow(context.Background(), query, args...).Scan(&value) != nil {
			t.Fatal("RBAC fixture failed; details withheld")
		}
		return value
	}
	super := "00000000-0000-4000-8000-000000000001"
	tenantAdmin := "00000000-0000-4000-8000-000000000002"
	hrAdmin := "00000000-0000-4000-8000-000000000003"
	if scalar("SELECT count(*) FROM roles") != 6 || scalar("SELECT count(*) FROM roles WHERE (code='SUPER_ADMIN' AND role_scope='PLATFORM') OR (code IN ('TENANT_ADMIN','HR_ADMIN','MANAGER','SUPERVISOR','EMPLOYEE') AND role_scope='TENANT')") != 6 {
		t.Fatal("fixed role catalog mismatch")
	}
	if scalar("SELECT count(*) FROM role_assignments") != 0 {
		t.Fatal("no initial user assignments permitted")
	}
	if os.Getenv("M7_STAGE") == "b" && (scalar("SELECT count(*) FROM permissions") != 0 || scalar("SELECT count(*) FROM role_permissions") != 0) {
		t.Fatal("schema-only migrations seeded permissions before policy")
	}
	reject("UPDATE roles SET role_scope='TENANT' WHERE id=$1", "23514", super)
	reject("UPDATE roles SET id=$1 WHERE id=$2", "23514", uuid.NewString(), super)
	reject("UPDATE roles SET code='CUSTOM' WHERE id=$1", "23514", tenantAdmin)
	reject("INSERT INTO roles(id,code,name,role_scope) VALUES ($1,'CUSTOM','Custom','TENANT')", "23514", uuid.NewString())
	permission := id("INSERT INTO permissions(code,name) VALUES ('fixture.inspect','Disposable mechanics only') RETURNING id::text")
	reject("INSERT INTO permissions(code,name) VALUES ('fixture.inspect','Duplicate')", "23505")
	reject("INSERT INTO permissions(code,name) VALUES ('INVALID','Invalid')", "23514")
	execSQL("INSERT INTO role_permissions(role_id,permission_id) VALUES ($1,$2)", tenantAdmin, permission)
	reject("INSERT INTO role_permissions(role_id,permission_id) VALUES ($1,$2)", "23505", tenantAdmin, permission)
	reject("INSERT INTO role_permissions(role_id,permission_id) VALUES ($1,$2)", "23503", uuid.NewString(), permission)
	reject("INSERT INTO role_permissions(role_id,permission_id) VALUES ($1,$2)", "23503", tenantAdmin, uuid.NewString())
	orgA := id("INSERT INTO organizations(tenant_id,code,name,organization_type) VALUES ($1,'A','A','COMPANY') RETURNING id::text", tenantA)
	orgA2 := id("INSERT INTO organizations(tenant_id,code,name,organization_type) VALUES ($1,'A2','A2','COMPANY') RETURNING id::text", tenantA)
	orgB := id("INSERT INTO organizations(tenant_id,code,name,organization_type) VALUES ($1,'B','B','COMPANY') RETURNING id::text", tenantB)
	branchA := id("INSERT INTO branches(tenant_id,organization_id,code,name,timezone) VALUES ($1,$2,'A','A','UTC') RETURNING id::text", tenantA, orgA)
	branchA2 := id("INSERT INTO branches(tenant_id,organization_id,code,name,timezone) VALUES ($1,$2,'A2','A2','UTC') RETURNING id::text", tenantA, orgA2)
	branchB := id("INSERT INTO branches(tenant_id,organization_id,code,name,timezone) VALUES ($1,$2,'B','B','UTC') RETURNING id::text", tenantB, orgB)
	outsider := id("INSERT INTO users(email,display_name,is_active) VALUES ('platform@test.invalid','Platform',true) RETURNING id::text")
	insert := "INSERT INTO role_assignments(user_id,role_id,scope_type,tenant_id,organization_id,branch_id) VALUES ($1,$2,$3,$4,$5,$6)"
	cases := []struct {
		kind                string
		tenant, org, branch any
		role                string
	}{{"PLATFORM", nil, nil, nil, super}, {"TENANT", tenantA, nil, nil, tenantAdmin}, {"ORGANIZATION", tenantA, orgA, nil, tenantAdmin}, {"BRANCH", tenantA, orgA, branchA, tenantAdmin}}
	for _, c := range cases {
		user := userA
		if c.kind == "PLATFORM" {
			user = outsider
		}
		execSQL(insert, user, c.role, c.kind, c.tenant, c.org, c.branch)
		reject(insert, "23505", user, c.role, c.kind, c.tenant, c.org, c.branch)
	}
	execSQL(insert, userA, hrAdmin, "TENANT", tenantA, nil, nil)
	execSQL(insert, userA, hrAdmin, "TENANT", tenantB, nil, nil)
	for _, c := range []struct {
		kind                string
		tenant, org, branch any
	}{{"INVALID", nil, nil, nil}, {"PLATFORM", tenantA, nil, nil}, {"TENANT", nil, nil, nil}, {"TENANT", tenantA, orgA, nil}, {"ORGANIZATION", tenantA, nil, nil}, {"ORGANIZATION", tenantA, orgA, branchA}, {"BRANCH", tenantA, orgA, nil}} {
		role := tenantAdmin
		if c.kind == "PLATFORM" {
			role = super
		}
		reject(insert, "23514", userA, role, c.kind, c.tenant, c.org, c.branch)
	}
	reject(insert, "23514", userA, tenantAdmin, "PLATFORM", nil, nil, nil)
	reject(insert, "23514", userA, super, "TENANT", tenantA, nil, nil)
	reject(insert, "23503", userB, tenantAdmin, "TENANT", tenantA, nil, nil)
	reject(insert, "23503", uuid.NewString(), super, "PLATFORM", nil, nil, nil)
	reject(insert, "23503", userA, tenantAdmin, "ORGANIZATION", tenantA, orgB, nil)
	reject(insert, "23503", userA, tenantAdmin, "BRANCH", tenantA, orgA, branchB)
	reject(insert, "23503", userA, tenantAdmin, "BRANCH", tenantA, orgA, branchA2)
	reject("UPDATE role_assignments SET organization_id=$1 WHERE scope_type='BRANCH' AND user_id=$2", "23503", orgA2, userA)
	reject("UPDATE role_assignments SET tenant_id=$1 WHERE scope_type='ORGANIZATION' AND user_id=$2", "23503", tenantB, userA)
	reject("UPDATE branches SET organization_id=$1 WHERE id=$2", "23503", orgA2, branchA)
	reject("DELETE FROM tenant_memberships WHERE tenant_id=$1 AND user_id=$2", "23503", tenantA, userA)
	execSQL("UPDATE role_assignments SET is_active=false WHERE scope_type='BRANCH' AND user_id=$1", userA)
	reject(insert, "23505", userA, tenantAdmin, "BRANCH", tenantA, orgA, branchA)
	execSQL("UPDATE role_assignments SET is_active=true WHERE scope_type='BRANCH' AND user_id=$1", userA)
	if scalar("SELECT count(*) FROM information_schema.columns WHERE table_schema='public' AND table_name='role_assignments'") != 12 || scalar("SELECT count(*) FROM information_schema.columns WHERE table_schema='public' AND table_name='role_assignments' AND column_name IN ('assigned_at','expires_at','expired_at','role_scope','tenant_membership_id')") != 0 {
		t.Fatal("assignment column drift")
	}
	if scalar("SELECT count(*) FROM information_schema.columns WHERE table_schema='public' AND table_name IN ('roles','permissions','role_permissions') AND column_name='tenant_id'") != 0 {
		t.Fatal("global catalog cannot be tenant-owned")
	}
	if scalar("SELECT count(*) FROM pg_constraint WHERE conrelid IN ('tenant_memberships'::regclass,'roles'::regclass,'permissions'::regclass,'role_permissions'::regclass,'role_assignments'::regclass) AND contype='f' AND (confdeltype<>'a' OR confupdtype<>'a' OR condeferrable)") != 0 {
		t.Fatal("unexpected cascading or deferred FK")
	}
	t.Log("PASS six fixed roles and immutable identity/scope; no seeded assignments; mapping FKs/uniques; all four scope shapes, NULL-aware duplicates, multiple roles, platform without membership, cross-tenant/org/branch insert/update and ownership rejection")
	if os.Getenv("M7_STAGE") == "full" {
		verifyAuthorization(t, conn, userA, userB, outsider, tenantA, tenantB, orgA, orgA2, orgB, branchA, branchA2)
	}
}

func verifyPolicyCatalog(t *testing.T, conn *pgx.Conn) {
	t.Helper()
	ctx := context.Background()
	var codes []string
	if err := conn.QueryRow(ctx, "SELECT array_agg(code ORDER BY code COLLATE \"C\") FROM permissions").Scan(&codes); err != nil {
		t.Fatal("permission catalog unavailable")
	}
	if want := []string{"branch.read", "department.read", "membership.read", "organization.read", "platform.manage", "position.read", "tenant.read", "work_location.read"}; !slices.Equal(codes, want) {
		t.Fatalf("permission catalog mismatch: %v", codes)
	}
	expected := map[string][]string{
		"SUPER_ADMIN":  {"platform.manage"},
		"TENANT_ADMIN": {"branch.read", "department.read", "membership.read", "organization.read", "position.read", "tenant.read", "work_location.read"},
		"HR_ADMIN":     {"branch.read", "department.read", "organization.read", "position.read", "work_location.read"},
		"MANAGER":      {"branch.read", "department.read", "organization.read", "position.read"},
		"SUPERVISOR":   {"branch.read", "department.read"},
		"EMPLOYEE":     {},
	}
	rows, err := conn.Query(ctx, `SELECT r.code, ARRAY(
		SELECT p.code FROM role_permissions rp JOIN permissions p ON p.id=rp.permission_id
		WHERE rp.role_id=r.id ORDER BY p.code COLLATE "C")
		FROM roles r`)
	if err != nil {
		t.Fatal("role matrix unavailable")
	}
	defer rows.Close()
	for rows.Next() {
		var role string
		var grants []string
		if err := rows.Scan(&role, &grants); err != nil {
			t.Fatal("role matrix scan failed")
		}
		want, ok := expected[role]
		if !ok || !slices.Equal(grants, want) {
			t.Fatalf("unexpected grants for %s: %v", role, grants)
		}
		delete(expected, role)
	}
	if rows.Err() != nil || len(expected) != 0 {
		t.Fatal("incomplete role matrix")
	}
	t.Log("PASS exact eight permission codes and six-role policy matrix")
}

func verifyAuthorization(t *testing.T, conn *pgx.Conn, userA, userB, outsider, tenantA, tenantB, orgA, orgA2, orgB, branchA, branchA2 string) {
	t.Helper()
	db := stdlib.OpenDB(*conn.Config())
	t.Cleanup(func() { _ = db.Close() })
	var actual string
	if db.QueryRow("SELECT current_database()").Scan(&actual) != nil || actual != conn.Config().Database {
		t.Fatal("authorization pool target mismatch")
	}
	service := access.NewService(accessstore.NewStore(db, time.Second))
	identity := func(user string) context.Context {
		return auth.WithIdentity(context.Background(), models.Identity{UserID: uuid.MustParse(user), SessionID: uuid.New()})
	}
	tenantScope := access.Scope{Type: "TENANT", TenantID: uuid.MustParse(tenantA)}
	orgScope := access.Scope{Type: "ORGANIZATION", TenantID: uuid.MustParse(tenantA), OrganizationID: uuid.MustParse(orgA)}
	branchScope := orgScope
	branchScope.Type, branchScope.BranchID = "BRANCH", uuid.MustParse(branchA)
	childScope := orgScope
	childScope.OrganizationID = uuid.MustParse(orgA2)
	childBranch := branchScope
	childBranch.OrganizationID, childBranch.BranchID = childScope.OrganizationID, uuid.MustParse(branchA2)
	otherTenant := access.Scope{Type: "TENANT", TenantID: uuid.MustParse(tenantB)}
	platform := access.Scope{Type: "PLATFORM"}
	check := func(label string, ctx context.Context, code string, scope access.Scope, want error) {
		t.Helper()
		authorized, err := service.Authorize(ctx, code, scope)
		if !errors.Is(err, want) {
			t.Fatalf("%s: got %v, want %v", label, err, want)
		}
		if want == nil {
			if got, ok := access.ScopeFromContext(authorized); !ok || got != scope {
				t.Fatalf("%s: authorized context scope mismatch", label)
			}
		} else if authorized != nil {
			t.Fatalf("%s: denial returned authorized context", label)
		}
	}
	execSQL := func(query string, args ...any) {
		t.Helper()
		if _, err := conn.Exec(context.Background(), query, args...); err != nil {
			t.Fatal("authorization fixture SQL failed; details withheld")
		}
	}
	ctx := identity(userA)
	denied := access.ErrForbidden
	execSQL("UPDATE organizations SET parent_organization_id=$1 WHERE id=$2", orgA, orgA2)
	execSQL("DELETE FROM role_assignments WHERE user_id=$1 AND role_id=(SELECT id FROM roles WHERE code='HR_ADMIN')", userA)
	execSQL(`INSERT INTO role_assignments (tenant_id,user_id,role_id,scope_type)
		VALUES ($1,$2,(SELECT id FROM roles WHERE code='EMPLOYEE'),'TENANT')`, tenantB, userB)
	activate := func(scope string) {
		t.Helper()
		execSQL("UPDATE role_assignments SET is_active=(scope_type=$2) WHERE user_id=$1", userA, scope)
	}

	activate("TENANT")
	check("missing identity", context.Background(), "tenant.read", tenantScope, auth.ErrUnauthenticated)
	check("membership without grant", identity(userB), "tenant.read", otherTenant, denied)
	check("unknown permission", ctx, "fixture.unknown", tenantScope, denied)
	check("invalid scope", ctx, "tenant.read", access.Scope{Type: "INVALID"}, denied)
	check("tenant itself", ctx, "tenant.read", tenantScope, nil)
	check("tenant organization", ctx, "organization.read", orgScope, nil)
	check("tenant branch", ctx, "work_location.read", branchScope, nil)
	check("tenant child organization", ctx, "branch.read", childBranch, nil)
	check("tenant cannot cross tenant even with membership there", ctx, "tenant.read", otherTenant, denied)
	check("tenant without platform grant", ctx, "platform.manage", tenantScope, denied)
	check("platform grant", identity(outsider), "platform.manage", platform, nil)
	check("platform grant cannot cross into tenant", identity(outsider), "platform.manage", tenantScope, denied)
	check("platform has no tenant grant", identity(outsider), "tenant.read", tenantScope, denied)
	check("tenant cannot grant platform", ctx, "tenant.read", platform, denied)
	wrongOrg := orgScope
	wrongOrg.OrganizationID = uuid.MustParse(orgB)
	check("cross-tenant organization tuple", ctx, "organization.read", wrongOrg, denied)
	wrongBranch := branchScope
	wrongBranch.BranchID = uuid.MustParse(branchA2)
	check("branch outside requested organization", ctx, "branch.read", wrongBranch, denied)
	wrongBranch.BranchID = uuid.New()
	check("unknown branch", ctx, "branch.read", wrongBranch, denied)

	activate("ORGANIZATION")
	check("assigned organization", ctx, "organization.read", orgScope, nil)
	check("branch owned by assigned organization", ctx, "branch.read", branchScope, nil)
	check("child organization", ctx, "organization.read", childScope, denied)
	check("child organization's branch", ctx, "branch.read", childBranch, denied)
	check("organization cannot authorize tenant", ctx, "tenant.read", tenantScope, denied)
	activate("BRANCH")
	check("assigned branch resource", ctx, "work_location.read", branchScope, nil)
	var sibling uuid.UUID
	if err := conn.QueryRow(context.Background(), `INSERT INTO branches (tenant_id,organization_id,code,name,timezone)
		VALUES ($1,$2,'sibling','Sibling','UTC') RETURNING id`, tenantA, orgA).Scan(&sibling); err != nil {
		t.Fatal("sibling branch fixture failed")
	}
	siblingScope := branchScope
	siblingScope.BranchID = sibling
	check("sibling branch", ctx, "branch.read", siblingScope, denied)
	check("branch cannot authorize organization resource", ctx, "work_location.read", orgScope, denied)
	activate("")
	check("inactive assignment", ctx, "branch.read", branchScope, denied)
	activate("TENANT")

	for _, change := range []struct {
		table, where string
		args         []any
	}{
		{"tenant_memberships", "tenant_id=$1 AND user_id=$2", []any{tenantA, userA}},
		{"tenants", "id=$1", []any{tenantA}},
		{"organizations", "id=$1", []any{orgA}},
		{"branches", "id=$1", []any{branchA}},
		{"users", "id=$1", []any{userA}},
	} {
		execSQL("UPDATE "+change.table+" SET is_active=false WHERE "+change.where, change.args...)
		check("inactive "+change.table, ctx, "branch.read", branchScope, denied)
		execSQL("UPDATE "+change.table+" SET is_active=true WHERE "+change.where, change.args...)
		check("reactivated "+change.table, ctx, "branch.read", branchScope, nil)
	}
	const mapping = `SELECT r.id, p.id FROM roles r, permissions p
		WHERE r.code='TENANT_ADMIN' AND p.code='branch.read'`
	execSQL("DELETE FROM role_permissions WHERE (role_id,permission_id) IN (" + mapping + ")")
	check("removed mapping", ctx, "branch.read", branchScope, denied)
	execSQL("INSERT INTO role_permissions(role_id,permission_id) " + mapping)
	check("restored mapping", ctx, "branch.read", branchScope, nil)

	// The fixture route resolves a real location by its requested tenant and ID.
	// A caller cannot authorize a foreign location by supplying their own tenant.
	location := func(tenant, org string, branch any, code string) string {
		t.Helper()
		var id string
		if err := conn.QueryRow(context.Background(), `INSERT INTO work_locations (tenant_id,organization_id,branch_id,code,name)
			VALUES ($1,$2,$3,$4,'Fixture') RETURNING id::text`, tenant, org, branch, code).Scan(&id); err != nil {
			t.Fatal("location fixture failed")
		}
		return id
	}
	var foreignBranch string
	if err := conn.QueryRow(context.Background(), "SELECT id::text FROM branches WHERE organization_id=$1", orgB).Scan(&foreignBranch); err != nil {
		t.Fatal("foreign branch fixture missing")
	}
	localLocation := location(tenantA, orgA, branchA, "branch_location")
	orgLocation := location(tenantA, orgA, nil, "org_location")
	foreignLocation := location(tenantB, orgB, foreignBranch, "foreign_location")
	gormDB, err := gorm.Open(postgres.New(postgres.Config{Conn: db}), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal("auth fixture store unavailable")
	}
	const secret = "disposable M7 authentication secret"
	hashed, err := password.Hash(secret)
	if err != nil {
		t.Fatal(err)
	}
	execSQL("UPDATE users SET password_hash=$1 WHERE id=$2", hashed, userA)
	authentication := auth.NewService(authstore.NewStore(gormDB, time.Second), time.Minute)
	login := func() string {
		t.Helper()
		result, err := authentication.Login(context.Background(), "a@test.invalid", secret, auth.Metadata{})
		if err != nil {
			t.Fatal("fixture login failed")
		}
		return result.AccessToken
	}
	token := login()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/fixture/:tenant_id/:location_id", middlewares.BearerAuthentication(authentication, nil), middlewares.RequirePermission(service, "work_location.read", func(c *gin.Context) (access.Scope, error) {
		tenantID, tenantErr := uuid.Parse(c.Param("tenant_id"))
		locationID, locationErr := uuid.Parse(c.Param("location_id"))
		if tenantErr != nil || locationErr != nil {
			return access.Scope{}, denied
		}
		var scope access.Scope
		var branch uuid.NullUUID
		err := db.QueryRowContext(c.Request.Context(), `SELECT tenant_id, organization_id, branch_id
			FROM work_locations WHERE tenant_id=$1 AND id=$2 AND is_active`, tenantID, locationID).Scan(&scope.TenantID, &scope.OrganizationID, &branch)
		if errors.Is(err, sql.ErrNoRows) {
			return access.Scope{}, denied
		}
		scope.Type = "ORGANIZATION"
		if branch.Valid {
			scope.Type, scope.BranchID = "BRANCH", branch.UUID
		}
		return scope, err
	}), func(c *gin.Context) {
		scope, ok := access.ScopeFromContext(c.Request.Context())
		want := branchScope
		if c.Param("location_id") == orgLocation {
			want = orgScope
		}
		if !ok || scope != want {
			t.Error("handler did not receive the verified resource scope")
		}
		c.Status(200)
	})
	request := func(tenant, location, token string) int {
		r := httptest.NewRequest("GET", "/fixture/"+tenant+"/"+location, nil)
		if token != "" {
			r.Header.Set("Authorization", "Bearer "+token)
		}
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)
		return w.Code
	}
	if request(tenantA, localLocation, "") != 401 || request(tenantA, localLocation, token) != 200 ||
		request(tenantB, foreignLocation, token) != 403 || request(tenantA, foreignLocation, token) != 403 ||
		request(tenantB, localLocation, token) != 403 || request("malformed", localLocation, token) != 403 {
		t.Fatal("bearer authorization or resource ownership boundary failed")
	}
	activate("ORGANIZATION")
	if request(tenantA, localLocation, token) != 200 || request(tenantA, orgLocation, token) != 200 {
		t.Fatal("organization assignment did not cover stored organization and branch resources")
	}
	activate("BRANCH")
	if request(tenantA, localLocation, token) != 200 || request(tenantA, orgLocation, token) != 403 {
		t.Fatal("branch assignment did not distinguish stored branch and organization resources")
	}
	activate("TENANT")
	execSQL("UPDATE tenant_memberships SET is_active=false WHERE tenant_id=$1 AND user_id=$2", tenantA, userA)
	if _, err := authentication.Authenticate(context.Background(), token); err != nil || request(tenantA, localLocation, token) != 403 {
		t.Fatal("membership revocation must deny without revoking global session")
	}
	execSQL("UPDATE tenant_memberships SET is_active=true WHERE tenant_id=$1 AND user_id=$2", tenantA, userA)
	session, err := authentication.Authenticate(context.Background(), token)
	if err != nil || authentication.Logout(context.Background(), session) != nil || request(tenantA, localLocation, token) != 401 {
		t.Fatal("revoked session accepted")
	}
	expired := login()
	execSQL("UPDATE auth_sessions SET expires_at=CURRENT_TIMESTAMP WHERE token_hash=$1", auth.TokenHash(expired))
	if request(tenantA, localLocation, expired) != 401 {
		t.Fatal("expired session accepted")
	}
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	check("canceled query", canceled, "branch.read", branchScope, access.ErrUnavailable)
	if db.Close() != nil {
		t.Fatal("pool close failed")
	}
	check("closed database", ctx, "branch.read", branchScope, access.ErrUnavailable)
	t.Log("PASS v1.0 containment, no child traversal/platform bypass/implicit grants, active owners, immediate mapping changes, stored-resource ID isolation and bearer expiry/revocation")
}

func verifyM7Catalog(t *testing.T, conn *pgx.Conn, last int) {
	t.Helper()
	contracts := map[string]string{
		"tenant_memberships": "id:uuid:NO|tenant_id:uuid:NO|user_id:uuid:NO|is_active:boolean:NO",
	}
	if last >= 14 {
		contracts["roles"] = "id:uuid:NO|code:character varying(30):NO|name:character varying(100):NO|role_scope:character varying(10):NO"
		contracts["permissions"] = "id:uuid:NO|code:character varying(100):NO|name:character varying(150):NO"
		contracts["role_permissions"] = "role_id:uuid:NO|permission_id:uuid:NO"
		contracts["role_assignments"] = "id:uuid:NO|tenant_id:uuid:YES|user_id:uuid:NO|role_id:uuid:NO|scope_type:character varying(12):NO|organization_id:uuid:YES|branch_id:uuid:YES|is_active:boolean:NO"
	}
	for table, columns := range contracts {
		expected := strings.Split(columns+"|created_at:timestamp with time zone:NO|created_by:uuid:YES|updated_at:timestamp with time zone:NO|updated_by:uuid:YES", "|")
		sort.Strings(expected)
		var actual []string
		if conn.QueryRow(context.Background(), `SELECT array_agg(a.attname || ':' || format_type(a.atttypid,a.atttypmod) || ':' || CASE WHEN a.attnotnull THEN 'NO' ELSE 'YES' END ORDER BY a.attname) FROM pg_attribute a WHERE a.attrelid=('public.' || $1)::regclass AND a.attnum>0 AND NOT a.attisdropped`, table).Scan(&actual) != nil || strings.Join(actual, "|") != strings.Join(expected, "|") {
			t.Fatalf("%s exact type/nullability mismatch", table)
		}
		var defaults int
		if conn.QueryRow(context.Background(), `SELECT count(*) FROM information_schema.columns WHERE table_schema='public' AND table_name=$1 AND ((column_name IN ('created_at','updated_at') AND column_default='CURRENT_TIMESTAMP') OR (column_name='is_active' AND column_default='true') OR (column_name='id' AND column_default='gen_random_uuid()'))`, table).Scan(&defaults) != nil {
			t.Fatal("default query failed")
		}
		want := 2
		if table == "tenant_memberships" || table == "role_assignments" {
			want = 4
		} else if table == "permissions" {
			want = 3
		}
		if defaults != want {
			t.Fatalf("%s default contract mismatch", table)
		}
	}
	if last >= 14 {
		var fks, indexes, triggers int
		tables := "('tenant_memberships','roles','permissions','role_permissions','role_assignments')"
		if conn.QueryRow(context.Background(), "SELECT count(*) FROM pg_constraint c JOIN pg_class t ON t.oid=c.conrelid JOIN pg_namespace n ON n.oid=t.relnamespace WHERE n.nspname='public' AND t.relname IN "+tables+" AND c.contype='f'").Scan(&fks) != nil || fks != 20 {
			t.Fatal("expected20 M7 foreign keys")
		}
		if conn.QueryRow(context.Background(), "SELECT count(*) FROM pg_indexes WHERE schemaname='public' AND tablename IN "+tables).Scan(&indexes) != nil || indexes != 9 {
			t.Fatal("expected9 M7 indexes")
		}
		if conn.QueryRow(context.Background(), "SELECT count(*) FROM pg_trigger g JOIN pg_class t ON t.oid=g.tgrelid JOIN pg_namespace n ON n.oid=t.relnamespace WHERE n.nspname='public' AND t.relname IN "+tables+" AND NOT g.tgisinternal").Scan(&triggers) != nil || triggers != 0 {
			t.Fatal("no M7 user triggers expected")
		}
	}
	t.Log("PASS exact M7 columns/types/nullability/defaults; full catalog20 FKs/9 indexes/no user triggers")
}
