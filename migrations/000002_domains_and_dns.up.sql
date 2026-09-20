-- Mailvex Phase 2: Domains, DNS Records, DKIM Keys, and Message Streams

CREATE TABLE IF NOT EXISTS domains (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id VARCHAR(36) NOT NULL,
    name VARCHAR(255) NOT NULL,
    type VARCHAR(32) NOT NULL DEFAULT 'public', -- 'public' or 'local'
    status VARCHAR(32) NOT NULL DEFAULT 'pending', -- 'pending', 'verifying', 'verified', 'degraded', 'suspended', 'disabled'
    ownership_token VARCHAR(64) NOT NULL,
    degraded_at DATETIME(6) NULL,
    verified_at DATETIME(6) NULL,
    last_checked_at DATETIME(6) NULL,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    CONSTRAINT fk_domains_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
    UNIQUE KEY uk_domains_name_verified (name, status),
    INDEX idx_domains_tenant (tenant_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS domain_delegations (
    id VARCHAR(36) PRIMARY KEY,
    parent_domain_id VARCHAR(36) NOT NULL,
    delegated_to_tenant_id VARCHAR(36) NOT NULL,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    CONSTRAINT fk_dd_parent FOREIGN KEY (parent_domain_id) REFERENCES domains(id) ON DELETE CASCADE,
    CONSTRAINT fk_dd_tenant FOREIGN KEY (delegated_to_tenant_id) REFERENCES tenants(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS dkim_keys (
    id VARCHAR(36) PRIMARY KEY,
    domain_id VARCHAR(36) NOT NULL,
    tenant_id VARCHAR(36) NOT NULL,
    selector VARCHAR(64) NOT NULL DEFAULT 'mvx',
    algorithm VARCHAR(32) NOT NULL DEFAULT 'rsa2048',
    encrypted_private_key TEXT NOT NULL,
    public_key TEXT NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'pending', -- 'pending', 'active', 'retiring', 'retired'
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    activated_at DATETIME(6) NULL,
    retired_at DATETIME(6) NULL,
    CONSTRAINT fk_dkim_domain FOREIGN KEY (domain_id) REFERENCES domains(id) ON DELETE CASCADE,
    CONSTRAINT fk_dkim_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
    INDEX idx_dkim_domain (domain_id, status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS domain_dns_records (
    id VARCHAR(36) PRIMARY KEY,
    domain_id VARCHAR(36) NOT NULL,
    record_type VARCHAR(16) NOT NULL, -- TXT, CNAME, MX
    name VARCHAR(255) NOT NULL,
    expected_value TEXT NOT NULL,
    observed_value TEXT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'pending', -- 'pending', 'pass', 'warning', 'fail'
    error_message TEXT NULL,
    last_checked_at DATETIME(6) NULL,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    CONSTRAINT fk_dns_domain FOREIGN KEY (domain_id) REFERENCES domains(id) ON DELETE CASCADE,
    INDEX idx_dns_domain (domain_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS message_streams (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id VARCHAR(36) NOT NULL,
    name VARCHAR(64) NOT NULL,
    type VARCHAR(32) NOT NULL DEFAULT 'transactional', -- 'transactional' or 'broadcast'
    is_default BOOLEAN NOT NULL DEFAULT FALSE,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    CONSTRAINT fk_streams_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
    UNIQUE KEY uk_tenant_stream_name (tenant_id, name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
