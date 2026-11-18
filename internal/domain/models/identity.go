package models

import (
	"time"

	"github.com/google/uuid"
)

// User is the global M5A account, distinct from tenant/employee identity.
// Persistence models are not HTTP response types; hashes never serialize.
type User struct {
	ID              uuid.UUID  `json:"-"`
	Email           string     `json:"-"`
	PasswordHash    *string    `json:"-"`
	DisplayName     string     `json:"-"`
	EmailVerifiedAt *time.Time `json:"-"`
	LastLoginAt     *time.Time `json:"-"`
	IsActive        bool       `json:"-"`
	CreatedAt       time.Time  `json:"-"`
	CreatedBy       *uuid.UUID `json:"-"`
	UpdatedAt       time.Time  `json:"-"`
	UpdatedBy       *uuid.UUID `json:"-"`
}

func (User) TableName() string { return "users" }

type AuthSession struct {
	ID         uuid.UUID  `json:"-"`
	UserID     uuid.UUID  `json:"-"`
	TokenHash  string     `json:"-"`
	DeviceName *string    `json:"-"`
	IPAddress  *string    `json:"-"`
	UserAgent  *string    `json:"-"`
	ExpiresAt  time.Time  `json:"-"`
	RevokedAt  *time.Time `json:"-"`
	LastUsedAt *time.Time `json:"-"`
	CreatedAt  time.Time  `json:"-"`
	CreatedBy  *uuid.UUID `json:"-"`
	UpdatedAt  time.Time  `json:"-"`
	UpdatedBy  *uuid.UUID `json:"-"`
}

func (AuthSession) TableName() string { return "auth_sessions" }

// Identity contains only the fields required by the authenticated request.
type Identity struct {
	UserID          uuid.UUID
	SessionID       uuid.UUID
	Email           string
	DisplayName     string
	EmailVerifiedAt *time.Time
}
