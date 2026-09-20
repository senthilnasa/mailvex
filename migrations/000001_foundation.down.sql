-- Mailvex Foundation Schema Rollback

DROP TABLE IF EXISTS system_settings;
DROP TABLE IF EXISTS audit_logs;
DROP TABLE IF EXISTS tenant_memberships;
DROP TABLE IF EXISTS role_permissions;
DROP TABLE IF EXISTS permissions;
DROP TABLE IF EXISTS roles;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS tenant_keys;
DROP TABLE IF EXISTS tenants;
DROP TABLE IF EXISTS organizations;
