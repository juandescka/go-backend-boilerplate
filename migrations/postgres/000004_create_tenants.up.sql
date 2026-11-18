CREATE TABLE tenants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code VARCHAR(30) NOT NULL,
    name VARCHAR(150) NOT NULL,
    slug VARCHAR(100) NOT NULL,
    timezone VARCHAR(60) NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    deactivation_reason TEXT,
    deactivated_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_by UUID,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_by UUID,
    CONSTRAINT tenants_code_key UNIQUE (code),
    CONSTRAINT tenants_slug_key UNIQUE (slug),
    CONSTRAINT tenants_created_by_fkey FOREIGN KEY (created_by)
        REFERENCES users (id) ON UPDATE NO ACTION ON DELETE NO ACTION,
    CONSTRAINT tenants_updated_by_fkey FOREIGN KEY (updated_by)
        REFERENCES users (id) ON UPDATE NO ACTION ON DELETE NO ACTION
);
