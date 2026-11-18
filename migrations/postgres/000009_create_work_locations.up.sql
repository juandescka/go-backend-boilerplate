CREATE TABLE work_locations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    organization_id UUID NOT NULL,
    code VARCHAR(30) NOT NULL,
    name VARCHAR(150) NOT NULL,
    branch_id UUID,
    address TEXT,
    latitude NUMERIC(10,7),
    longitude NUMERIC(10,7),
    geofence_radius_meters INTEGER,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_by UUID,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_by UUID,
    CONSTRAINT work_locations_scope_code_key UNIQUE (tenant_id, organization_id, code),
    CONSTRAINT work_locations_branch_fkey FOREIGN KEY (tenant_id, organization_id, branch_id)
        REFERENCES branches (tenant_id, organization_id, id) ON UPDATE NO ACTION ON DELETE NO ACTION,
    CONSTRAINT work_locations_latitude_check CHECK (latitude BETWEEN -90 AND 90),
    CONSTRAINT work_locations_longitude_check CHECK (longitude BETWEEN -180 AND 180),
    CONSTRAINT work_locations_radius_check CHECK (geofence_radius_meters > 0),
    CONSTRAINT work_locations_tenant_id_fkey FOREIGN KEY (tenant_id)
        REFERENCES tenants (id) ON UPDATE NO ACTION ON DELETE NO ACTION,
    CONSTRAINT work_locations_organization_fkey FOREIGN KEY (tenant_id, organization_id)
        REFERENCES organizations (tenant_id, id) ON UPDATE NO ACTION ON DELETE NO ACTION,
    CONSTRAINT work_locations_created_by_fkey FOREIGN KEY (created_by)
        REFERENCES users (id) ON UPDATE NO ACTION ON DELETE NO ACTION,
    CONSTRAINT work_locations_updated_by_fkey FOREIGN KEY (updated_by)
        REFERENCES users (id) ON UPDATE NO ACTION ON DELETE NO ACTION
);

CREATE INDEX work_locations_branch_idx ON work_locations (tenant_id, organization_id, branch_id)
    WHERE branch_id IS NOT NULL;
