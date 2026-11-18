package authentication

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"hris-core-service/internal/domain/models"
	"hris-core-service/internal/shared/password"
)

var (
	ErrUnauthenticated = errors.New("authentication failed")
	ErrUnavailable     = errors.New("authentication unavailable")
	ErrBusy            = errors.New("password verification capacity reached")
)

// Store is the small persistence boundary consumed by authentication.
// Missing records return ErrUnauthenticated; operational failures stay errors.
type Store interface {
	NormalizeLoginIdentifier(context.Context, string) (string, error)
	FindUserByEmail(context.Context, string) (models.User, error)
	CreateSession(context.Context, models.User, models.AuthSession) error
	FindIdentity(context.Context, string) (models.Identity, error)
	RevokeSession(context.Context, models.Identity) error
	RevokeAllSessions(context.Context, models.Identity) error
}

// NormalizeLoginIdentifier uses the database's locale and normalization rules,
// rather than approximating its identity boundary with Go Unicode case folding.
func (s *Service) NormalizeLoginIdentifier(ctx context.Context, email string) (string, error) {
	value, err := s.store.NormalizeLoginIdentifier(ctx, email)
	if err != nil || ctx.Err() != nil || value == "" {
		return "", ErrUnavailable
	}
	return value, nil
}

type Service struct {
	store         Store
	lifetime      time.Duration
	passwordSlots chan struct{}
}

func NewService(store Store, lifetime time.Duration) *Service {
	return &Service{store: store, lifetime: lifetime, passwordSlots: make(chan struct{}, 2)}
}

type Metadata struct{ IPAddress, UserAgent string }

type LoginResult struct {
	AccessToken string
	ExpiresAt   time.Time
	User        models.User
}

func (s *Service) Login(ctx context.Context, email, secret string, metadata Metadata) (LoginResult, error) {
	if !password.ValidInput(secret) {
		return LoginResult{}, ErrUnauthenticated
	}
	if ctx.Err() != nil {
		return LoginResult{}, ErrUnavailable
	}
	// Fixed process-local resource bound, not a brute-force/rate-limit policy.
	select {
	case s.passwordSlots <- struct{}{}:
		defer func() { <-s.passwordSlots }()
	default:
		return LoginResult{}, ErrBusy
	}
	user, err := s.store.FindUserByEmail(ctx, email)
	if err != nil && !errors.Is(err, ErrUnauthenticated) {
		return LoginResult{}, ErrUnavailable
	}
	encoded := ""
	if user.PasswordHash != nil {
		encoded = *user.PasswordHash
	}
	valid := password.Verify(encoded, secret)
	if ctx.Err() != nil {
		return LoginResult{}, ErrUnavailable
	}
	if err != nil || !valid || !user.IsActive || user.ID == uuid.Nil {
		return LoginResult{}, ErrUnauthenticated
	}
	token, hash, err := newToken()
	if err != nil {
		return LoginResult{}, ErrUnavailable
	}
	now := time.Now().UTC()
	session := models.AuthSession{ID: uuid.New(), UserID: user.ID, TokenHash: hash, ExpiresAt: now.Add(s.lifetime), CreatedAt: now, UpdatedAt: now, CreatedBy: &user.ID, UpdatedBy: &user.ID}
	if metadata.IPAddress != "" {
		session.IPAddress = &metadata.IPAddress
	}
	if metadata.UserAgent != "" {
		session.UserAgent = &metadata.UserAgent
	}
	if err := s.store.CreateSession(ctx, user, session); err != nil {
		if errors.Is(err, ErrUnauthenticated) {
			return LoginResult{}, ErrUnauthenticated
		}
		return LoginResult{}, ErrUnavailable
	}
	return LoginResult{AccessToken: token, ExpiresAt: session.ExpiresAt, User: user}, nil
}

func (s *Service) Authenticate(ctx context.Context, token string) (models.Identity, error) {
	if !ValidToken(token) {
		return models.Identity{}, ErrUnauthenticated
	}
	identity, err := s.store.FindIdentity(ctx, TokenHash(token))
	if errors.Is(err, ErrUnauthenticated) {
		return models.Identity{}, ErrUnauthenticated
	}
	if err != nil {
		return models.Identity{}, ErrUnavailable
	}
	if identity.UserID == uuid.Nil || identity.SessionID == uuid.Nil {
		return models.Identity{}, ErrUnauthenticated
	}
	return identity, nil
}

func (s *Service) Logout(ctx context.Context, identity models.Identity) error {
	return s.logout(ctx, identity, false)
}

func (s *Service) LogoutAll(ctx context.Context, identity models.Identity) error {
	return s.logout(ctx, identity, true)
}

func (s *Service) logout(ctx context.Context, identity models.Identity, all bool) error {
	if identity.UserID == uuid.Nil || identity.SessionID == uuid.Nil {
		return ErrUnauthenticated
	}
	var err error
	if all {
		err = s.store.RevokeAllSessions(ctx, identity)
	} else {
		err = s.store.RevokeSession(ctx, identity)
	}
	if errors.Is(err, ErrUnauthenticated) {
		return ErrUnauthenticated
	}
	if err != nil {
		return ErrUnavailable
	}
	return nil
}

type identityKey struct{}

func WithIdentity(ctx context.Context, identity models.Identity) context.Context {
	return context.WithValue(ctx, identityKey{}, identity)
}

func IdentityFromContext(ctx context.Context) (models.Identity, bool) {
	identity, ok := ctx.Value(identityKey{}).(models.Identity)
	return identity, ok && identity.UserID != uuid.Nil && identity.SessionID != uuid.Nil
}
