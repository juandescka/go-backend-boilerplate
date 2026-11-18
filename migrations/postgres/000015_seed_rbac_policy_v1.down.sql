-- Remove only the grants and permission identities seeded by this migration.
WITH seeded_permissions (id, code) AS (VALUES
    ('00000000-0000-4000-8001-000000000001'::uuid, 'platform.manage'),
    ('00000000-0000-4000-8001-000000000002'::uuid, 'tenant.read'),
    ('00000000-0000-4000-8001-000000000003'::uuid, 'membership.read'),
    ('00000000-0000-4000-8001-000000000004'::uuid, 'organization.read'),
    ('00000000-0000-4000-8001-000000000005'::uuid, 'branch.read'),
    ('00000000-0000-4000-8001-000000000006'::uuid, 'department.read'),
    ('00000000-0000-4000-8001-000000000007'::uuid, 'position.read'),
    ('00000000-0000-4000-8001-000000000008'::uuid, 'work_location.read')
), seeded_grants (role_code, permission_code) AS (VALUES
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
)
DELETE FROM role_permissions rp
USING roles r, permissions p, seeded_permissions sp, seeded_grants sg
WHERE rp.role_id = r.id AND r.code = sg.role_code
  AND rp.permission_id = p.id AND p.id = sp.id AND p.code = sp.code
  AND sp.code = sg.permission_code;

-- A later grant referencing one of these permissions blocks rollback via the FK.
DELETE FROM permissions p
USING (VALUES
    ('00000000-0000-4000-8001-000000000001'::uuid, 'platform.manage'),
    ('00000000-0000-4000-8001-000000000002'::uuid, 'tenant.read'),
    ('00000000-0000-4000-8001-000000000003'::uuid, 'membership.read'),
    ('00000000-0000-4000-8001-000000000004'::uuid, 'organization.read'),
    ('00000000-0000-4000-8001-000000000005'::uuid, 'branch.read'),
    ('00000000-0000-4000-8001-000000000006'::uuid, 'department.read'),
    ('00000000-0000-4000-8001-000000000007'::uuid, 'position.read'),
    ('00000000-0000-4000-8001-000000000008'::uuid, 'work_location.read')
) AS seeded(id, code)
WHERE p.id = seeded.id AND p.code = seeded.code;
