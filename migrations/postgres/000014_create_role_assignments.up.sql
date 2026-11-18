CREATE TABLE role_assignments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID,
    user_id UUID NOT NULL,
    role_id UUID NOT NULL,
    scope_type VARCHAR(12) NOT NULL,
    organization_id UUID,
    branch_id UUID,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_by UUID,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_by UUID,
    CONSTRAINT role_assignments_scope_check CHECK (
        (scope_type = 'PLATFORM' AND tenant_id IS NULL AND organization_id IS NULL AND branch_id IS NULL)
        OR (scope_type = 'TENANT' AND tenant_id IS NOT NULL AND organization_id IS NULL AND branch_id IS NULL)
        OR (scope_type = 'ORGANIZATION' AND tenant_id IS NOT NULL AND organization_id IS NOT NULL AND branch_id IS NULL)
        OR (scope_type = 'BRANCH' AND tenant_id IS NOT NULL AND organization_id IS NOT NULL AND branch_id IS NOT NULL)
    ),
    -- Catalog identity/scope tuples are fixed by roles_system_catalog_check.
    CONSTRAINT role_assignments_role_scope_check CHECK (
        (scope_type = 'PLATFORM' AND role_id = '00000000-0000-4000-8000-000000000001')
        OR (scope_type IN ('TENANT', 'ORGANIZATION', 'BRANCH') AND role_id IN (
            '00000000-0000-4000-8000-000000000002',
            '00000000-0000-4000-8000-000000000003',
            '00000000-0000-4000-8000-000000000004',
            '00000000-0000-4000-8000-000000000005',
            '00000000-0000-4000-8000-000000000006'
        ))
    ),
    CONSTRAINT role_assignments_subject_role_scope_key
        UNIQUE NULLS NOT DISTINCT (user_id, role_id, scope_type, tenant_id, organization_id, branch_id),
    CONSTRAINT role_assignments_user_id_fkey FOREIGN KEY (user_id)
        REFERENCES users (id) ON UPDATE NO ACTION ON DELETE NO ACTION,
    CONSTRAINT role_assignments_role_id_fkey FOREIGN KEY (role_id)
        REFERENCES roles (id) ON UPDATE NO ACTION ON DELETE NO ACTION,
    CONSTRAINT role_assignments_tenant_id_fkey FOREIGN KEY (tenant_id)
        REFERENCES tenants (id) ON UPDATE NO ACTION ON DELETE NO ACTION,
    -- NULL tenant exempts only PLATFORM; the independent user FK still applies.
    CONSTRAINT role_assignments_membership_fkey FOREIGN KEY (tenant_id, user_id)
        REFERENCES tenant_memberships (tenant_id, user_id) ON UPDATE NO ACTION ON DELETE NO ACTION,
    CONSTRAINT role_assignments_organization_fkey FOREIGN KEY (tenant_id, organization_id)
        REFERENCES organizations (tenant_id, id) ON UPDATE NO ACTION ON DELETE NO ACTION,
    CONSTRAINT role_assignments_branch_fkey FOREIGN KEY (tenant_id, organization_id, branch_id)
        REFERENCES branches (tenant_id, organization_id, id) ON UPDATE NO ACTION ON DELETE NO ACTION,
    CONSTRAINT role_assignments_created_by_fkey FOREIGN KEY (created_by)
        REFERENCES users (id) ON UPDATE NO ACTION ON DELETE NO ACTION,
    CONSTRAINT role_assignments_updated_by_fkey FOREIGN KEY (updated_by)
        REFERENCES users (id) ON UPDATE NO ACTION ON DELETE NO ACTION
);
