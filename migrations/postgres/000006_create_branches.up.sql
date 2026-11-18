CREATE TABLE branches (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    organization_id UUID NOT NULL,
    code VARCHAR(30) NOT NULL,
    name VARCHAR(150) NOT NULL,
    description TEXT,
    is_head_office BOOLEAN NOT NULL DEFAULT FALSE,
    address TEXT,
    latitude NUMERIC(10,7),
    longitude NUMERIC(10,7),
    timezone VARCHAR(60) NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_by UUID,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_by UUID,
    CONSTRAINT branches_scope_code_key UNIQUE (tenant_id, organization_id, code),
    -- The scope key lets work locations reject branches from other organizations.
    CONSTRAINT branches_scope_id_key UNIQUE (tenant_id, organization_id, id),
    CONSTRAINT branches_latitude_check CHECK (latitude BETWEEN -90 AND 90),
    CONSTRAINT branches_longitude_check CHECK (longitude BETWEEN -180 AND 180),
    CONSTRAINT branches_tenant_id_fkey FOREIGN KEY (tenant_id)
        REFERENCES tenants (id) ON UPDATE NO ACTION ON DELETE NO ACTION,
    CONSTRAINT branches_organization_fkey FOREIGN KEY (tenant_id, organization_id)
        REFERENCES organizations (tenant_id, id) ON UPDATE NO ACTION ON DELETE NO ACTION,
    CONSTRAINT branches_created_by_fkey FOREIGN KEY (created_by)
        REFERENCES users (id) ON UPDATE NO ACTION ON DELETE NO ACTION,
    CONSTRAINT branches_updated_by_fkey FOREIGN KEY (updated_by)
        REFERENCES users (id) ON UPDATE NO ACTION ON DELETE NO ACTION
);

-- Regional offices may coexist; head-office status is intentionally not unique.
