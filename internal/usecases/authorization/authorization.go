package authorization

import (
	"context"
	"errors"

	"github.com/google/uuid"
	auth "hris-core-service/internal/usecases/authentication"
)

var (
	ErrForbidden   = errors.New("authorization denied")
	ErrUnavailable = errors.New("authorization unavailable")
)

// Scope is the verified ownership tuple of the resource being accessed.
// Zero UUIDs represent absent owners. Containment uses the stored owner relationships.
type Scope struct {
	Type           string
	TenantID       uuid.UUID
	OrganizationID uuid.UUID
	BranchID       uuid.UUID
}

func (s Scope) Valid() bool {
	switch s.Type {
	case "PLATFORM":
		return s.TenantID == uuid.Nil && s.OrganizationID == uuid.Nil && s.BranchID == uuid.Nil
	case "TENANT":
		return s.TenantID != uuid.Nil && s.OrganizationID == uuid.Nil && s.BranchID == uuid.Nil
	case "ORGANIZATION":
		return s.TenantID != uuid.Nil && s.OrganizationID != uuid.Nil && s.BranchID == uuid.Nil
	case "BRANCH":
		return s.TenantID != uuid.Nil && s.OrganizationID != uuid.Nil && s.BranchID != uuid.Nil
	default:
		return false
	}
}

// Store is the query boundary used by the permission evaluator.
type Store interface {
	Allows(context.Context, uuid.UUID, string, Scope) (bool, error)
}

type Service struct{ store Store }

func NewService(store Store) *Service { return &Service{store: store} }

// Authorize returns a scoped context only after a successful database decision.
func (s *Service) Authorize(ctx context.Context, permission string, scope Scope) (context.Context, error) {
	identity, ok := auth.IdentityFromContext(ctx)
	if !ok {
		return nil, auth.ErrUnauthenticated
	}
	if !scope.Valid() || permission == "" || len(permission) > 100 {
		return nil, ErrForbidden
	}
	if s == nil || s.store == nil {
		return nil, ErrUnavailable
	}
	allowed, err := s.store.Allows(ctx, identity.UserID, permission, scope)
	if err != nil {
		return nil, ErrUnavailable
	}
	if !allowed {
		return nil, ErrForbidden
	}
	return context.WithValue(ctx, scopeKey{}, scope), nil
}

type scopeKey struct{}

// ScopeFromContext reads the owner tuple validated for the current operation.
func ScopeFromContext(ctx context.Context) (Scope, bool) {
	scope, ok := ctx.Value(scopeKey{}).(Scope)
	return scope, ok && scope.Valid()
}
