-- Mailvex Foundation Schema: Organizations, Tenants, RBAC, Users, Keys, Audit

CREATE TABLE IF NOT EXISTS organizations (
    id VARCHAR(36) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(100) NOT NULL UNIQUE,
    status VARCHAR(32) NOT NULL DEFAULT 'active',
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS tenants (
    id VARCHAR(36) PRIMARY KEY,
    organization_id VARCHAR(36) NOT NULL,
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(100) NOT NULL UNIQUE,
    environment VARCHAR(32) NOT NULL DEFAULT 'live',
    status VARCHAR(32) NOT NULL DEFAULT 'active',
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    CONSTRAINT fk_tenants_organization FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE,
    INDEX idx_tenants_org (organization_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS tenant_keys (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id VARCHAR(36) NOT NULL,
    key_version INT NOT NULL DEFAULT 1,
    encrypted_dek TEXT NOT NULL,
    kek_version VARCHAR(32) NOT NULL DEFAULT 'v1',
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    CONSTRAINT fk_tenant_keys_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
    UNIQUE KEY uk_tenant_key_version (tenant_id, key_version)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS users (
    id VARCHAR(36) PRIMARY KEY,
    organization_id VARCHAR(36) NOT NULL,
    email VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'active',
    is_platform_admin BOOLEAN NOT NULL DEFAULT FALSE,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    CONSTRAINT fk_users_organization FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE,
    INDEX idx_users_org (organization_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS roles (
    id VARCHAR(36) PRIMARY KEY,
    organization_id VARCHAR(36) NULL,
    name VARCHAR(64) NOT NULL,
    scope VARCHAR(32) NOT NULL DEFAULT 'tenant',
    description VARCHAR(255) NOT NULL DEFAULT '',
    is_system BOOLEAN NOT NULL DEFAULT TRUE,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    CONSTRAINT fk_roles_org FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS permissions (
    id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(64) NOT NULL UNIQUE,
    description VARCHAR(255) NOT NULL DEFAULT '',
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS role_permissions (
    role_id VARCHAR(36) NOT NULL,
    permission_id VARCHAR(64) NOT NULL,
    PRIMARY KEY (role_id, permission_id),
    CONSTRAINT fk_rp_role FOREIGN KEY (role_id) REFERENCES roles(id) ON DELETE CASCADE,
    CONSTRAINT fk_rp_permission FOREIGN KEY (permission_id) REFERENCES permissions(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS tenant_memberships (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id VARCHAR(36) NOT NULL,
    user_id VARCHAR(36) NOT NULL,
    role_id VARCHAR(36) NOT NULL,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    CONSTRAINT fk_tm_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
    CONSTRAINT fk_tm_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT fk_tm_role FOREIGN KEY (role_id) REFERENCES roles(id) ON DELETE CASCADE,
    UNIQUE KEY uk_tenant_user (tenant_id, user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS audit_logs (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id VARCHAR(36) NULL,
    org_id VARCHAR(36) NULL,
    actor_user_id VARCHAR(36) NULL,
    actor_api_key_id VARCHAR(36) NULL,
    impersonator_user_id VARCHAR(36) NULL,
    action VARCHAR(64) NOT NULL,
    resource_type VARCHAR(64) NOT NULL,
    resource_id VARCHAR(64) NOT NULL,
    before_state JSON NULL,
    after_state JSON NULL,
    reason VARCHAR(255) NULL,
    ip_address VARCHAR(45) NULL,
    user_agent VARCHAR(255) NULL,
    occurred_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    INDEX idx_audit_tenant (tenant_id, occurred_at),
    INDEX idx_audit_org (org_id, occurred_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS system_settings (
    `key` VARCHAR(128) PRIMARY KEY,
    `value` TEXT NOT NULL,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Seeds for standard permissions (§9)
INSERT IGNORE INTO permissions (id, name, description) VALUES
('tenants.read', 'tenants.read', 'Read tenant metadata and list tenants'),
('tenants.write', 'tenants.write', 'Create and modify tenants'),
('tenants.suspend', 'tenants.suspend', 'Suspend or disable tenants'),
('org.read', 'org.read', 'Read organization details'),
('org.write', 'org.write', 'Modify organization settings'),
('org.sso.write', 'org.sso.write', 'Manage SSO configuration'),
('users.read', 'users.read', 'List and view organization users'),
('users.write', 'users.write', 'Manage organization users'),
('users.invite', 'users.invite', 'Invite users to organization'),
('roles.read', 'roles.read', 'View roles and permissions'),
('roles.write', 'roles.write', 'Manage roles and assign permissions'),
('domains.read', 'domains.read', 'View domains and DNS records'),
('domains.write', 'domains.write', 'Add and update domains'),
('domains.verify', 'domains.verify', 'Trigger domain DNS verification'),
('domains.dkim.rotate', 'domains.dkim.rotate', 'Rotate domain DKIM keys'),
('domains.delete', 'domains.delete', 'Delete domains'),
('streams.read', 'streams.read', 'List message streams'),
('streams.write', 'streams.write', 'Manage message streams'),
('api_keys.read', 'api_keys.read', 'List API keys'),
('api_keys.create', 'api_keys.create', 'Create API keys'),
('api_keys.revoke', 'api_keys.revoke', 'Revoke API keys'),
('smtp_credentials.read', 'smtp_credentials.read', 'List SMTP credentials'),
('smtp_credentials.create', 'smtp_credentials.create', 'Create SMTP credentials'),
('smtp_credentials.revoke', 'smtp_credentials.revoke', 'Revoke SMTP credentials'),
('ip.read', 'ip.read', 'View IP addresses and reputation'),
('ip.write', 'ip.write', 'Manage IP assignments'),
('pools.read', 'pools.read', 'View IP pools'),
('pools.write', 'pools.write', 'Manage IP pools'),
('messages.read', 'messages.read', 'View message logs and delivery status'),
('messages.body.read', 'messages.body.read', 'View full message bodies and raw MIME'),
('messages.retry', 'messages.retry', 'Retry failed messages'),
('messages.export', 'messages.export', 'Export message logs'),
('messages.cancel', 'messages.cancel', 'Cancel queued messages'),
('suppressions.read', 'suppressions.read', 'View suppression list'),
('suppressions.write', 'suppressions.write', 'Add or remove suppressions'),
('suppressions.import', 'suppressions.import', 'Import suppression lists'),
('templates.read', 'templates.read', 'View email templates'),
('templates.write', 'templates.write', 'Create and modify email templates'),
('templates.publish', 'templates.publish', 'Publish template versions'),
('webhooks.read', 'webhooks.read', 'View webhooks and deliveries'),
('webhooks.write', 'webhooks.write', 'Configure webhook endpoints'),
('webhooks.replay', 'webhooks.replay', 'Replay webhook events'),
('webhooks.secret.rotate', 'webhooks.secret.rotate', 'Rotate webhook secrets'),
('inbound.read', 'inbound.read', 'View inbound routes and messages'),
('inbound.write', 'inbound.write', 'Manage inbound routes'),
('analytics.read', 'analytics.read', 'View delivery statistics and charts'),
('reports.read', 'reports.read', 'View DMARC, TLS-RPT and FBL reports'),
('audit.read', 'audit.read', 'View system and tenant audit logs'),
('settings.read', 'settings.read', 'View tenant settings'),
('settings.write', 'settings.write', 'Update tenant settings'),
('retention.write', 'retention.write', 'Configure retention policies'),
('privacy.erase', 'privacy.erase', 'Execute GDPR/DPDP recipient erasure'),
('privacy.export', 'privacy.export', 'Export recipient data'),
('impersonate.read', 'impersonate.read', 'Enter tenant in read-only impersonation mode'),
('impersonate.write', 'impersonate.write', 'Enter tenant in read-write impersonation mode'),
('system.read', 'system.read', 'View system health and node status'),
('system.write', 'system.write', 'Manage system nodes and workers');

-- Seed built-in roles
INSERT IGNORE INTO roles (id, organization_id, name, scope, description, is_system) VALUES
('role_platform_owner', NULL, 'Platform Owner', 'platform', 'Full platform access including other owners', TRUE),
('role_platform_admin', NULL, 'Platform Admin', 'platform', 'Full platform administrative access', TRUE),
('role_platform_support', NULL, 'Platform Support', 'platform', 'Read-only across tenants with masked bodies', TRUE),
('role_tenant_owner', NULL, 'Tenant Owner', 'tenant', 'Full ownership of tenant resources', TRUE),
('role_tenant_admin', NULL, 'Tenant Admin', 'tenant', 'Manage tenant domains, keys, webhooks, templates', TRUE),
('role_developer', NULL, 'Developer', 'tenant', 'Manage API keys, webhooks, templates, view logs', TRUE),
('role_analyst', NULL, 'Analyst', 'tenant', 'Read-only access to messages, stats, and domains', TRUE),
('role_support', NULL, 'Support', 'tenant', 'Read message metadata with bodies and secrets masked', TRUE);
