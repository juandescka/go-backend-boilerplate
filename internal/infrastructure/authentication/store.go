package authentication

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"hris-core-service/internal/domain/models"
	auth "hris-core-service/internal/usecases/authentication"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Store struct {
	db      *gorm.DB
	timeout time.Duration
}

func NewStore(db *gorm.DB, timeout time.Duration) *Store {
	return &Store{db: db, timeout: timeout}
}

func (s *Store) NormalizeLoginIdentifier(ctx context.Context, email string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	var normalized string
	err := s.db.WithContext(ctx).Raw("SELECT lower(btrim(?::text))", email).Scan(&normalized).Error
	return normalized, err
}

func (s *Store) FindUserByEmail(ctx context.Context, email string) (models.User, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	var user models.User
	// Both sides use PostgreSQL's normalization/locale, exactly as M5A's index.
	err := s.db.WithContext(ctx).Select("id,email,password_hash,display_name,email_verified_at,is_active").Where("lower(btrim(email)) = lower(btrim(?))", email).Take(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return models.User{}, auth.ErrUnauthenticated
	}
	return user, err
}

func (s *Store) CreateSession(ctx context.Context, user models.User, session models.AuthSession) error {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// UPDATE takes a row lock and rechecks the state/hash at write time. The
		// login timestamp and session either commit together or both roll back.
		result := tx.Model(&models.User{}).Where("id = ? AND is_active = true AND password_hash = ?", user.ID, user.PasswordHash).Updates(map[string]any{"last_login_at": session.CreatedAt, "updated_at": session.CreatedAt, "updated_by": user.ID})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return auth.ErrUnauthenticated
		}
		return tx.Create(&session).Error
	})
}

func (s *Store) FindIdentity(ctx context.Context, hash string) (models.Identity, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	var identity models.Identity
	// Resolve both states together so each request observes revocation and deactivation.
	err := s.db.WithContext(ctx).Table("auth_sessions AS s").Select("s.id AS session_id,u.id AS user_id,u.email,u.display_name,u.email_verified_at").Joins("JOIN users AS u ON u.id = s.user_id").Where("s.token_hash = ? AND s.revoked_at IS NULL AND s.expires_at > CURRENT_TIMESTAMP AND u.is_active = true", hash).Take(&identity).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return models.Identity{}, auth.ErrUnauthenticated
	}
	return identity, err
}

func (s *Store) RevokeSession(ctx context.Context, identity models.Identity) error {
	return s.revoke(ctx, identity, false)
}

func (s *Store) RevokeAllSessions(ctx context.Context, identity models.Identity) error {
	return s.revoke(ctx, identity, true)
}

func (s *Store) revoke(ctx context.Context, identity models.Identity, all bool) error {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Match login's user→session lock order. For logout-all, creations committed
		// before this lock are included; a later password login is a new session.
		var user models.User
		if err := tx.Select("id").Where("id = ? AND is_active = true", identity.UserID).Clauses(clause.Locking{Strength: "UPDATE"}).Take(&user).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return auth.ErrUnauthenticated
			}
			return err
		}
		var caller models.AuthSession
		if err := tx.Select("id,expires_at,revoked_at").Where("id = ? AND user_id = ?", identity.SessionID, identity.UserID).Clauses(clause.Locking{Strength: "UPDATE"}).Take(&caller).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return auth.ErrUnauthenticated
			}
			return err
		}
		// Transaction-start time can be stale after either lock wait.
		var now time.Time
		if err := tx.Raw("SELECT clock_timestamp()").Scan(&now).Error; err != nil {
			return err
		}
		if caller.RevokedAt != nil || !caller.ExpiresAt.After(now) {
			return auth.ErrUnauthenticated
		}
		query := tx.Model(&models.AuthSession{}).Where("user_id = ? AND revoked_at IS NULL AND expires_at > ?", identity.UserID, now)
		if !all {
			query = query.Where("id = ?", identity.SessionID)
		}
		result := query.Updates(map[string]any{"revoked_at": now, "updated_at": now, "updated_by": identity.UserID})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return auth.ErrUnauthenticated
		}
		return nil
	}, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
}
