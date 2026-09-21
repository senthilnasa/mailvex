-- 000012_sso_impersonation_compliance.up.sql
-- Mailvex Phase 12: SSO (OIDC/SAML), Platform Admin Impersonation, and Deliverability Compliance

-- SSO Providers (OIDC / SAML 2.0 SP)
CREATE TABLE IF NOT EXISTS sso_providers (
    id VARCHAR(32) PRIMARY KEY,
    org_id VARCHAR(32) NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    tenant_id VARCHAR(32) REFERENCES tenants(id) ON DELETE SET NULL,
    name VARCHAR(255) NOT NULL,
    provider_type VARCHAR(32) NOT NULL, -- 'oidc', 'saml'
    issuer_url VARCHAR(1024),
    client_id VARCHAR(255),
    client_secret_enc TEXT,
    idp_metadata_url VARCHAR(1024),
    idp_metadata_xml TEXT,
    sp_entity_id VARCHAR(512),
    sp_acs_url VARCHAR(1024),
    allowed_domains JSONB NOT NULL DEFAULT '[]'::jsonb,
    default_role VARCHAR(64) NOT NULL DEFAULT 'viewer',
    attribute_mapping JSONB NOT NULL DEFAULT '{}'::jsonb,
    enforce_sso BOOLEAN NOT NULL DEFAULT false,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_sso_providers_org ON sso_providers(org_id);
CREATE INDEX IF NOT EXISTS idx_sso_providers_tenant ON sso_providers(tenant_id);

-- Impersonation Sessions (Audit log of platform admin 'view as tenant')
CREATE TABLE IF NOT EXISTS impersonation_sessions (
    id VARCHAR(32) PRIMARY KEY,
    actor_user_id VARCHAR(32) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    target_tenant_id VARCHAR(32) NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    reason TEXT NOT NULL,
    access_type VARCHAR(32) NOT NULL DEFAULT 'read_only', -- 'read_only', 'read_write'
    is_support_masking BOOLEAN NOT NULL DEFAULT false,
    ip_address VARCHAR(45) NOT NULL,
    user_agent TEXT NOT NULL,
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ended_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_impersonation_actor ON impersonation_sessions(actor_user_id);
CREATE INDEX IF NOT EXISTS idx_impersonation_target ON impersonation_sessions(target_tenant_id);
CREATE INDEX IF NOT EXISTS idx_impersonation_active ON impersonation_sessions(ended_at) WHERE ended_at IS NULL;

-- Deliverability Compliance Checks
CREATE TABLE IF NOT EXISTS compliance_checks (
    id VARCHAR(32) PRIMARY KEY,
    tenant_id VARCHAR(32) NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    domain_id VARCHAR(32) REFERENCES domains(id) ON DELETE CASCADE,
    domain_name VARCHAR(255) NOT NULL,
    overall_score INT NOT NULL, -- 0 to 100
    spf_status VARCHAR(32) NOT NULL, -- 'pass', 'warn', 'fail'
    spf_details TEXT,
    dkim_status VARCHAR(32) NOT NULL,
    dkim_details TEXT,
    dmarc_status VARCHAR(32) NOT NULL,
    dmarc_details TEXT,
    fcrdns_status VARCHAR(32) NOT NULL,
    fcrdns_details TEXT,
    tls_status VARCHAR(32) NOT NULL,
    tls_details TEXT,
    dnsbl_status VARCHAR(32) NOT NULL,
    dnsbl_details TEXT,
    unsubscribe_status VARCHAR(32) NOT NULL,
    unsubscribe_details TEXT,
    checked_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_compliance_checks_tenant ON compliance_checks(tenant_id);
CREATE INDEX IF NOT EXISTS idx_compliance_checks_domain ON compliance_checks(domain_name);
