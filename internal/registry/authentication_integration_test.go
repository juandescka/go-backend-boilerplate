package registry

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"hris-core-service/config"
	"hris-core-service/internal/domain/models"
	authstore "hris-core-service/internal/infrastructure/authentication"
	"hris-core-service/internal/infrastructure/migration"
	"hris-core-service/internal/middlewares"
	"hris-core-service/internal/routes"
	"hris-core-service/internal/shared/observability"
	"hris-core-service/internal/shared/password"
	securityevent "hris-core-service/internal/shared/security"
	auth "hris-core-service/internal/usecases/authentication"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Actual application components + real PostgreSQL. Every write/CLI migration
// targets a newly created random database; the development catalog is read-only.
func TestAuthenticationPostgresIntegration(t *testing.T) {
	if os.Getenv("M5B_INTEGRATION") != "1" {
		t.Skip("set M5B_INTEGRATION=1 and M5B_CLI_BINARY for disposable authentication verification")
	}
	binary, err := filepath.Abs(os.Getenv("M5B_CLI_BINARY"))
	if err != nil || os.Getenv("M5B_CLI_BINARY") == "" {
		t.Fatal("current CLI binary required")
	}
	t.Chdir("../..")
	previous := config.AppConfig
	t.Cleanup(func() { config.AppConfig = previous })
	if config.LoadConfig() != nil {
		t.Fatal("effective configuration could not be loaded")
	}
	base := config.AppConfig.Database
	t.Logf("Effective target: %s:%d database=%s user=%s ssl=%s; secrets withheld", base.Host, base.Port, base.Name, base.Username, base.SSLMode)
	baseURL := migration.NewConfig().DatabaseURL
	connect := func(name string) *pgx.Conn {
		t.Helper()
		u, e := url.Parse(baseURL)
		if e != nil {
			t.Fatal("invalid connection configuration")
		}
		u.Path = "/" + name
		u.RawPath = ""
		parsed, e := pgx.ParseConfig(u.String())
		if e != nil {
			t.Fatal("invalid connection configuration")
		}
		parsed.RuntimeParams["application_name"] = "m5b-verification"
		parsed.RuntimeParams["statement_timeout"] = "5000"
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		conn, e := pgx.ConnectConfig(ctx, parsed)
		if e != nil {
			t.Fatal("PostgreSQL verification connection failed (details withheld)")
		}
		return conn
	}
	execSQL := func(c *pgx.Conn, sql string, args ...any) {
		t.Helper()
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if _, e := c.Exec(ctx, sql, args...); e != nil {
			t.Fatal("verification SQL failed (details withheld)")
		}
	}
	intValue := func(c *pgx.Conn, sql string, args ...any) int {
		t.Helper()
		var value int
		if c.QueryRow(context.Background(), sql, args...).Scan(&value) != nil {
			t.Fatal("verification query failed (details withheld)")
		}
		return value
	}
	development := connect(base.Name)
	t.Cleanup(func() { _ = development.Close(context.Background()) })
	snapshot := func() string {
		t.Helper()
		execSQL(development, "BEGIN READ ONLY")
		var value string
		e := development.QueryRow(context.Background(), `SELECT COALESCE(string_agg(c.relname || ':' || c.relkind::text,',' ORDER BY c.relname),'') FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='public'`).Scan(&value)
		execSQL(development, "ROLLBACK")
		if e != nil {
			t.Fatal("read-only baseline failed")
		}
		return value
	}
	before := snapshot()
	t.Cleanup(func() {
		if snapshot() != before {
			t.Error("development catalog changed")
		} else {
			t.Log("PASS configured development catalog unchanged; no migrator targeted it")
		}
	})
	admin := connect("postgres")
	t.Cleanup(func() { _ = admin.Close(context.Background()) })
	if intValue(admin, "SELECT count(*) FROM pg_roles WHERE rolname=current_user AND (rolcreatedb OR rolsuper)") != 1 {
		t.Fatal("disposable creation permission required")
	}
	random := make([]byte, 8)
	if _, err := rand.Read(random); err != nil {
		t.Fatal(err)
	}
	dbName := "m5b_verify_" + hex.EncodeToString(random)
	if dbName == base.Name {
		t.Fatal("test name equals configured database")
	}
	execSQL(admin, "CREATE DATABASE "+pgx.Identifier{dbName}.Sanitize())
	var oid uint32
	if admin.QueryRow(context.Background(), "SELECT oid FROM pg_database WHERE datname=$1 AND datdba=(SELECT oid FROM pg_roles WHERE rolname=current_user)", dbName).Scan(&oid) != nil {
		t.Fatal("created ownership unavailable; inspect manually")
	}
	var observer *pgx.Conn
	t.Cleanup(func() {
		if observer != nil {
			_ = observer.Close(context.Background())
		}
		if intValue(admin, "SELECT count(*) FROM pg_database WHERE datname=$1 AND oid=$2 AND datdba=(SELECT oid FROM pg_roles WHERE rolname=current_user)", dbName, oid) != 1 {
			t.Error("ownership changed; refusing cleanup")
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if _, e := admin.Exec(ctx, "DROP DATABASE "+pgx.Identifier{dbName}.Sanitize()); e != nil {
			t.Errorf("owned database %s cleanup failed (details withheld)", dbName)
			return
		}
		if intValue(admin, "SELECT count(*) FROM pg_database WHERE datname=$1", dbName) != 0 {
			t.Error("owned database still present")
		}
		t.Logf("PASS ownership/OID-checked cleanup: %s removed", dbName)
	})
	observer = connect(dbName)
	if intValue(observer, "SELECT current_setting('server_version_num')::int/10000") != 17 || intValue(observer, "SELECT count(*) FROM information_schema.tables WHERE table_schema='public'") != 0 {
		t.Fatal("fresh PostgreSQL17 disposable required")
	}
	t.Setenv("DB_NAME", dbName)
	t.Setenv("AUTH_SESSION_LIFETIME_MINUTES", "15")
	t.Setenv("AUTH_LOGIN_IP_RATE_LIMIT", "10000")
	t.Setenv("AUTH_LOGIN_IDENTIFIER_FAILURE_LIMIT", "10000")
	reg, err := NewRegistry()
	if err != nil {
		t.Fatal("registry initialization failed (details withheld)")
	}
	t.Cleanup(func() { _ = reg.Close() })
	var actualDB string
	if reg.DB.Raw("SELECT current_database()").Scan(&actualDB).Error != nil || actualDB != dbName {
		t.Fatal("registry target mismatch")
	}
	if intValue(observer, "SELECT count(*) FROM information_schema.tables WHERE table_schema='public'") != 0 {
		t.Fatal("startup unexpectedly created schema/metadata")
	}
	var logs bytes.Buffer
	reg.Logger.SetOutput(&logs)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	_ = router.SetTrustedProxies(nil)
	router.Use(observability.RequestIDMiddleware(), observability.RequestCompletionLogger(reg.Logger), observability.PanicRecovery(reg.Logger), middlewares.SecurityHeaders(), middlewares.CORSMiddleware(config.AppConfig.CORS), middlewares.RequestBodyLimit(config.AppConfig.HTTP.MaxRequestBodyBytes))
	routes.SetupRoutes(router, reg.OperationalHandler)
	routes.SetupAuthenticationRoutes(router, reg.AuthenticationHandler, middlewares.BearerAuthentication(reg.Authentication, securityevent.NewLogrusRecorder(reg.Logger)))
	routes.SetupAPIDocsRoutes(router, "development")
	perform := func(method, path, body, token string) *httptest.ResponseRecorder {
		t.Helper()
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		r.RemoteAddr = "192.0.2.10:4321"
		r.Header.Set("User-Agent", "Verification\r\nagent")
		r.Header.Set("X-Forwarded-For", "198.51.100.25")
		if method == "POST" {
			r.Header.Set("Content-Type", "application/json")
		}
		if token != "" {
			r.Header.Set("Authorization", "Bearer "+token)
		}
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)
		return w
	}
	if perform("GET", "/health", "", "").Code != 200 || perform("GET", "/ready", "", "").Code != 200 {
		t.Fatal("operational regression before schema")
	}
	if perform("POST", "/v1/auth/login", `{"email":"absent@example.com","password":"test input"}`, "").Code != 500 {
		t.Fatal("missing schema must be a safe operational failure")
	}
	// Keep authentication checks on the full current schema while retaining the
	// original token-index rollback check below as new module migrations arrive.
	entries, err := os.ReadDir(migration.DefaultMigrationsPath)
	if err != nil {
		t.Fatal("migration inventory unavailable")
	}
	latestVersion := 0
	for _, entry := range entries {
		var version int
		if strings.HasSuffix(entry.Name(), ".up.sql") {
			if _, err := fmt.Sscanf(entry.Name(), "%06d_", &version); err == nil && version > latestVersion {
				latestVersion = version
			}
		}
	}
	if latestVersion < 3 {
		t.Fatal("authentication migration source incomplete")
	}
	cli := func(command, expected string) {
		t.Helper()
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		p := exec.CommandContext(ctx, binary, "migrate", command)
		p.Env = os.Environ()
		output, e := p.CombinedOutput()
		if e != nil || !strings.Contains(string(output), expected) {
			t.Fatalf("CLI migrate %s failed (output withheld)", command)
		}
		for _, secret := range []string{base.Password, config.AppConfig.Security.JWTAuthSecretKey} {
			if secret != "" && strings.Contains(string(output), secret) {
				t.Fatal("CLI output leaked credential")
			}
		}
	}
	cli("up", "Migrations applied successfully")
	cli("version", fmt.Sprintf("Migration version: %d dirty: false", latestVersion))
	cli("up", "No pending migrations")
	if intValue(observer, "SELECT count(*) FROM pg_indexes WHERE schemaname='public' AND indexname='auth_sessions_token_hash_uidx' AND indexdef LIKE 'CREATE UNIQUE INDEX%'") != 1 {
		t.Fatal("unique token lookup index missing")
	}
	t.Logf("PASS fresh registry has no startup migrations; actual CLI applies versions1–%d cleanly and repeat-up succeeds", latestVersion)

	secret := uuid.NewString()
	encoded, err := password.Hash(secret)
	if err != nil {
		t.Fatal(err)
	}
	var userID uuid.UUID
	if observer.QueryRow(context.Background(), `INSERT INTO users(email,password_hash,display_name,is_active) VALUES (' Integration@Mapatu.test ',$1,'Integration user',true) RETURNING id`, encoded).Scan(&userID) != nil {
		t.Fatal("hashed fixture creation failed")
	}
	body := func(email, value string) string {
		b, _ := json.Marshal(map[string]string{"email": email, "password": value})
		return string(b)
	}
	var generic string
	for _, tc := range []struct{ email, value string }{{"integration@mapatu.test", "incorrect"}, {"absent@mapatu.test", secret}} {
		result := perform("POST", "/v1/auth/login", body(tc.email, tc.value), "")
		if result.Code != 401 {
			t.Fatal("invalid credentials accepted")
		}
		if generic == "" {
			generic = result.Body.String()
		} else if result.Body.String() != generic {
			t.Fatal("enumeration response differs")
		}
	}
	execSQL(observer, "UPDATE users SET is_active=false WHERE id=$1", userID)
	if r := perform("POST", "/v1/auth/login", body("integration@mapatu.test", secret), ""); r.Code != 401 || r.Body.String() != generic {
		t.Fatal("inactive account accepted/distinguishable")
	}
	execSQL(observer, "UPDATE users SET is_active=true,password_hash=NULL WHERE id=$1", userID)
	if r := perform("POST", "/v1/auth/login", body("integration@mapatu.test", secret), ""); r.Code != 401 || r.Body.String() != generic {
		t.Fatal("no-password account accepted/distinguishable")
	}
	execSQL(observer, "UPDATE users SET password_hash=$2 WHERE id=$1", userID, encoded)
	if intValue(observer, "SELECT count(*) FROM auth_sessions") != 0 {
		t.Fatal("failed login persisted a session")
	}
	login := perform("POST", "/v1/auth/login", body(" INTEGRATION@MAPATU.TEST ", secret), "")
	var response struct {
		Success bool `json:"success"`
		Data    struct {
			Token     string    `json:"access_token"`
			TokenType string    `json:"token_type"`
			ExpiresAt time.Time `json:"expires_at"`
			User      struct {
				ID string `json:"id"`
			}
		}
	}
	if login.Code != 200 || json.Unmarshal(login.Body.Bytes(), &response) != nil || !response.Success || !auth.ValidToken(response.Data.Token) || response.Data.TokenType != "Bearer" || response.Data.User.ID != userID.String() {
		t.Fatal("login response contract failed (body withheld)")
	}
	token := response.Data.Token
	var hash, ip, agent string
	var expires time.Time
	if observer.QueryRow(context.Background(), "SELECT token_hash,host(ip_address),user_agent,expires_at FROM auth_sessions WHERE user_id=$1", userID).Scan(&hash, &ip, &agent, &expires) != nil || hash != auth.TokenHash(token) || hash == token || len(hash) != 64 || ip != "192.0.2.10" || strings.ContainsAny(agent, "\r\n") || expires.Sub(response.Data.ExpiresAt).Abs() > time.Microsecond {
		t.Fatal("hash-only persistence/metadata/expiration incorrect")
	}
	if time.Until(expires) < 14*time.Minute || time.Until(expires) > 16*time.Minute {
		t.Fatal("configured15-minute lifetime not applied")
	}
	if intValue(observer, "SELECT count(*) FROM auth_sessions WHERE created_by=$1 AND updated_by=$1 AND created_at=updated_at", userID) != 1 || intValue(observer, "SELECT count(*) FROM users WHERE id=$1 AND last_login_at IS NOT NULL AND updated_by=$1", userID) != 1 {
		t.Fatal("audit/login write missing")
	}
	_, duplicate := observer.Exec(context.Background(), "INSERT INTO auth_sessions(user_id,token_hash,expires_at) VALUES ($1,$2,CURRENT_TIMESTAMP)", userID, hash)
	var pgErr *pgconn.PgError
	if !errors.As(duplicate, &pgErr) || pgErr.Code != "23505" {
		t.Fatal("duplicate token verifier accepted")
	}
	me := perform("GET", "/v1/auth/me", "", token)
	if me.Code != 200 || !strings.Contains(me.Body.String(), userID.String()) {
		t.Fatal("valid bearer current user failed")
	}
	for _, sensitive := range []string{"password_hash", "token_hash", "session_id", token} {
		if strings.Contains(me.Body.String(), sensitive) {
			t.Fatal("current-user response leaked internals")
		}
	}
	if perform("GET", "/v1/auth/me?access_token="+token, "", "").Code != 401 {
		t.Fatal("query token accepted")
	}
	unknownBytes := make([]byte, 32)
	_, _ = rand.Read(unknownBytes)
	if perform("GET", "/v1/auth/me", "", base64.RawURLEncoding.EncodeToString(unknownBytes)).Code != 401 {
		t.Fatal("unknown bearer accepted")
	}
	for _, state := range []string{"expired", "revoked", "inactive"} {
		switch state {
		case "expired":
			execSQL(observer, "UPDATE auth_sessions SET expires_at=CURRENT_TIMESTAMP")
		case "revoked":
			execSQL(observer, "UPDATE auth_sessions SET revoked_at=CURRENT_TIMESTAMP")
		case "inactive":
			execSQL(observer, "UPDATE users SET is_active=false")
		}
		if perform("GET", "/v1/auth/me", "", token).Code != 401 {
			t.Fatalf("%s identity accepted", state)
		}
		execSQL(observer, "UPDATE auth_sessions SET expires_at=$1,revoked_at=NULL", expires)
		execSQL(observer, "UPDATE users SET is_active=true")
	}
	t.Log("PASS login→me, normalized email, generic denials, no failed-login sessions, hash-only unique token storage, safe metadata/audit, invalid/expired/revoked/inactive401")

	store := authstore.NewStore(reg.DB, 5*time.Second)
	verified, err := store.FindUserByEmail(context.Background(), "integration@mapatu.test")
	if err != nil {
		t.Fatal("store lookup failed")
	}
	session := models.AuthSession{ID: uuid.New(), UserID: userID, TokenHash: strings.Repeat("c", 64), ExpiresAt: time.Now().Add(time.Minute), CreatedAt: time.Now(), UpdatedAt: time.Now()}
	for _, changed := range []string{"inactive", "password"} {
		if changed == "inactive" {
			execSQL(observer, "UPDATE users SET is_active=false")
		} else {
			execSQL(observer, "UPDATE users SET password_hash=NULL")
		}
		if err := store.CreateSession(context.Background(), verified, session); !errors.Is(err, auth.ErrUnauthenticated) {
			t.Fatal("stale verified account created session")
		}
		execSQL(observer, "UPDATE users SET is_active=true,password_hash=$1", encoded)
	}
	execSQL(observer, "UPDATE users SET last_login_at='2000-01-01Z'")
	execSQL(observer, "ALTER TABLE auth_sessions ADD CONSTRAINT m5b_reject_fixture CHECK (false) NOT VALID")
	failed := perform("POST", "/v1/auth/login", body("integration@mapatu.test", secret), "")
	if failed.Code != 500 || strings.Contains(failed.Body.String(), "access_token") || intValue(observer, "SELECT count(*) FROM auth_sessions") != 1 || intValue(observer, "SELECT count(*) FROM users WHERE last_login_at='2000-01-01Z'") != 1 {
		t.Fatal("session failure did not roll back login write safely")
	}
	execSQL(observer, "ALTER TABLE auth_sessions DROP CONSTRAINT m5b_reject_fixture")
	execSQL(observer, "BEGIN")
	execSQL(observer, "SELECT id FROM users FOR UPDATE")
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	started := time.Now()
	blocked := store.CreateSession(ctx, verified, session)
	cancel()
	execSQL(observer, "ROLLBACK")
	if blocked == nil || time.Since(started) > time.Second || intValue(observer, "SELECT count(*) FROM auth_sessions") != 1 {
		t.Fatal("context-bounded transaction wait failed")
	}
	if perform("GET", "/docs", "", "").Code != 200 || !strings.Contains(perform("GET", "/swagger/doc.json", "", "").Body.String(), "/v1/auth/me") {
		t.Fatal("Scalar/spec regression")
	}
	t.Log("PASS stale password/activation recheck, atomic persistence-failure rollback, context-bounded DB wait and Scalar current spec")
	verifyLoginHardening(t, reg, observer, binary, secret)
	lifecycleSecrets := verifySessionLifecycle(t, reg.DB, observer, perform)
	usersBeforeRollback := intValue(observer, "SELECT count(*) FROM users")
	sessionsBeforeRollback := intValue(observer, "SELECT count(*) FROM auth_sessions")
	if reg.Close() != nil {
		t.Fatal("pool cleanup failed")
	}
	if perform("POST", "/v1/auth/login", body("integration@mapatu.test", secret), "").Code != 500 || perform("GET", "/v1/auth/me", "", token).Code != 500 || perform("GET", "/health", "", "").Code != 200 || perform("GET", "/ready", "", "").Code != 503 {
		t.Fatal("closed-pool operational error/regression contract failed")
	}
	for _, sensitive := range append(lifecycleSecrets, secret, encoded, token, hash, base.Password, config.AppConfig.Security.JWTAuthSecretKey) {
		if sensitive != "" && strings.Contains(logs.String(), sensitive) {
			t.Fatal("captured logs leaked credentials")
		}
	}
	for version := latestVersion; version > 3; version-- {
		cli("down", "Migration rolled back successfully")
		cli("version", fmt.Sprintf("Migration version: %d dirty: false", version-1))
		if intValue(observer, "SELECT count(*) FROM auth_sessions") != sessionsBeforeRollback || intValue(observer, "SELECT count(*) FROM users") != usersBeforeRollback || intValue(observer, "SELECT count(*) FROM pg_indexes WHERE schemaname='public' AND indexname='auth_sessions_token_hash_uidx'") != 1 {
			t.Fatal("later-module rollback changed authentication data/index")
		}
	}
	cli("down", "Migration rolled back successfully")
	cli("version", "Migration version: 2 dirty: false")
	if intValue(observer, "SELECT count(*) FROM auth_sessions") != sessionsBeforeRollback || intValue(observer, "SELECT count(*) FROM users") != usersBeforeRollback || intValue(observer, "SELECT count(*) FROM pg_indexes WHERE indexname='auth_sessions_token_hash_uidx'") != 0 {
		t.Fatal("index rollback changed business data")
	}
	cli("up", "Migrations applied successfully")
	cli("version", fmt.Sprintf("Migration version: %d dirty: false", latestVersion))
	t.Logf("PASS safe DB-failure500, health200/ready503, secret-free logs, later-module rollback, index-only down2/reapply%d retaining data", latestVersion)
}
