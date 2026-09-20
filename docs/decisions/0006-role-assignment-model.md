# ADR 0006: Role Assignment and RBAC Model

## Status
Accepted

## Context
Mailvex needs a clean, flexible role-based access control (RBAC) model supporting:
- Multi-tenancy: Users belong to an Organization and have specific roles within one or more Tenants.
- Platform Administrators: Cross-tenant management with full audit logging.
- Fine-grained permissions: Ability to compose custom roles from a catalog of permissions.

## Decision
1. **Model Entities**:
   - `users`: Core identity belonging to an Organization.
   - `roles`: Role definitions (e.g. `platform_owner`, `tenant_admin`, `developer`, `support`).
   - `permissions`: Atomic permissions (`domains.read`, `messages.write`, etc.).
   - `role_permissions`: Join table linking roles to atomic permissions.
   - `tenant_memberships`: Links a `user_id` to a `tenant_id` and references `role_id`.
2. **Platform Admins**:
   - Represented by `users.is_platform_admin = true` and assigned platform roles (`platform_owner`, `platform_admin`).
3. **No Redundant `user_roles` Table**:
   - Tenant-level access is mapped exclusively through `tenant_memberships.role_id`. This eliminates ambiguity between tenant membership and user roles.

## Consequences
- Every API request authenticates the user, resolves their active `tenant_id`, looks up their role via `tenant_memberships`, and verifies required permissions via `role_permissions`.
- Zero-cost permission caching in Redis sessions.
