-- Refuse a collision rather than taking ownership of an existing permission.
INSERT INTO permissions (id, code, name) VALUES
    ('00000000-0000-4000-8001-000000000001', 'platform.manage', 'Manage platform'),
    ('00000000-0000-4000-8001-000000000002', 'tenant.read', 'Read tenant'),
    ('00000000-0000-4000-8001-000000000003', 'membership.read', 'Read membership'),
    ('00000000-0000-4000-8001-000000000004', 'organization.read', 'Read organization'),
    ('00000000-0000-4000-8001-000000000005', 'branch.read', 'Read branch'),
    ('00000000-0000-4000-8001-000000000006', 'department.read', 'Read department'),
    ('00000000-0000-4000-8001-000000000007', 'position.read', 'Read position'),
    ('00000000-0000-4000-8001-000000000008', 'work_location.read', 'Read work location');

INSERT INTO role_permissions (role_id, permission_id)
SELECT (SELECT id FROM roles WHERE code = grants.role_code),
       (SELECT id FROM permissions WHERE code = grants.permission_code)
FROM (VALUES
    ('SUPER_ADMIN', 'platform.manage'),
    ('TENANT_ADMIN', 'tenant.read'),
    ('TENANT_ADMIN', 'membership.read'),
    ('TENANT_ADMIN', 'organization.read'),
    ('TENANT_ADMIN', 'branch.read'),
    ('TENANT_ADMIN', 'department.read'),
    ('TENANT_ADMIN', 'position.read'),
    ('TENANT_ADMIN', 'work_location.read'),
    ('HR_ADMIN', 'organization.read'),
    ('HR_ADMIN', 'branch.read'),
    ('HR_ADMIN', 'department.read'),
    ('HR_ADMIN', 'position.read'),
    ('HR_ADMIN', 'work_location.read'),
    ('MANAGER', 'organization.read'),
    ('MANAGER', 'branch.read'),
    ('MANAGER', 'department.read'),
    ('MANAGER', 'position.read'),
    ('SUPERVISOR', 'branch.read'),
    ('SUPERVISOR', 'department.read')
) AS grants(role_code, permission_code);
