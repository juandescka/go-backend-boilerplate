CREATE TABLE organizations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    parent_organization_id UUID,
    code VARCHAR(30) NOT NULL,
    name VARCHAR(150) NOT NULL,
    organization_type VARCHAR(30) NOT NULL,
    legal_name VARCHAR(200),
    registration_number VARCHAR(60),
    email VARCHAR(150),
    phone VARCHAR(30),
    address TEXT,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_by UUID,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_by UUID,
    CONSTRAINT organizations_tenant_code_key UNIQUE (tenant_id, code),
    -- PostgreSQL needs this scope key to enforce same-tenant parent/child FKs.
    CONSTRAINT organizations_tenant_id_key UNIQUE (tenant_id, id),
    CONSTRAINT organizations_type_check CHECK (
        organization_type IN ('COMPANY', 'SCHOOL', 'GOVERNMENT', 'FOUNDATION', 'OTHER')
    ),
    CONSTRAINT organizations_not_self_parent_check CHECK (parent_organization_id <> id),
    CONSTRAINT organizations_tenant_id_fkey FOREIGN KEY (tenant_id)
        REFERENCES tenants (id) ON UPDATE NO ACTION ON DELETE NO ACTION,
    CONSTRAINT organizations_parent_fkey FOREIGN KEY (tenant_id, parent_organization_id)
        REFERENCES organizations (tenant_id, id) ON UPDATE NO ACTION ON DELETE NO ACTION,
    CONSTRAINT organizations_created_by_fkey FOREIGN KEY (created_by)
        REFERENCES users (id) ON UPDATE NO ACTION ON DELETE NO ACTION,
    CONSTRAINT organizations_updated_by_fkey FOREIGN KEY (updated_by)
        REFERENCES users (id) ON UPDATE NO ACTION ON DELETE NO ACTION
);

CREATE INDEX organizations_parent_idx ON organizations (tenant_id, parent_organization_id)
    WHERE parent_organization_id IS NOT NULL;
