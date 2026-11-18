# Tenant Membership & RBAC Schema

## APPROVED DESIGN DECISIONS

The user recovered the Module B design on 2026-11-18 from the 2026-11-09 discussions. This document records that recovery before new migrations. The later recovery supersedes the initial M7 prompt: **role_assignments has neither assigned_at nor expires_at**. Existing global [identity/session](identity-session-schema.md) and [organizational](tenancy-organization-schema.md) contracts remain unchanged.

Membership links global users to tenants, independently of employment, with UNIQUE (tenant_id,user_id). Membership and authentication grant no permission. Roles and permissions are global; roles and role_permissions have no tenant_id. Assignments use user_id directly and the exact columns listed below, including is_active. Six system roles exist; no custom roles, implicit SUPER_ADMIN bypass, default account, automatic membership or privileged assignment is permitted.

| Role code | role_scope |
|---|---|
| SUPER_ADMIN | PLATFORM |
| TENANT_ADMIN | TENANT |
| HR_ADMIN | TENANT |
| MANAGER | TENANT |
| SUPERVISOR | TENANT |
| EMPLOYEE | TENANT |

| scope_type | tenant_id | organization_id | branch_id | Compatible role_scope |
|---|---|---|---|---|
| PLATFORM | NULL | NULL | NULL | PLATFORM |
| TENANT | Required | NULL | NULL | TENANT |
| ORGANIZATION | Required | Required | NULL | TENANT |
| BRANCH | Required | Required | Required | TENANT |

Tenant, organization and branch ownership must agree. Platform assignments do not require membership. Inactive membership or assignment grants nothing. Permission codes are globally unique; grants can come only from explicit role_permissions.

## TECHNICAL IMPLEMENTATION DECISIONS

These are compatible implementation choices made now, not historical business approvals. All UUID PKs default to gen_random_uuid(), except the deterministic role catalog described below. All FKs use immediate, nondeferrable ON UPDATE/DELETE NO ACTION; no cascading deletion. Shared audit fields are created_at/updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP and nullable created_by/updated_by UUID FKs to users(id). Writers maintain updated_at; no timestamp trigger is added. Activation flags default TRUE. No speculative employee, expiration or soft-delete columns exist.

### tenant_memberships

| Column | Type | Constraint |
|---|---|---|
| id | UUID | PK, generated |
| tenant_id | UUID | NOT NULL, FK tenants(id) |
| user_id | UUID | NOT NULL, FK users(id) |
| is_active | BOOLEAN | NOT NULL DEFAULT TRUE |
| created_at, updated_at | TIMESTAMPTZ | Shared audit defaults |
| created_by, updated_by | UUID | Shared nullable actor FKs |

UNIQUE (tenant_id,user_id) retains one membership identity on deactivation/reactivation. Deactivating one membership never changes global user/session state. No role is stored here.

### roles

Columns: id UUID PK (explicit deterministic value), code VARCHAR(30) NOT NULL UNIQUE, name VARCHAR(100) NOT NULL, role_scope VARCHAR(10) NOT NULL, and shared audit fields. A CHECK fixes each (id,code,role_scope) tuple to the six system definitions. IDs are 00000000-0000-4000-8000-000000000001 through 000000000006, in the table order above. This stable identity allows assignment compatibility to be enforced without adding a role_scope column to the approved assignment column list or introducing concurrency-sensitive triggers. Names remain display metadata; they never grant access.

Migration initializes exactly six rows with explicit IDs and names. Changing system identity/scope requires a reviewed new migration and contract decision. There is no per-tenant duplication or custom role API.

### permissions

Columns: id UUID PK generated, code VARCHAR(100) NOT NULL UNIQUE, name VARCHAR(150) NOT NULL, and shared audit fields. Technical naming convention is lowercase resource.action: each segment starts with an ASCII letter and continues with ASCII letters, digits or underscores. A CHECK enforces this syntax. This defines format only; it approves no permission or business capability.

### role_permissions

Columns: role_id UUID NOT NULL FK roles(id), permission_id UUID NOT NULL FK permissions(id), and shared audit fields. Composite PK (role_id,permission_id) prevents duplicate mappings. No surrogate identity, tenant_id or implicit mapping. Migration 000015 initializes the approved RBAC Business Policy v1.0 described below.

### role_assignments

| Column | Type | Constraint |
|---|---|---|
| id | UUID | PK, generated |
| tenant_id | UUID | Nullable only for PLATFORM; FK tenants(id) |
| user_id | UUID | NOT NULL, FK users(id) |
| role_id | UUID | NOT NULL, FK roles(id) |
| scope_type | VARCHAR(12) | NOT NULL; exact scope-shape CHECK |
| organization_id | UUID | Nullable according to scope |
| branch_id | UUID | Nullable according to scope |
| is_active | BOOLEAN | NOT NULL DEFAULT TRUE |
| created_at, updated_at | TIMESTAMPTZ | Shared audit defaults |
| created_by, updated_by | UUID | Shared nullable actor FKs |

Composite (tenant_id,user_id) FK references membership. It is skipped for PLATFORM through nullable tenant_id; the independent user FK remains mandatory. The FK proves association, not activation: membership can be deactivated while retaining historical assignments, and request-time evaluation must reject it. Composite (tenant_id,organization_id) FK references organizations(tenant_id,id); (tenant_id,organization_id,branch_id) references branches(tenant_id,organization_id,id). Shape CHECKs ensure required owner columns cannot be NULL to bypass these FKs.

Compatibility CHECK permits only the fixed SUPER_ADMIN identity on PLATFORM and only the other five fixed role identities on tenant scopes. The role tuple CHECK prevents changing a referenced role's scope underneath assignments. UNIQUE NULLS NOT DISTINCT (user_id,role_id,scope_type,tenant_id,organization_id,branch_id) rejects duplicates including nullable scopes, retaining inactive identities for reactivation and allowing multiple different roles. PK/unique indexes serve references; no speculative lookup indexes are added before an actual query requires them.

## RBAC Business Policy v1.0

M7E adopted this policy on 2026-11-18 and resolved B1 (permission catalog and role matrix) and B2 (scope containment).

| Role | Explicit grants |
|---|---|
| SUPER_ADMIN | `platform.manage` |
| TENANT_ADMIN | `tenant.read`, `membership.read`, `organization.read`, `branch.read`, `department.read`, `position.read`, `work_location.read` |
| HR_ADMIN | `organization.read`, `branch.read`, `department.read`, `position.read`, `work_location.read` |
| MANAGER | `organization.read`, `branch.read`, `department.read`, `position.read` |
| SUPERVISOR | `branch.read`, `department.read` |
| EMPLOYEE | None |

The table contains the complete eight-code catalog and 19 initial mappings. Every decision requires an explicit grant.

| Assignment | Resources it can cover |
|---|---|
| PLATFORM | Platform operations only. Tenant data requires a separate tenant assignment and membership. |
| TENANT | The assigned tenant and resources owned within it, including organizations, branches and organizational master data. |
| ORGANIZATION | The assigned organization, its directly owned resources and branches, and resources owned by those branches. Child organizations are outside that scope. |
| BRANCH | The assigned branch and resources explicitly owned by it, such as branch-linked work locations. Organization-level resources and sibling branches are outside that scope. |

### Authorization boundary

After bearer authentication, `RequirePermission` receives an explicit permission code and a route-specific resolver. The resolver derives the scope from the stored resource and compares any requested owner IDs. The evaluator checks that the target tenant, organization and branch exist, belong to one another and are active. It also requires an active global user and assignment, compatible role scope and an explicit mapping. Tenant access requires active membership in the resource's tenant. A successful decision attaches the verified resource tuple to the typed request context.

Missing or invalid authentication returns 401. Missing or unknown grants, invalid ownership, inactive state and scope mismatches return a generic 403. Operational errors return a safe 500 and abort the request. One database statement evaluates each decision without a cache. The next check after a committed authorization change sees the new state. Work already authorized in flight may finish.

Business routes must supply their own resource resolver and constrain subsequent data access to the verified scope. A resource with no branch owner remains organization-scoped. Existing production routes cover global authentication, operational probes and development docs. Business ownership checks are exercised through integration fixtures.

### Migration and bootstrap

Versions 000010–000014 create the five tables and seed the six fixed role identities. Version 000015 inserts deterministic permission IDs and the 19 approved grant pairs. The migration runner records the version and reports no change on repeat apply. An existing permission ID or code collision fails the migration rather than claiming an existing row. Rollback removes the v1.0 grant pairs and their matching permission identities. The foreign key blocks rollback if a later grant still references a v1.0 permission, preserving that grant. Unrelated catalog rows, grants and user assignments survive the one-step rollback.

Apply migrations explicitly to a verified target. Startup and readiness do not apply migrations. Named local seeders provide demo accounts and memberships without assignments. The [interactive bootstrap CLI](../cli.md#initial-platform-administrator) provisions only the first local platform administrator.

### Assignment management

B3 is explicitly deferred and denied. No API creates, updates or revokes role assignments. Ordinary authentication and `platform.manage` grant no assignment-management authority. A future API requires approval of actor authority, scoped delegation, self-assignment prevention and privilege escalation safeguards.

### Verification

The disposable PostgreSQL 17 suite verified the complete seeded matrix, direct containment, child-organization denial, platform isolation, active state checks and permission removal on the next request. Bearer requests against stored work locations verified cross-tenant ID rejection and session expiry and revocation. The migration checks verified rollback and reapply, preservation of unrelated grants and assignments, and rollback refusal when a later grant depends on seeded data. See the [verification commands](../cli.md#membership--rbac-verification) and [M7E review](../../develop/reviews/tenant-access-rbac/M7E-01-review.md).
