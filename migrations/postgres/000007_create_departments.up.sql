CREATE TABLE departments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    organization_id UUID NOT NULL,
    code VARCHAR(30) NOT NULL,
    name VARCHAR(150) NOT NULL,
    parent_department_id UUID,
    description TEXT,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_by UUID,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_by UUID,
    CONSTRAINT departments_scope_code_key UNIQUE (tenant_id, organization_id, code),
    CONSTRAINT departments_scope_id_key UNIQUE (tenant_id, organization_id, id),
    CONSTRAINT departments_not_self_parent_check CHECK (parent_department_id <> id),
    CONSTRAINT departments_parent_fkey FOREIGN KEY (tenant_id, organization_id, parent_department_id)
        REFERENCES departments (tenant_id, organization_id, id) ON UPDATE NO ACTION ON DELETE NO ACTION,
    CONSTRAINT departments_tenant_id_fkey FOREIGN KEY (tenant_id)
        REFERENCES tenants (id) ON UPDATE NO ACTION ON DELETE NO ACTION,
    CONSTRAINT departments_organization_fkey FOREIGN KEY (tenant_id, organization_id)
        REFERENCES organizations (tenant_id, id) ON UPDATE NO ACTION ON DELETE NO ACTION,
    CONSTRAINT departments_created_by_fkey FOREIGN KEY (created_by)
        REFERENCES users (id) ON UPDATE NO ACTION ON DELETE NO ACTION,
    CONSTRAINT departments_updated_by_fkey FOREIGN KEY (updated_by)
        REFERENCES users (id) ON UPDATE NO ACTION ON DELETE NO ACTION
);

CREATE INDEX departments_parent_idx ON departments (tenant_id, organization_id, parent_department_id)
    WHERE parent_department_id IS NOT NULL;
