package registry

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"net/http/httptest"
	"testing"
	"time"

	"hris-core-service/internal/domain/models"
	authstore "hris-core-service/internal/infrastructure/authentication"
	auth "hris-core-service/internal/usecases/authentication"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"gorm.io/gorm"
)

// Reuse the authentication test's owned database, routes and cleanup. Group the
// lifecycle/locking checks here without copying its disposable-DB infrastructure.
func verifySessionLifecycle(t *testing.T, db *gorm.DB, observer *pgx.Conn, perform func(string, string, string, string) *httptest.ResponseRecorder) []string {
	t.Helper()
	ctx := context.Background()
	store := authstore.NewStore(db, 5*time.Second)
	var sensitive []string
	execSQL := func(query string, args ...any) {
		t.Helper()
		if _, err := observer.Exec(ctx, query, args...); err != nil {
			t.Fatal("lifecycle fixture SQL failed (details withheld)")
		}
	}
	count := func(query string, args ...any) int {
		t.Helper()
		var result int
		if observer.QueryRow(ctx, query, args...).Scan(&result) != nil {
			t.Fatal("lifecycle count failed")
		}
		return result
	}
	snapshot := func(user uuid.UUID) string {
		t.Helper()
		var value string
		if observer.QueryRow(ctx, `SELECT COALESCE(string_agg(id::text || ':' || COALESCE(revoked_at::text,'') || ':' || updated_at::text || ':' || COALESCE(updated_by::text,''),',' ORDER BY id),'') FROM auth_sessions WHERE user_id=$1`, user).Scan(&value) != nil {
			t.Fatal("lifecycle snapshot failed")
		}
		return value
	}
	assertHTTP := func(method, path, body, token string, want int) {
		t.Helper()
		result := perform(method, path, body, token)
		if result.Code != want {
			t.Fatalf("lifecycle %s %s: got%d want%d (body withheld)", method, path, result.Code, want)
		}
	}
	newUser := func() models.User {
		t.Helper()
		var user models.User
		// These accounts have no password provider; sessions below are internal fixtures.
		if observer.QueryRow(ctx, "INSERT INTO users(email,display_name,is_active) VALUES ($1,'Lifecycle verification',true) RETURNING id,email", uuid.NewString()+"@mapatu.test").Scan(&user.ID, &user.Email) != nil {
			t.Fatal("lifecycle user fixture failed")
		}
		return user
	}
	userA, userB := newUser(), newUser()
	type credential struct {
		token    string
		session  models.AuthSession
		identity models.Identity
	}
	newCredential := func(user models.User) credential {
		t.Helper()
		raw := make([]byte, 32)
		if _, err := rand.Read(raw); err != nil {
			t.Fatal(err)
		}
		token := base64.RawURLEncoding.EncodeToString(raw)
		now := time.Now().UTC()
		session := models.AuthSession{ID: uuid.New(), UserID: user.ID, TokenHash: auth.TokenHash(token), ExpiresAt: now.Add(time.Hour), CreatedAt: now, UpdatedAt: now, CreatedBy: &user.ID, UpdatedBy: &user.ID}
		sensitive = append(sensitive, token, session.TokenHash)
		return credential{token: token, session: session, identity: models.Identity{UserID: user.ID, SessionID: session.ID}}
	}
	issue := func(user models.User) credential {
		t.Helper()
		credential := newCredential(user)
		execSQL("INSERT INTO auth_sessions(id,user_id,token_hash,expires_at,created_at,updated_at,created_by,updated_by) VALUES ($1,$2,$3,$4,$5,$5,$2,$2)", credential.session.ID, user.ID, credential.session.TokenHash, credential.session.ExpiresAt, credential.session.CreatedAt)
		return credential
	}
	a1, a2, b1 := issue(userA), issue(userA), issue(userB)
	rows := count("SELECT count(*) FROM auth_sessions")
	assertHTTP("POST", "/v1/auth/logout", `{"user_id":"`+userB.ID.String()+`"}`, a1.token, 200)
	assertHTTP("GET", "/v1/auth/me", "", a1.token, 401)
	assertHTTP("POST", "/v1/auth/logout", "", a1.token, 401)
	assertHTTP("GET", "/v1/auth/me", "", a2.token, 200)
	assertHTTP("GET", "/v1/auth/me", "", b1.token, 200)
	if count("SELECT count(*) FROM auth_sessions WHERE id=$1 AND revoked_at=updated_at AND updated_by=$2", a1.session.ID, userA.ID) != 1 {
		t.Fatal("current logout audit incorrect")
	}
	beforeAllA, beforeAllB := snapshot(userA.ID), snapshot(userB.ID)
	assertHTTP("POST", "/v1/auth/logout-all", `{"user_id":"`+userB.ID.String()+`","session_id":"`+b1.session.ID.String()+`"}`, a2.token, 200)
	assertHTTP("GET", "/v1/auth/me", "", a2.token, 401)
	assertHTTP("POST", "/v1/auth/logout-all", "", a2.token, 401)
	assertHTTP("GET", "/v1/auth/me", "", b1.token, 200)
	if snapshot(userB.ID) != beforeAllB || snapshot(userA.ID) == beforeAllA || count("SELECT count(*) FROM auth_sessions") != rows {
		t.Fatal("logout-all scope or record retention incorrect")
	}
	var firstRevocation time.Time
	if observer.QueryRow(ctx, "SELECT revoked_at FROM auth_sessions WHERE id=$1", a1.session.ID).Scan(&firstRevocation) != nil {
		t.Fatal("revocation timestamp missing")
	}
	if err := store.RevokeAllSessions(ctx, a2.identity); !errors.Is(err, auth.ErrUnauthenticated) {
		t.Fatal("stale caller was accepted")
	}
	if count("SELECT count(*) FROM auth_sessions WHERE id=$1 AND revoked_at=$2", a1.session.ID, firstRevocation) != 1 {
		t.Fatal("old revocation rewritten")
	}
	if err := store.RevokeSession(ctx, models.Identity{UserID: userA.ID, SessionID: b1.session.ID}); !errors.Is(err, auth.ErrUnauthenticated) || snapshot(userB.ID) != beforeAllB {
		t.Fatal("cross-user direct revocation accepted")
	}
	t.Log("PASS current/all HTTP logout, old-token401, sibling survival, cross-user ownership, preserved records and audit/old revocations")

	// Observe PostgreSQL's actual blocking relation, rather than elapsed sleeps,
	// to establish each transaction's queue position on separate connections.
	var lockerPID int
	if observer.QueryRow(ctx, "SELECT pg_backend_pid()").Scan(&lockerPID) != nil {
		t.Fatal("locker PID unavailable")
	}
	var pending []<-chan error
	defer func() {
		_, _ = observer.Exec(ctx, "ROLLBACK")
		for _, done := range pending {
			select {
			case <-done:
			case <-time.After(6 * time.Second):
				t.Error("lifecycle verification operation did not stop")
			}
		}
	}()
	async := func(operation func(*authstore.Store) error) (int, <-chan error) {
		t.Helper()
		started := make(chan int, 1)
		done := make(chan error, 1)
		go func() {
			opCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()
			err := db.WithContext(opCtx).Connection(func(bound *gorm.DB) error {
				var pid int
				if err := bound.Raw("SELECT pg_backend_pid()").Scan(&pid).Error; err != nil {
					return err
				}
				started <- pid
				return operation(authstore.NewStore(bound, 5*time.Second))
			})
			done <- err
			close(done)
		}()
		pending = append(pending, done)
		select {
		case pid := <-started:
			return pid, done
		case <-done:
			t.Fatal("lifecycle operation failed before start")
		case <-time.After(5 * time.Second):
			t.Fatal("lifecycle operation did not start")
		}
		return 0, done
	}
	waitUntil := func(condition func() bool) {
		t.Helper()
		deadline := time.NewTimer(3 * time.Second)
		defer deadline.Stop()
		tick := time.NewTicker(5 * time.Millisecond)
		defer tick.Stop()
		for {
			if condition() {
				return
			}
			select {
			case <-tick.C:
			case <-deadline.C:
				t.Fatal("expected PostgreSQL lock/time state not observed")
			}
		}
	}
	waitBlocked := func(pid, blocker int) {
		t.Helper()
		waitUntil(func() bool {
			var blocked bool
			if observer.QueryRow(ctx, "SELECT $2=ANY(pg_blocking_pids($1))", pid, blocker).Scan(&blocked) != nil {
				t.Fatal("blocking-state query failed")
			}
			return blocked
		})
	}
	finish := func(done <-chan error, want error) {
		t.Helper()
		select {
		case err := <-done:
			if !errors.Is(err, want) {
				t.Fatal("unexpected serialized operation result (details withheld)")
			}
		case <-time.After(6 * time.Second):
			t.Fatal("serialized operation timed out")
		}
	}
	lockUser := func() { t.Helper(); execSQL("BEGIN"); execSQL("SELECT id FROM users WHERE id=$1 FOR UPDATE", userA.ID) }
	unlock := func() { t.Helper(); execSQL("COMMIT") }

	// Both callers passed authentication before either can acquire the user lock.
	current := issue(userA)
	lockUser()
	p1, d1 := async(func(s *authstore.Store) error { return s.RevokeSession(ctx, current.identity) })
	waitBlocked(p1, lockerPID)
	p2, d2 := async(func(s *authstore.Store) error { return s.RevokeSession(ctx, current.identity) })
	waitBlocked(p2, p1)
	assertHTTP("GET", "/v1/auth/me", "", current.token, 200)
	unlock()
	finish(d1, nil)
	finish(d2, auth.ErrUnauthenticated)
	assertHTTP("GET", "/v1/auth/me", "", current.token, 401)
	t.Log("PASS concurrent current logout: one commit/one401; authentication before commit may finish, later validation rejects")

	// Supply a real Argon2 credential snapshot for CreateSession's existing guard.
	var loginUser models.User
	if err := db.Select("id,email,password_hash,is_active").Where("password_hash IS NOT NULL").Take(&loginUser).Error; err != nil {
		t.Fatal("password fixture unavailable")
	}
	// Use the same user's row for these login/all orderings, with its own live caller.
	orderedUser := userA
	execSQL("UPDATE users SET password_hash=$2 WHERE id=$1", orderedUser.ID, loginUser.PasswordHash)
	verified, err := store.FindUserByEmail(ctx, orderedUser.Email)
	if err != nil {
		t.Fatal("verified lifecycle account unavailable")
	}
	caller := issue(orderedUser)
	createdBefore := newCredential(orderedUser)
	lockUser()
	createPID, created := async(func(s *authstore.Store) error { return s.CreateSession(ctx, verified, createdBefore.session) })
	waitBlocked(createPID, lockerPID)
	allPID, allDone := async(func(s *authstore.Store) error { return s.RevokeAllSessions(ctx, caller.identity) })
	waitBlocked(allPID, createPID)
	unlock()
	finish(created, nil)
	finish(allDone, nil)
	assertHTTP("GET", "/v1/auth/me", "", createdBefore.token, 401)
	assertHTTP("GET", "/v1/auth/me", "", caller.token, 401)

	caller = issue(orderedUser)
	createdAfter := newCredential(orderedUser)
	lockUser()
	allPID, allDone = async(func(s *authstore.Store) error { return s.RevokeAllSessions(ctx, caller.identity) })
	waitBlocked(allPID, lockerPID)
	createPID, created = async(func(s *authstore.Store) error { return s.CreateSession(ctx, verified, createdAfter.session) })
	waitBlocked(createPID, allPID)
	unlock()
	finish(allDone, nil)
	finish(created, nil)
	assertHTTP("GET", "/v1/auth/me", "", caller.token, 401)
	assertHTTP("GET", "/v1/auth/me", "", createdAfter.token, 200)
	assertHTTP("GET", "/v1/auth/me", "", b1.token, 200)
	t.Log("PASS both login/all lock orderings: pre-ordered creation revoked, later password-authenticated creation is a new valid session")

	for _, state := range []string{"expired", "revoked", "inactive"} {
		caller = issue(userA)
		sibling := issue(userA)
		lockUser()
		pid, done := async(func(s *authstore.Store) error { return s.RevokeAllSessions(ctx, caller.identity) })
		waitBlocked(pid, lockerPID)
		switch state {
		case "expired":
			execSQL("UPDATE auth_sessions SET expires_at=clock_timestamp()+interval '500 milliseconds' WHERE id=$1", caller.session.ID)
			waitUntil(func() bool {
				return count("SELECT count(*) FROM auth_sessions WHERE id=$1 AND expires_at<=clock_timestamp()", caller.session.ID) == 1
			})
		case "revoked":
			execSQL("UPDATE auth_sessions SET revoked_at=clock_timestamp() WHERE id=$1", caller.session.ID)
		case "inactive":
			execSQL("UPDATE users SET is_active=false WHERE id=$1", userA.ID)
		}
		unlock()
		finish(done, auth.ErrUnauthenticated)
		if count("SELECT count(*) FROM auth_sessions WHERE id=$1 AND revoked_at IS NULL", sibling.session.ID) != 1 {
			t.Fatal("invalidated waiter revoked sibling sessions")
		}
		if state == "inactive" {
			execSQL("UPDATE users SET is_active=true WHERE id=$1", userA.ID)
		}
	}
	// Wait on the caller row itself as well as the user row before sampling time.
	caller = issue(userA)
	execSQL("BEGIN")
	execSQL("SELECT id FROM auth_sessions WHERE id=$1 FOR UPDATE", caller.session.ID)
	pid, done := async(func(s *authstore.Store) error { return s.RevokeAllSessions(ctx, caller.identity) })
	waitBlocked(pid, lockerPID)
	execSQL("UPDATE auth_sessions SET expires_at=clock_timestamp()+interval '500 milliseconds' WHERE id=$1", caller.session.ID)
	waitUntil(func() bool {
		return count("SELECT count(*) FROM auth_sessions WHERE id=$1 AND expires_at<=clock_timestamp()", caller.session.ID) == 1
	})
	unlock()
	finish(done, auth.ErrUnauthenticated)
	t.Log("PASS caller expiry/revocation/activation rechecked after user/caller lock waits using fresh server time")

	caller = issue(userA)
	beforeFailure := snapshot(userA.ID)
	execSQL("ALTER TABLE auth_sessions ADD CONSTRAINT lifecycle_reject_fixture CHECK (revoked_at IS NULL) NOT VALID")
	assertHTTP("POST", "/v1/auth/logout-all", "", caller.token, 500)
	if snapshot(userA.ID) != beforeFailure {
		t.Fatal("failed revocation partially changed session/audit rows")
	}
	execSQL("ALTER TABLE auth_sessions DROP CONSTRAINT lifecycle_reject_fixture")
	lockUser()
	deadlineCtx, cancel := context.WithTimeout(ctx, 250*time.Millisecond)
	pid, done = async(func(s *authstore.Store) error { return s.RevokeAllSessions(deadlineCtx, caller.identity) })
	waitBlocked(pid, lockerPID)
	select {
	case err := <-done:
		if err == nil || errors.Is(err, auth.ErrUnauthenticated) {
			t.Fatal("canceled revocation was accepted or misclassified")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("revocation ignored request deadline")
	}
	cancel()
	unlock()
	if snapshot(userA.ID) != beforeFailure {
		t.Fatal("canceled revocation changed rows")
	}
	if snapshot(userB.ID) != beforeAllB {
		t.Fatal("lifecycle operations changed another user's sessions")
	}
	t.Log("PASS atomic failure rollback, bounded lock cancellation, ownership and retained audit history")
	return sensitive
}
