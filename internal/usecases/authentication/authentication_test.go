package authentication

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"hris-core-service/internal/domain/models"
	"hris-core-service/internal/shared/password"
)

type testStore struct {
	user                            models.User
	findErr, createErr, identityErr error
	sessions                        []models.AuthSession
	identity                        models.Identity
	lookups                         int
}

func (s *testStore) NormalizeLoginIdentifier(_ context.Context, value string) (string, error) {
	return strings.ToLower(strings.Trim(value, " ")), nil
}
func (s *testStore) FindUserByEmail(context.Context, string) (models.User, error) {
	return s.user, s.findErr
}
func (s *testStore) CreateSession(_ context.Context, _ models.User, session models.AuthSession) error {
	if s.createErr == nil {
		s.sessions = append(s.sessions, session)
	}
	return s.createErr
}
func (s *testStore) FindIdentity(context.Context, string) (models.Identity, error) {
	s.lookups++
	return s.identity, s.identityErr
}

func (s *testStore) RevokeSession(context.Context, models.Identity) error {
	return errors.New("unexpected revocation")
}
func (s *testStore) RevokeAllSessions(context.Context, models.Identity) error {
	return errors.New("unexpected revocation")
}

func TestLoginOutcomesAndHashOnlySession(t *testing.T) {
	secret := uuid.NewString()
	hash, err := password.Hash(secret)
	if err != nil {
		t.Fatal(err)
	}
	user := models.User{ID: uuid.New(), Email: "user@example.com", DisplayName: "Test", PasswordHash: &hash, IsActive: true}
	for _, tc := range []struct {
		name                     string
		user                     models.User
		findErr, createErr, want error
		secret                   string
	}{
		{"valid", user, nil, nil, nil, secret},
		{"wrong password", user, nil, nil, ErrUnauthenticated, "incorrect"},
		{"unknown email", models.User{}, ErrUnauthenticated, nil, ErrUnauthenticated, secret},
		{"inactive", func() models.User { u := user; u.IsActive = false; return u }(), nil, nil, ErrUnauthenticated, secret},
		{"no password", func() models.User { u := user; u.PasswordHash = nil; return u }(), nil, nil, ErrUnauthenticated, secret},
		{"malformed hash", func() models.User { u := user; v := "invalid"; u.PasswordHash = &v; return u }(), nil, nil, ErrUnauthenticated, secret},
		{"database error", user, errors.New("private database details"), nil, ErrUnavailable, secret},
		{"session failure", user, nil, errors.New("private SQL"), ErrUnavailable, secret},
		{"state changed", user, nil, ErrUnauthenticated, ErrUnauthenticated, secret},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store := &testStore{user: tc.user, findErr: tc.findErr, createErr: tc.createErr}
			service := NewService(store, 15*time.Minute)
			before := time.Now().UTC()
			result, err := service.Login(context.Background(), user.Email, tc.secret, Metadata{IPAddress: "192.0.2.1", UserAgent: "Test"})
			if !errors.Is(err, tc.want) {
				t.Fatalf("unexpected result: %v", err)
			}
			if tc.want != nil {
				if result.AccessToken != "" || len(store.sessions) != 0 {
					t.Fatal("failed login emitted token/session")
				}
				return
			}
			if !ValidToken(result.AccessToken) || len(store.sessions) != 1 {
				t.Fatal("valid login failed")
			}
			session := store.sessions[0]
			if session.TokenHash != TokenHash(result.AccessToken) || session.TokenHash == result.AccessToken || session.UserID != user.ID || session.ExpiresAt.Before(before.Add(15*time.Minute)) || session.ExpiresAt.After(time.Now().Add(15*time.Minute)) || session.CreatedBy == nil || *session.CreatedBy != user.ID {
				t.Fatal("session contract incorrect")
			}
		})
	}
}

func TestTokenFormatHashAndAuthenticationContext(t *testing.T) {
	if TokenHash("abc") != "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad" {
		t.Fatal("SHA-256 verifier encoding incorrect")
	}
	if ValidToken("AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAB") {
		t.Fatal("noncanonical base64url padding bits accepted")
	}
	seen := map[string]bool{}
	for i := 0; i < 100; i++ {
		token, hash, err := newToken()
		if err != nil || !ValidToken(token) || len(hash) != 64 || hash != TokenHash(token) || seen[token] {
			t.Fatal("token contract failed")
		}
		seen[token] = true
	}
	store := &testStore{identity: models.Identity{UserID: uuid.New(), SessionID: uuid.New()}}
	service := NewService(store, time.Minute)
	for _, invalid := range []string{"", "abc", " ", "Bearer token", string(make([]byte, 43))} {
		if _, err := service.Authenticate(context.Background(), invalid); !errors.Is(err, ErrUnauthenticated) {
			t.Fatal("malformed token accepted")
		}
	}
	if store.lookups != 0 {
		t.Fatal("invalid token reached persistence")
	}
	token, _, _ := newToken()
	identity, err := service.Authenticate(context.Background(), token)
	if err != nil {
		t.Fatal(err)
	}
	ctx := WithIdentity(context.Background(), identity)
	got, ok := IdentityFromContext(ctx)
	if !ok || got.UserID != identity.UserID {
		t.Fatal("identity context missing")
	}
	if _, ok := IdentityFromContext(context.Background()); ok {
		t.Fatal("identity leaked between requests")
	}
	store.identityErr = errors.New("private failure")
	if _, err := service.Authenticate(context.Background(), token); !errors.Is(err, ErrUnavailable) {
		t.Fatal("DB error mapped to invalid credentials")
	}
	store.identityErr = ErrUnauthenticated
	if _, err := service.Authenticate(context.Background(), token); !errors.Is(err, ErrUnauthenticated) {
		t.Fatal("unknown session accepted")
	}
}

type blockingStore struct {
	entered chan struct{}
	release chan struct{}
}

func (s *blockingStore) NormalizeLoginIdentifier(_ context.Context, value string) (string, error) {
	return value, nil
}
func (s *blockingStore) FindUserByEmail(ctx context.Context, _ string) (models.User, error) {
	s.entered <- struct{}{}
	select {
	case <-s.release:
		return models.User{}, ErrUnauthenticated
	case <-ctx.Done():
		return models.User{}, ctx.Err()
	}
}
func (s *blockingStore) CreateSession(context.Context, models.User, models.AuthSession) error {
	return errors.New("unexpected write")
}
func (s *blockingStore) FindIdentity(context.Context, string) (models.Identity, error) {
	return models.Identity{}, ErrUnauthenticated
}

func (s *blockingStore) RevokeSession(context.Context, models.Identity) error {
	return errors.New("unexpected revocation")
}
func (s *blockingStore) RevokeAllSessions(context.Context, models.Identity) error {
	return errors.New("unexpected revocation")
}

func TestLoginCapacityAndCancellation(t *testing.T) {
	store := &blockingStore{entered: make(chan struct{}, 2), release: make(chan struct{})}
	service := NewService(store, time.Minute)
	released := false
	defer func() {
		if !released {
			close(store.release)
		}
	}()
	results := make(chan error, 2)
	for i := 0; i < 2; i++ {
		go func() {
			_, err := service.Login(context.Background(), "user@example.com", "test", Metadata{})
			results <- err
		}()
	}
	for i := 0; i < 2; i++ {
		select {
		case <-store.entered:
		case <-time.After(5 * time.Second):
			t.Fatal("concurrent logins did not enter persistence")
		}
	}
	if _, err := service.Login(context.Background(), "user@example.com", "test", Metadata{}); !errors.Is(err, ErrBusy) {
		t.Fatal("excess concurrent login was not bounded")
	}
	close(store.release)
	released = true
	for i := 0; i < 2; i++ {
		if err := <-results; !errors.Is(err, ErrUnauthenticated) {
			t.Fatal("blocked credential denial incorrect")
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := service.Login(ctx, "user@example.com", "test", Metadata{}); !errors.Is(err, ErrUnavailable) {
		t.Fatal("canceled request accepted")
	}
}
