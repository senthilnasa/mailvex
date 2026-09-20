-- 000004_api_keys_and_templates.up.sql

CREATE TABLE IF NOT EXISTS api_keys (
    id VARCHAR(32) PRIMARY KEY,
    tenant_id VARCHAR(32) NOT NULL,
    name VARCHAR(255) NOT NULL,
    prefix VARCHAR(16) NOT NULL,
    key_hash VARCHAR(64) NOT NULL UNIQUE,
    last_four VARCHAR(4) NOT NULL,
    environment VARCHAR(16) NOT NULL DEFAULT 'live',
    scopes JSON NOT NULL,
    allowed_domains JSON,
    allowed_cidrs JSON,
    stream_id VARCHAR(32),
    rate_limit INT NOT NULL DEFAULT 100,
    expires_at TIMESTAMP NULL,
    last_used_at TIMESTAMP NULL,
    revoked_at TIMESTAMP NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_api_keys_tenant (tenant_id),
    INDEX idx_api_keys_hash (key_hash)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS templates (
    id VARCHAR(32) PRIMARY KEY,
    tenant_id VARCHAR(32) NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_templates_tenant (tenant_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS template_versions (
    id VARCHAR(32) PRIMARY KEY,
    template_id VARCHAR(32) NOT NULL,
    tenant_id VARCHAR(32) NOT NULL,
    version INT NOT NULL,
    subject VARCHAR(998) NOT NULL,
    html MEDIUMTEXT,
    text MEDIUMTEXT,
    variables JSON,
    is_published BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY uq_template_version (tenant_id, template_id, version),
    INDEX idx_tmpl_ver_lookup (tenant_id, template_id, is_published)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS idempotency_records (
    id VARCHAR(64) PRIMARY KEY, -- tenant_id:api_key_id:key
    tenant_id VARCHAR(32) NOT NULL,
    request_hash VARCHAR(64) NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'in_progress',
    status_code INT NOT NULL DEFAULT 0,
    response_body MEDIUMBLOB,
    response_headers JSON,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP NOT NULL,
    INDEX idx_idempotency_tenant (tenant_id),
    INDEX idx_idempotency_expires (expires_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
