CREATE TABLE permissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code VARCHAR(100) NOT NULL UNIQUE,
    name VARCHAR(150) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_by UUID,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_by UUID,
    CONSTRAINT permissions_code_check CHECK (code ~ '^[a-z][a-z0-9_]*[.][a-z][a-z0-9_]*$'),
    CONSTRAINT permissions_created_by_fkey FOREIGN KEY (created_by)
        REFERENCES users (id) ON UPDATE NO ACTION ON DELETE NO ACTION,
    CONSTRAINT permissions_updated_by_fkey FOREIGN KEY (updated_by)
        REFERENCES users (id) ON UPDATE NO ACTION ON DELETE NO ACTION
);
