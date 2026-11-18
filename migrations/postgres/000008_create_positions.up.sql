CREATE TABLE positions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    organization_id UUID NOT NULL,
    code VARCHAR(30) NOT NULL,
    name VARCHAR(150) NOT NULL,
    description TEXT,
    hierarchy_level SMALLINT,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_by UUID,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_by UUID,
    CONSTRAINT positions_scope_code_key UNIQUE (tenant_id, organization_id, code),
    CONSTRAINT positions_hierarchy_level_check CHECK (hierarchy_level >= 1),
    CONSTRAINT positions_tenant_id_fkey FOREIGN KEY (tenant_id)
        REFERENCES tenants (id) ON UPDATE NO ACTION ON DELETE NO ACTION,
    CONSTRAINT positions_organization_fkey FOREIGN KEY (tenant_id, organization_id)
        REFERENCES organizations (tenant_id, id) ON UPDATE NO ACTION ON DELETE NO ACTION,
    CONSTRAINT positions_created_by_fkey FOREIGN KEY (created_by)
        REFERENCES users (id) ON UPDATE NO ACTION ON DELETE NO ACTION,
    CONSTRAINT positions_updated_by_fkey FOREIGN KEY (updated_by)
        REFERENCES users (id) ON UPDATE NO ACTION ON DELETE NO ACTION
);
