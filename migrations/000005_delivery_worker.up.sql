-- Mailvex Phase 5: Outbound Delivery Engine, Suppressions, IP Pools, Relays, Bounce Rules, DLQ

CREATE TABLE IF NOT EXISTS suppressions (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id VARCHAR(36) NOT NULL,
    stream_id VARCHAR(36) NULL,
    address VARCHAR(255) NOT NULL,
    address_domain VARCHAR(255) NOT NULL,
    type VARCHAR(32) NOT NULL, -- hard_bounce, complaint, unsubscribe, manual, domain_block, role_account
    reason TEXT NULL,
    source VARCHAR(32) NOT NULL, -- smtp, dsn, fbl, one_click, api, import, admin
    source_ref VARCHAR(255) NULL,
    expires_at DATETIME(6) NULL,
    created_by VARCHAR(64) NULL,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    CONSTRAINT fk_supp_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
    INDEX idx_supp_tenant_addr (tenant_id, address),
    INDEX idx_supp_tenant_domain (tenant_id, address_domain),
    INDEX idx_supp_tenant_stream_addr (tenant_id, stream_id, address)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS ip_pools (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id VARCHAR(36) NULL, -- NULL for shared pools, or dedicated_tenant_id
    name VARCHAR(64) NOT NULL,
    strategy VARCHAR(32) NOT NULL DEFAULT 'round_robin', -- fixed, round_robin, least_used, weighted
    is_default BOOLEAN NOT NULL DEFAULT FALSE,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    INDEX idx_pool_tenant (tenant_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS ip_addresses (
    id VARCHAR(36) PRIMARY KEY,
    pool_id VARCHAR(36) NOT NULL,
    ip VARCHAR(45) NOT NULL UNIQUE,
    subnet VARCHAR(45) NULL,
    ehlo_hostname VARCHAR(255) NOT NULL,
    ptr_record VARCHAR(255) NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'active', -- active, warming, quarantine, disabled
    warmup_day INT NOT NULL DEFAULT 1,
    max_daily_volume INT NOT NULL DEFAULT 0, -- 0 = unlimited
    sent_today INT NOT NULL DEFAULT 0,
    node_id VARCHAR(64) NOT NULL DEFAULT 'node-1',
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    CONSTRAINT fk_ip_pool FOREIGN KEY (pool_id) REFERENCES ip_pools(id) ON DELETE CASCADE,
    INDEX idx_ip_pool (pool_id),
    INDEX idx_ip_node (node_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS relays (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id VARCHAR(36) NULL,
    name VARCHAR(128) NOT NULL,
    host VARCHAR(255) NOT NULL,
    port INT NOT NULL DEFAULT 587,
    tls_mode VARCHAR(32) NOT NULL DEFAULT 'starttls', -- starttls, tls, none
    auth_user VARCHAR(255) NULL,
    auth_pass_encrypted TEXT NULL,
    from_rewrite_policy VARCHAR(32) NOT NULL DEFAULT 'preserve', -- preserve, replace
    from_rewrite_address VARCHAR(255) NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    INDEX idx_relay_tenant (tenant_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS bounce_rules (
    id VARCHAR(36) PRIMARY KEY,
    pattern VARCHAR(255) NOT NULL,
    enhanced_code VARCHAR(32) NULL,
    provider VARCHAR(64) NULL,
    classification VARCHAR(32) NOT NULL, -- hard_bounce, soft_bounce, policy_rejection, spam_rejection, rate_limited, auth_rejection, unknown
    action VARCHAR(32) NOT NULL, -- suppress_immediate, retry, dead_letter, alert
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS provider_mx_rules (
    id VARCHAR(36) PRIMARY KEY,
    pattern VARCHAR(255) NOT NULL,
    provider VARCHAR(64) NOT NULL,
    max_connections INT NOT NULL DEFAULT 5,
    messages_per_connection INT NOT NULL DEFAULT 100,
    rate_limit_per_min INT NOT NULL DEFAULT 600,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS dead_letter_messages (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id VARCHAR(36) NOT NULL,
    message_id VARCHAR(36) NOT NULL,
    recipient_id VARCHAR(36) NOT NULL,
    destination VARCHAR(255) NOT NULL,
    reason VARCHAR(64) NOT NULL,
    last_smtp_code VARCHAR(16) NULL,
    last_enhanced_code VARCHAR(32) NULL,
    last_response TEXT NULL,
    attempts INT NOT NULL DEFAULT 0,
    source_ip VARCHAR(45) NULL,
    node_id VARCHAR(64) NULL,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    INDEX idx_dlq_tenant (tenant_id),
    INDEX idx_dlq_msg (message_id),
    INDEX idx_dlq_rcpt (recipient_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Seeds for IP pools
INSERT IGNORE INTO ip_pools (id, tenant_id, name, strategy, is_default, created_at)
VALUES ('pool_default', NULL, 'default', 'round_robin', TRUE, NOW(6));

-- Seeds for Provider MX Rules
INSERT IGNORE INTO provider_mx_rules (id, pattern, provider, max_connections, messages_per_connection, rate_limit_per_min, created_at)
VALUES
('pmx_google', '*.google.com', 'google', 5, 100, 600, NOW(6)),
('pmx_googlemail', '*.googlemail.com', 'google', 5, 100, 600, NOW(6)),
('pmx_microsoft', '*.mail.protection.outlook.com', 'microsoft', 5, 100, 600, NOW(6)),
('pmx_yahoo', '*.yahoodns.net', 'yahoo', 4, 50, 300, NOW(6)),
('pmx_apple', '*.icloud.com', 'apple', 3, 50, 300, NOW(6));

-- Seeds for Bounce Rules
INSERT IGNORE INTO bounce_rules (id, pattern, enhanced_code, provider, classification, action, created_at)
VALUES
('br_user_unknown', 'user unknown|mailbox not found|no such user|recipient address rejected', '5.1.1', NULL, 'hard_bounce', 'suppress_immediate', NOW(6)),
('br_bad_domain', 'domain not found|host or domain name not found|unrouteable address', '5.1.2', NULL, 'hard_bounce', 'suppress_immediate', NOW(6)),
('br_mailbox_full', 'mailbox full|quota exceeded|over quota', '4.2.2', NULL, 'soft_bounce', 'retry', NOW(6)),
('br_rate_limited', 'rate limit|too many connections|too many messages|throttled', '4.7.28', NULL, 'rate_limited', 'retry', NOW(6)),
('br_spam_block', 'blocked using spamhaus|blocked by spam|spam detected|reputation', '5.7.1', NULL, 'spam_rejection', 'dead_letter', NOW(6)),
('br_auth_failure', 'dkim check failed|spf check failed|dmarc policy', '5.7.26', NULL, 'auth_rejection', 'dead_letter', NOW(6));
