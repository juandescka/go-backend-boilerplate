# Tenancy & Organization Schema

## Contract and boundaries

This is the official Module A contract recovered by the user on 2026-11-17 from design discussions on 2026-11-09, including the later tenant deactivation amendment. This contract was recorded and checked before implementing migrations. Implementation choices below are explicitly distinguished from recovered business decisions. Deployment requires explicit migrations; server startup never creates these tables.

Module A contains only tenants, organizations, branches, departments, positions and work_locations. Global [users and authentication sessions](identity-session-schema.md) remain separate. Audit actor references grant no tenant membership or authorization. Membership/RBAC, employee assignments, onboarding/default records, subscription, soft deletion and administrative APIs are outside this schema.

## Shared columns (all six tables)

| Column | PostgreSQL type | Null/default/constraint |
|---|---|---|
| id | UUID | PK; DEFAULT gen_random_uuid() |
| is_active | BOOLEAN | NOT NULL DEFAULT TRUE |
| created_at | TIMESTAMPTZ | NOT NULL DEFAULT CURRENT_TIMESTAMP |
| created_by | UUID | NULL; FK users(id) |
| updated_at | TIMESTAMPTZ | NOT NULL DEFAULT CURRENT_TIMESTAMP |
| updated_by | UUID | NULL; FK users(id) |

CURRENT_TIMESTAMP and NOW() express the same transaction-time default. UUID generation follows existing Mapatu SQL without extensions. Actor columns retain the existing approved `_by` names. Audit actors may be null for bootstrap/system operations. There is no timestamp-update trigger: future writers must set updated_at and the appropriate updated_by themselves.

## tenants

| Additional column | Type | Nullable |
|---|---|---|
| code | VARCHAR(30) | No |
| name | VARCHAR(150) | No |
| slug | VARCHAR(100) | No |
| timezone | VARCHAR(60) | No |
| deactivation_reason | TEXT | Yes |
| deactivated_at | TIMESTAMPTZ | Yes |

Unique constraints: (code), (slug). Names are not globally unique. is_active is the activation flag; there is no status or subscription_status column. Deactivation reason/time are optional metadata, not a subscription lifecycle.

## organizations

| Additional column | Type | Nullable |
|---|---|---|
| tenant_id | UUID | No |
| parent_organization_id | UUID | Yes |
| code | VARCHAR(30) | No |
| name | VARCHAR(150) | No |
| organization_type | VARCHAR(30) | No |
| legal_name | VARCHAR(200) | Yes |
| registration_number | VARCHAR(60) | Yes |
| email | VARCHAR(150) | Yes |
| phone | VARCHAR(30) | Yes |
| address | TEXT | Yes |

tenant_id references tenants(id). Unique (tenant_id, code) allows many organizations per tenant. organization_type must be COMPANY, SCHOOL, GOVERNMENT, FOUNDATION or OTHER. Nullable parent references the same table through (tenant_id, parent_organization_id) → (tenant_id, id). Supporting UNIQUE (tenant_id, id) makes the tenant boundary a PostgreSQL-enforced reference. CHECK forbids id = parent_organization_id.

## branches

| Additional column | Type | Nullable/default |
|---|---|---|
| tenant_id | UUID | No |
| organization_id | UUID | No |
| code | VARCHAR(30) | No |
| name | VARCHAR(150) | No |
| description | TEXT | Yes |
| is_head_office | BOOLEAN | No; DEFAULT FALSE |
| address | TEXT | Yes |
| latitude | NUMERIC(10,7) | Yes |
| longitude | NUMERIC(10,7) | Yes |
| timezone | VARCHAR(60) | No |

Unique (tenant_id, organization_id, code). Organization reference must match tenant. Multiple head offices, including regional offices, are allowed; no single-head-office index exists. Supporting UNIQUE (tenant_id, organization_id, id) enables scope-safe work-location branch references. Latitude/longitude use geographic degrees and are individually bounded to [-90,90] and [-180,180]. These range checks are an implementation choice consistent with coordinate semantics, not a new location service.

## departments

| Additional column | Type | Nullable |
|---|---|---|
| tenant_id | UUID | No |
| organization_id | UUID | No |
| parent_department_id | UUID | Yes |
| code | VARCHAR(30) | No |
| name | VARCHAR(150) | No |
| description | TEXT | Yes |

Unique (tenant_id, organization_id, code). No branch_id. Parent references (tenant_id, organization_id, parent_department_id) → departments(tenant_id, organization_id, id), backed by the corresponding supporting unique constraint. CHECK forbids direct self-parent. Parent must be in the same tenant AND organization.

## positions

| Additional column | Type | Nullable |
|---|---|---|
| tenant_id | UUID | No |
| organization_id | UUID | No |
| code | VARCHAR(30) | No |
| name | VARCHAR(150) | No |
| description | TEXT | Yes |
| hierarchy_level | SMALLINT | Yes |

Unique (tenant_id, organization_id, code). hierarchy_level >= 1 when supplied; smaller values represent higher positions, configured per organization. This is not RBAC. No department_id, parent_position_id or employee assignment exists.

## work_locations

| Additional column | Type | Nullable |
|---|---|---|
| tenant_id | UUID | No |
| organization_id | UUID | No |
| branch_id | UUID | Yes |
| code | VARCHAR(30) | No |
| name | VARCHAR(150) | No |
| address | TEXT | Yes |
| latitude | NUMERIC(10,7) | Yes |
| longitude | NUMERIC(10,7) | Yes |
| geofence_radius_meters | INTEGER | Yes |

Unique (tenant_id, organization_id, code). Nullable branch reference uses (tenant_id, organization_id, branch_id) → branches(tenant_id, organization_id, id). Null branch means an organization-level location; any supplied branch must match both tenant and organization. Latitude/longitude are geographic degrees bounded to [-90,90] and [-180,180]. Radius is positive meters when supplied. No timezone, location_type, GPS tracking or attendance evaluation is added.

## Relationships, constraints and indexes

All five child tables include a direct tenant_id FK to tenants. branches, departments, positions and work_locations additionally use (tenant_id, organization_id) → organizations(tenant_id, id). A single organization_id FK would allow inconsistent tenant ownership; the composite FK rejects it on inserts and updates. Organization and branch scope changes that would invalidate existing references are also rejected.

All FKs use ON UPDATE NO ACTION ON DELETE NO ACTION, immediate and nondeferrable, following the existing identity schema. No cascading business-data deletion or actor nulling occurs. Composite nullable references use PostgreSQL's default MATCH SIMPLE: only the optional parent/branch can be null because its tenant/organization columns are NOT NULL; mandatory organization/tenant FKs remain enforced independently.

PKs, business uniqueness and the three supporting scope unique constraints create B-tree indexes. Business unique indexes begin with the owner columns, so separate tenant_id/organization_id indexes would duplicate those useful prefixes. Additional partial nonunique indexes cover non-null organization parents (tenant_id, parent_organization_id), department parents (tenant_id, organization_id, parent_department_id), and location branches (tenant_id, organization_id, branch_id). No duplicate single-ID unique/index, head-office uniqueness, standalone activation or speculative search/actor indexes are added. Reassess actor indexes when a real user-deletion/actor-query workload exists.

These FK/null/check/index mechanisms follow the [PostgreSQL 17 constraint reference](https://www.postgresql.org/docs/17/ddl-constraints.html).

## Implementation choices and deferred domain validation

| Detail left open in recovered design | Selected implementation and limit |
|---|---|
| UUID and timestamps | Existing gen_random_uuid() and CURRENT_TIMESTAMP; no extensions/triggers |
| Delete/update semantics | Conservative NO ACTION on every FK; no cascades |
| Code/slug normalization | Ordinary VARCHAR uniqueness under the database collation, with no lower/trim expression, case-folding, format or nonblank rule. Values are preserved; no new canonicalization policy is inferred |
| Recursive hierarchy cycles | Domain contract forbids cycles. PostgreSQL enforces parent existence, matching scope and direct self-parent rejection only. Longer cycles are NOT prevented by these migrations. Future hierarchy writes must validate acyclicity transactionally with concurrency control before they are exposed. No write API currently exists; no recursive trigger is introduced |
| Activation/deactivation consistency | No new coupling CHECK; active state and nullable metadata remain independently representable. Lifecycle/onboarding writes need a separately defined policy |
| Coordinate pairing | Independently nullable as approved; no pair-required or radius-requires-coordinate rule. A future attendance/geofence consumer must validate usable geometry before evaluation |
| Timezone validation | Required VARCHAR(60) only for tenants/branches; no default/inheritance or IANA validation invented |

The recovered contract is sufficient for M6 schema foundation. These choices do not claim that optional business policies were previously approved. In particular, M6 is not full database enforcement of recursive acyclicity or tenant read authorization.

## Migration source and operations

| Version | Pair in migrations/postgres | Dependency |
|---|---|---|
| 000004 | [tenants up](../../migrations/postgres/000004_create_tenants.up.sql) / [down](../../migrations/postgres/000004_create_tenants.down.sql) | Existing users for audit actors |
| 000005 | [organizations up](../../migrations/postgres/000005_create_organizations.up.sql) / [down](../../migrations/postgres/000005_create_organizations.down.sql) | tenants/users |
| 000006 | [branches up](../../migrations/postgres/000006_create_branches.up.sql) / [down](../../migrations/postgres/000006_create_branches.down.sql) | organizations/tenants/users |
| 000007 | [departments up](../../migrations/postgres/000007_create_departments.up.sql) / [down](../../migrations/postgres/000007_create_departments.down.sql) | organizations/tenants/users |
| 000008 | [positions up](../../migrations/postgres/000008_create_positions.up.sql) / [down](../../migrations/postgres/000008_create_positions.down.sql) | organizations/tenants/users |
| 000009 | [work_locations up](../../migrations/postgres/000009_create_work_locations.up.sql) / [down](../../migrations/postgres/000009_create_work_locations.down.sql) | branches/organizations/tenants/users |

The matching SQL pairs are available in the migration source; execution/review evidence is recorded separately under develop/reviews/tenancy-organization. Apply up in version order. Each down drops only its corresponding table, without CASCADE. Reverse order 9→8→7→6→5→4→3 removes Module A while preserving users, auth_sessions and their token index. Rollback deletes Module A data and is tested only on owned disposable databases. Never demonstrate down on mapatu_dev or remove Docker volumes.

Use the [existing migration CLI](../cli.md): verify effective DB_* target, then explicit migrate up/version. Compose POSTGRES_* values do not select the backend database. Startup does not migrate. Version may initialize metadata. Explicit local demo seeders are documented in the [CLI](../cli.md#4-development-seeding-and-administrator-bootstrap). Applied migrations 000001–000003 are unchanged.

For reproducible disposable schema verification, use the opt-in stage commands in the [CLI reference](../cli.md#tenancy--organization-verification). The suite checks exact types/defaults/nullability, 19 indexes, 24 FKs, key relationships and lifecycle boundaries; existing authentication integration separately exercises login/session/health behavior with the full schema. Execution results and review history belong in develop/, not this technical reference.
