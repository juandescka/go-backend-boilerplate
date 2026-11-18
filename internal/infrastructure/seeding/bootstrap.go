package seeding

import (
	"context"
	"database/sql"
	"errors"
)

const (
	superAdminRoleID = "00000000-0000-4000-8000-000000000001"
	bootstrapUserID  = "00000000-0000-4000-8002-000000000005"
)

func BootstrapSuperAdmin(ctx context.Context, tx *sql.Tx, email, name, hash string) error {
	// READ COMMITTED plus this lock serializes the absence check with writes
	// from other bootstrap invocations and any assignment writer.
	if _, err := tx.ExecContext(ctx, "LOCK TABLE public.role_assignments IN SHARE ROW EXCLUSIVE MODE"); err != nil {
		return errors.New("database could not serialize administrator bootstrap")
	}
	var eligible bool
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS (
		SELECT 1 FROM public.roles r JOIN public.role_permissions rp ON rp.role_id=r.id
		JOIN public.permissions p ON p.id=rp.permission_id
		WHERE r.id=$1 AND r.code='SUPER_ADMIN' AND r.role_scope='PLATFORM' AND p.code='platform.manage'
		  AND (SELECT count(*) FROM public.role_permissions WHERE role_id=r.id)=1
	)`, superAdminRoleID).Scan(&eligible); err != nil || !eligible {
		return errors.New("required platform administrator role and grant are unavailable")
	}
	// An inactive assignment still counts. The reserved user also prevents a
	// rerun if an operator manually removes only the original assignment.
	var initialized bool
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS (
		SELECT 1 FROM public.role_assignments WHERE role_id=$1 AND scope_type='PLATFORM'
	) OR EXISTS (SELECT 1 FROM public.users WHERE id=$2)`, superAdminRoleID, bootstrapUserID).Scan(&initialized); err != nil {
		return errors.New("database could not verify initial administrator state")
	}
	if initialized {
		return errors.New("initial platform administrator bootstrap has already been recorded")
	}
	result, err := tx.ExecContext(ctx, `INSERT INTO public.users (id, email, display_name, password_hash, is_active)
		VALUES ($1, lower(btrim($2)), $3, $4, true) ON CONFLICT (lower(btrim(email))) DO NOTHING`,
		bootstrapUserID, email, name, hash)
	if err != nil {
		return errors.New("database could not create the initial administrator account")
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return errors.New("database did not report the administrator insert result")
	}
	if rows != 1 {
		return errors.New("email already belongs to an existing account; bootstrap requires a new account")
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO public.role_assignments (user_id, role_id, scope_type)
		VALUES ($1, $2, 'PLATFORM')`, bootstrapUserID, superAdminRoleID); err != nil {
		return errors.New("database could not create the initial platform assignment")
	}
	return nil
}
