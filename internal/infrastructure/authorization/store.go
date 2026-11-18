package authorization

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	access "hris-core-service/internal/usecases/authorization"
)

type Store struct {
	db      *sql.DB
	timeout time.Duration
}

func NewStore(db *sql.DB, timeout time.Duration) *Store { return &Store{db: db, timeout: timeout} }

func nullableID(id uuid.UUID) any {
	if id == uuid.Nil {
		return nil
	}
	return id
}

func (s *Store) Allows(ctx context.Context, userID uuid.UUID, permission string, scope access.Scope) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	var allowed bool
	// Join the resource owners independently of the assignment so broader grants
	// still require a valid, active target organization and branch.
	err := s.db.QueryRowContext(ctx, `SELECT EXISTS (
        SELECT 1 FROM role_assignments a
        JOIN users u ON u.id = a.user_id AND u.is_active
        JOIN roles r ON r.id = a.role_id
        JOIN role_permissions rp ON rp.role_id = r.id
        JOIN permissions p ON p.id = rp.permission_id AND p.code = $2
        LEFT JOIN tenants t ON t.id = $4::uuid AND t.is_active
        LEFT JOIN tenant_memberships m ON m.tenant_id = t.id AND m.user_id = a.user_id AND m.is_active
        LEFT JOIN organizations o ON o.tenant_id = t.id AND o.id = $5::uuid AND o.is_active
        LEFT JOIN branches b ON b.tenant_id = t.id AND b.organization_id = o.id AND b.id = $6::uuid AND b.is_active
        WHERE a.user_id = $1 AND a.is_active AND (
            ($3 = 'PLATFORM' AND a.scope_type = 'PLATFORM' AND r.role_scope = 'PLATFORM')
            OR ($3 IN ('TENANT', 'ORGANIZATION', 'BRANCH') AND r.role_scope = 'TENANT'
                AND a.tenant_id = t.id AND m.user_id IS NOT NULL
                AND ($3 = 'TENANT' OR o.id IS NOT NULL)
                AND ($3 <> 'BRANCH' OR b.id IS NOT NULL)
                AND (
                    a.scope_type = 'TENANT'
                    OR (a.scope_type = 'ORGANIZATION' AND a.organization_id = o.id)
                    OR (a.scope_type = 'BRANCH' AND a.organization_id = o.id AND a.branch_id = b.id)
                ))
        )
    )`, userID, permission, scope.Type, nullableID(scope.TenantID), nullableID(scope.OrganizationID), nullableID(scope.BranchID)).Scan(&allowed)
	return allowed, err
}
