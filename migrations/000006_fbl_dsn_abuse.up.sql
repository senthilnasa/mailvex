-- Mailvex Phase 6: Feedback Loops, Inbound DSN, Abuse Reports, Provider Snapshots

CREATE TABLE IF NOT EXISTS fbl_reports (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id VARCHAR(36) NOT NULL,
    message_id VARCHAR(36) NULL,
    recipient_id VARCHAR(36) NULL,
    feedback_type VARCHAR(64) NOT NULL DEFAULT 'abuse',
    user_agent VARCHAR(255) NULL,
    reported_domain VARCHAR(255) NULL,
    original_headers MEDIUMTEXT NULL,
    arrival_date DATETIME(6) NULL,
    raw_mime_ref VARCHAR(255) NULL,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    CONSTRAINT fk_fbl_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
    INDEX idx_fbl_tenant (tenant_id),
    INDEX idx_fbl_msg (message_id),
    INDEX idx_fbl_rcpt (recipient_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS unmatched_bounces (
    id VARCHAR(36) PRIMARY KEY,
    sender VARCHAR(255) NULL,
    recipient VARCHAR(255) NULL,
    subject VARCHAR(255) NULL,
    raw_mime_ref VARCHAR(255) NULL,
    parse_error TEXT NULL,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS abuse_reports (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id VARCHAR(36) NULL,
    domain_id VARCHAR(36) NULL,
    message_id VARCHAR(36) NULL,
    sender VARCHAR(255) NOT NULL,
    recipient VARCHAR(255) NOT NULL,
    subject VARCHAR(255) NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'pending', -- pending, resolved, escalated
    notes TEXT NULL,
    raw_mime_ref VARCHAR(255) NULL,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    resolved_at DATETIME(6) NULL,
    INDEX idx_abuse_status (status),
    INDEX idx_abuse_tenant (tenant_id),
    INDEX idx_abuse_domain (domain_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS ip_reputation_snapshots (
    id VARCHAR(36) PRIMARY KEY,
    ip_id VARCHAR(36) NOT NULL,
    source VARCHAR(32) NOT NULL, -- snds, postmaster, dnsbl
    sample_date DATE NOT NULL,
    score DOUBLE NOT NULL DEFAULT 100.0,
    sample_data JSON NULL,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    INDEX idx_ip_rep_sample (ip_id, sample_date)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS postmaster_snapshots (
    id VARCHAR(36) PRIMARY KEY,
    domain_id VARCHAR(36) NOT NULL,
    sample_date DATE NOT NULL,
    spam_rate DOUBLE NOT NULL DEFAULT 0.0,
    domain_reputation VARCHAR(32) NOT NULL DEFAULT 'high', -- bad, low, medium, high
    ip_reputation VARCHAR(32) NOT NULL DEFAULT 'high',
    dkim_success_rate DOUBLE NOT NULL DEFAULT 1.0,
    spf_success_rate DOUBLE NOT NULL DEFAULT 1.0,
    dmarc_success_rate DOUBLE NOT NULL DEFAULT 1.0,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    INDEX idx_pm_domain_date (domain_id, sample_date)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
