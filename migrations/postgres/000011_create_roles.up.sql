CREATE TABLE roles (
    id UUID PRIMARY KEY,
    code VARCHAR(30) NOT NULL UNIQUE,
    name VARCHAR(100) NOT NULL,
    role_scope VARCHAR(10) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_by UUID,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_by UUID,
    -- Fixed identities make scope compatibility enforceable without mutable policy triggers.
    CONSTRAINT roles_system_catalog_check CHECK (
        (id = '00000000-0000-4000-8000-000000000001' AND code = 'SUPER_ADMIN' AND role_scope = 'PLATFORM')
        OR (id = '00000000-0000-4000-8000-000000000002' AND code = 'TENANT_ADMIN' AND role_scope = 'TENANT')
        OR (id = '00000000-0000-4000-8000-000000000003' AND code = 'HR_ADMIN' AND role_scope = 'TENANT')
        OR (id = '00000000-0000-4000-8000-000000000004' AND code = 'MANAGER' AND role_scope = 'TENANT')
        OR (id = '00000000-0000-4000-8000-000000000005' AND code = 'SUPERVISOR' AND role_scope = 'TENANT')
        OR (id = '00000000-0000-4000-8000-000000000006' AND code = 'EMPLOYEE' AND role_scope = 'TENANT')
    ),
    CONSTRAINT roles_created_by_fkey FOREIGN KEY (created_by)
        REFERENCES users (id) ON UPDATE NO ACTION ON DELETE NO ACTION,
    CONSTRAINT roles_updated_by_fkey FOREIGN KEY (updated_by)
        REFERENCES users (id) ON UPDATE NO ACTION ON DELETE NO ACTION
);

INSERT INTO roles (id, code, name, role_scope) VALUES
    ('00000000-0000-4000-8000-000000000001', 'SUPER_ADMIN', 'Super administrator', 'PLATFORM'),
    ('00000000-0000-4000-8000-000000000002', 'TENANT_ADMIN', 'Tenant administrator', 'TENANT'),
    ('00000000-0000-4000-8000-000000000003', 'HR_ADMIN', 'HR administrator', 'TENANT'),
    ('00000000-0000-4000-8000-000000000004', 'MANAGER', 'Manager', 'TENANT'),
    ('00000000-0000-4000-8000-000000000005', 'SUPERVISOR', 'Supervisor', 'TENANT'),
    ('00000000-0000-4000-8000-000000000006', 'EMPLOYEE', 'Employee', 'TENANT');
