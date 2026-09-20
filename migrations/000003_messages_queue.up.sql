-- Mailvex Phase 3: Messages, Recipients, Events, Outbox, SMTP Credentials, LAN Devices

CREATE TABLE IF NOT EXISTS smtp_credentials (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id VARCHAR(36) NOT NULL,
    username VARCHAR(64) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    allowed_domains JSON NULL,
    allowed_cidrs JSON NULL,
    stream_id VARCHAR(36) NULL,
    rate_limit INT NOT NULL DEFAULT 100,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    revoked_at DATETIME(6) NULL,
    CONSTRAINT fk_smtp_cred_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
    INDEX idx_smtp_cred_tenant (tenant_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS trusted_networks (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id VARCHAR(36) NOT NULL,
    cidr VARCHAR(64) NOT NULL,
    description VARCHAR(255) NOT NULL DEFAULT '',
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    CONSTRAINT fk_tn_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
    INDEX idx_tn_tenant (tenant_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS devices (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id VARCHAR(36) NOT NULL,
    name VARCHAR(128) NOT NULL,
    type VARCHAR(64) NOT NULL DEFAULT 'generic',
    expected_ip VARCHAR(45) NOT NULL,
    smtp_credential_id VARCHAR(36) NULL,
    allowed_from JSON NULL,
    last_seen_at DATETIME(6) NULL,
    message_count BIGINT NOT NULL DEFAULT 0,
    last_error TEXT NULL,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    CONSTRAINT fk_devices_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
    INDEX idx_devices_tenant (tenant_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS outbox (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id VARCHAR(36) NOT NULL,
    stream VARCHAR(64) NOT NULL,
    payload JSON NOT NULL,
    available_at DATETIME(6) NOT NULL,
    claimed_by VARCHAR(64) NULL,
    claimed_at DATETIME(6) NULL,
    published_at DATETIME(6) NULL,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    INDEX idx_outbox_pending (published_at, available_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS messages (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id VARCHAR(36) NOT NULL,
    stream_id VARCHAR(36) NOT NULL,
    domain_id VARCHAR(36) NOT NULL,
    source VARCHAR(32) NOT NULL DEFAULT 'api', -- api, smtp, system, inbound_reply
    api_key_id VARCHAR(36) NULL,
    smtp_credential_id VARCHAR(36) NULL,
    template_version_id VARCHAR(36) NULL,
    from_address VARCHAR(255) NOT NULL,
    from_name VARCHAR(255) NOT NULL DEFAULT '',
    reply_to VARCHAR(255) NULL,
    subject VARCHAR(500) NOT NULL DEFAULT '',
    message_id_header VARCHAR(255) NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'queued', -- queued, sending, partially_delivered, delivered, failed, cancelled
    priority INT NOT NULL DEFAULT 1,
    environment VARCHAR(32) NOT NULL DEFAULT 'live',
    size_bytes INT NOT NULL DEFAULT 0,
    body_ref VARCHAR(255) NULL,
    raw_mime_ref VARCHAR(255) NULL,
    tags JSON NULL,
    metadata JSON NULL,
    scheduled_at DATETIME(6) NULL,
    queued_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    sent_at DATETIME(6) NULL,
    completed_at DATETIME(6) NULL,
    expires_at DATETIME(6) NULL,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    INDEX idx_msg_tenant_created (tenant_id, created_at),
    INDEX idx_msg_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS message_recipients (
    id VARCHAR(36) PRIMARY KEY,
    message_id VARCHAR(36) NOT NULL,
    tenant_id VARCHAR(36) NOT NULL,
    address VARCHAR(255) NOT NULL,
    address_domain VARCHAR(255) NOT NULL,
    mx_group VARCHAR(64) NOT NULL DEFAULT 'custom',
    type VARCHAR(16) NOT NULL DEFAULT 'to', -- to, cc, bcc
    status VARCHAR(32) NOT NULL DEFAULT 'queued', -- queued, sending, delivered, deferred, bounced, suppressed, failed
    attempts INT NOT NULL DEFAULT 0,
    next_attempt_at DATETIME(6) NULL,
    last_smtp_code VARCHAR(16) NULL,
    last_response TEXT NULL,
    delivered_at DATETIME(6) NULL,
    bounced_at DATETIME(6) NULL,
    suppressed_reason VARCHAR(64) NULL,
    verp_token VARCHAR(64) NULL,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    INDEX idx_rcpt_msg (message_id),
    INDEX idx_rcpt_tenant_addr (tenant_id, address),
    INDEX idx_rcpt_status (status, next_attempt_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS message_events (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id VARCHAR(36) NOT NULL,
    message_id VARCHAR(36) NOT NULL,
    recipient_id VARCHAR(36) NULL,
    type VARCHAR(64) NOT NULL,
    data JSON NULL,
    occurred_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    INDEX idx_evt_msg (message_id),
    INDEX idx_evt_tenant_time (tenant_id, occurred_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS delivery_attempts (
    id VARCHAR(36) PRIMARY KEY,
    recipient_id VARCHAR(36) NOT NULL,
    tenant_id VARCHAR(36) NOT NULL,
    attempt_no INT NOT NULL DEFAULT 1,
    ip_id VARCHAR(36) NULL,
    node_id VARCHAR(36) NULL,
    remote_mx VARCHAR(255) NULL,
    remote_ip VARCHAR(45) NULL,
    tls_version VARCHAR(32) NULL,
    tls_cipher VARCHAR(64) NULL,
    tls_policy_applied VARCHAR(32) NULL,
    smtp_code VARCHAR(16) NULL,
    enhanced_code VARCHAR(32) NULL,
    smtp_response TEXT NULL,
    duration_ms INT NOT NULL DEFAULT 0,
    started_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    outcome VARCHAR(32) NOT NULL, -- delivered, deferred, bounced, failed
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    INDEX idx_att_rcpt (recipient_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
