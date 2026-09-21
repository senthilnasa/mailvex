-- Phase 10: Open/Click Tracking and Inbound Email Routing

CREATE TABLE IF NOT EXISTS tracking_links (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id VARCHAR(36) NOT NULL,
    message_id VARCHAR(36) NOT NULL,
    link_id VARCHAR(64) NOT NULL,
    target_url TEXT NOT NULL,
    created_at DATETIME(6) NOT NULL,
    INDEX idx_tenant_link (tenant_id, link_id),
    INDEX idx_message_id (message_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS tracking_events (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id VARCHAR(36) NOT NULL,
    message_id VARCHAR(36) NOT NULL,
    recipient_id VARCHAR(36) NOT NULL,
    type VARCHAR(32) NOT NULL, -- open, click
    target_url TEXT NULL,
    ip VARCHAR(64) NOT NULL,
    user_agent TEXT NULL,
    is_bot BOOLEAN NOT NULL DEFAULT FALSE,
    bot_reason VARCHAR(64) NULL,
    occurred_at DATETIME(6) NOT NULL,
    INDEX idx_tenant_msg (tenant_id, message_id),
    INDEX idx_recipient (recipient_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS inbound_routes (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id VARCHAR(36) NOT NULL,
    priority INT NOT NULL DEFAULT 10,
    match_type VARCHAR(32) NOT NULL, -- exact, wildcard, catch_all
    pattern VARCHAR(255) NOT NULL,
    action VARCHAR(32) NOT NULL, -- webhook, store_only, forward, discard
    action_target TEXT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at DATETIME(6) NOT NULL,
    updated_at DATETIME(6) NOT NULL,
    INDEX idx_tenant_priority (tenant_id, priority)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS inbound_messages (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id VARCHAR(36) NOT NULL,
    route_id VARCHAR(36) NULL,
    message_id VARCHAR(255) NULL,
    from_address VARCHAR(255) NOT NULL,
    to_address VARCHAR(255) NOT NULL,
    subject TEXT NULL,
    size_bytes INT NOT NULL DEFAULT 0,
    raw_mime_ref TEXT NULL,
    spf_status VARCHAR(32) NOT NULL DEFAULT 'none',
    dkim_status VARCHAR(32) NOT NULL DEFAULT 'none',
    dmarc_status VARCHAR(32) NOT NULL DEFAULT 'none',
    spam_score FLOAT NOT NULL DEFAULT 0.0,
    status VARCHAR(32) NOT NULL DEFAULT 'processed',
    created_at DATETIME(6) NOT NULL,
    INDEX idx_tenant_status (tenant_id, status),
    INDEX idx_to_address (to_address)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
