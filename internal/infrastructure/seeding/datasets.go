package seeding

import (
	"context"
	"database/sql"
	"errors"
)

const (
	demoUserID       = "00000000-0000-4000-8002-000000000001"
	demoTenantID     = "00000000-0000-4000-8002-000000000002"
	demoOrgID        = "00000000-0000-4000-8002-000000000003"
	demoMembershipID = "00000000-0000-4000-8002-000000000004"
	demoEmail        = "mapatu-demo-user@example.invalid"
	demoTenantCode   = "MAPATU_DEMO"
	demoTenantSlug   = "mapatu-demo"
	demoOrgCode      = "MAPATU_DEMO_ORG"
)

func devUsers(ctx context.Context, tx *sql.Tx) (bool, error) {
	return insertOwned(ctx, tx, "development user",
		`INSERT INTO public.users (id, email, display_name, is_active)
		  VALUES ($1, lower(btrim($2)), 'Mapatu Demo User', false) ON CONFLICT DO NOTHING`,
		"SELECT EXISTS (SELECT 1 FROM public.users WHERE id=$1 AND lower(btrim(email))=lower(btrim($2)))",
		demoUserID, demoEmail)
}

func demoTenant(ctx context.Context, tx *sql.Tx) (bool, error) {
	return insertOwned(ctx, tx, "demo tenant",
		`INSERT INTO public.tenants (id, code, slug, name, timezone)
		  VALUES ($1, $2, $3, 'Mapatu Demo', 'Asia/Makassar') ON CONFLICT DO NOTHING`,
		"SELECT EXISTS (SELECT 1 FROM public.tenants WHERE id=$1 AND code=$2 AND slug=$3)",
		demoTenantID, demoTenantCode, demoTenantSlug)
}

func demoOrganization(ctx context.Context, tx *sql.Tx) (bool, error) {
	if err := requireDemoTenant(ctx, tx); err != nil {
		return false, err
	}
	return insertOwned(ctx, tx, "demo organization",
		`INSERT INTO public.organizations (id, tenant_id, code, name, organization_type)
		  VALUES ($1, $2, $3, 'Mapatu Demo Organization', 'COMPANY') ON CONFLICT DO NOTHING`,
		"SELECT EXISTS (SELECT 1 FROM public.organizations WHERE id=$1 AND tenant_id=$2 AND code=$3)",
		demoOrgID, demoTenantID, demoOrgCode)
}

func demoMemberships(ctx context.Context, tx *sql.Tx) (bool, error) {
	if err := requireDemoTenant(ctx, tx); err != nil {
		return false, err
	}
	if err := requireOwner(ctx, tx,
		"SELECT 1 FROM public.users WHERE id=$1 AND lower(btrim(email))=lower(btrim($2)) FOR SHARE",
		"required demo user is missing or has an identity conflict; run seed run dev-users", demoUserID, demoEmail); err != nil {
		return false, err
	}
	return insertOwned(ctx, tx, "demo membership",
		`INSERT INTO public.tenant_memberships (id, tenant_id, user_id, is_active)
		  VALUES ($1, $2, $3, true) ON CONFLICT DO NOTHING`,
		"SELECT EXISTS (SELECT 1 FROM public.tenant_memberships WHERE id=$1 AND tenant_id=$2 AND user_id=$3)",
		demoMembershipID, demoTenantID, demoUserID)
}

func requireDemoTenant(ctx context.Context, tx *sql.Tx) error {
	return requireOwner(ctx, tx,
		"SELECT 1 FROM public.tenants WHERE id=$1 AND code=$2 AND slug=$3 FOR SHARE",
		"required demo tenant is missing or has an identity conflict; run seed run demo-tenant",
		demoTenantID, demoTenantCode, demoTenantSlug)
}

func requireOwner(ctx context.Context, tx *sql.Tx, query, missing string, args ...any) error {
	var exists int
	err := tx.QueryRowContext(ctx, query, args...).Scan(&exists)
	if errors.Is(err, sql.ErrNoRows) {
		return errors.New(missing)
	}
	if err != nil {
		return errors.New("database could not verify seed dependencies")
	}
	return nil
}

func insertOwned(ctx context.Context, tx *sql.Tx, label, insert, owned string, args ...any) (bool, error) {
	result, err := tx.ExecContext(ctx, insert, args...)
	if err != nil {
		return false, errors.New("database could not insert " + label)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return false, errors.New("database did not report the insert result")
	}
	if rows == 1 {
		return true, nil
	}
	// Matching a natural key alone must never claim an unrelated row.
	var exists bool
	if err := tx.QueryRowContext(ctx, owned, args...).Scan(&exists); err != nil {
		return false, errors.New("database could not verify ownership of " + label)
	}
	if !exists {
		return false, errors.New(label + " identity conflicts with an existing record")
	}
	return false, nil
}
