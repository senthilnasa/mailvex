-- Phase 11: Reputation, DNSBL, Warmup Engine, Alert Rules & Multi-Channel Notifications

CREATE TABLE IF NOT EXISTS dnsbl_providers (
    id VARCHAR(26) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    query_pattern VARCHAR(255) NOT NULL,
    return_code_meanings TEXT,
    delist_url VARCHAR(512),
    check_interval_seconds INT NOT NULL DEFAULT 3600,
    is_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS dnsbl_results (
    id VARCHAR(26) PRIMARY KEY,
    ip_address_id VARCHAR(26) NOT NULL,
    ip_address VARCHAR(45) NOT NULL,
    provider_id VARCHAR(26) NOT NULL,
    is_listed BOOLEAN NOT NULL DEFAULT FALSE,
    response_code VARCHAR(50),
    return_meaning TEXT,
    delist_url VARCHAR(512),
    checked_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_dnsbl_results_ip (ip_address_id),
    INDEX idx_dnsbl_results_checked (checked_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS warmup_plans (
    id VARCHAR(26) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    total_days INT NOT NULL DEFAULT 30,
    is_default BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS warmup_plan_days (
    id VARCHAR(26) PRIMARY KEY,
    plan_id VARCHAR(26) NOT NULL,
    day_number INT NOT NULL,
    max_volume INT NOT NULL,
    provider_caps TEXT,
    INDEX idx_warmup_plan_day (plan_id, day_number)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS ip_warmup_states (
    id VARCHAR(26) PRIMARY KEY,
    ip_address_id VARCHAR(26) NOT NULL UNIQUE,
    ip_address VARCHAR(45) NOT NULL,
    plan_id VARCHAR(26) NOT NULL,
    current_day INT NOT NULL DEFAULT 1,
    current_day_volume INT NOT NULL DEFAULT 0,
    provider_volume TEXT,
    status VARCHAR(50) NOT NULL DEFAULT 'warming',
    held_reason TEXT,
    started_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_advanced_at TIMESTAMP NULL,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_ip_warmup_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS alert_rules (
    id VARCHAR(26) PRIMARY KEY,
    tenant_id VARCHAR(26),
    name VARCHAR(255) NOT NULL,
    metric_type VARCHAR(50) NOT NULL,
    comparison_operator VARCHAR(10) NOT NULL DEFAULT '>',
    threshold DOUBLE NOT NULL,
    duration_minutes INT NOT NULL DEFAULT 5,
    cooldown_minutes INT NOT NULL DEFAULT 60,
    severity VARCHAR(20) NOT NULL DEFAULT 'warning',
    is_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_alert_rules_tenant (tenant_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS alert_incidents (
    id VARCHAR(26) PRIMARY KEY,
    rule_id VARCHAR(26) NOT NULL,
    tenant_id VARCHAR(26),
    metric_type VARCHAR(50) NOT NULL,
    severity VARCHAR(20) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'firing',
    triggered_value DOUBLE NOT NULL,
    threshold DOUBLE NOT NULL,
    summary TEXT NOT NULL,
    fired_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    acknowledged_at TIMESTAMP NULL,
    resolved_at TIMESTAMP NULL,
    INDEX idx_incidents_rule (rule_id),
    INDEX idx_incidents_status (status),
    INDEX idx_incidents_tenant (tenant_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS notification_channels (
    id VARCHAR(26) PRIMARY KEY,
    tenant_id VARCHAR(26),
    name VARCHAR(255) NOT NULL,
    type VARCHAR(50) NOT NULL,
    config TEXT NOT NULL,
    is_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_notif_channels_tenant (tenant_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS alert_rule_channels (
    rule_id VARCHAR(26) NOT NULL,
    channel_id VARCHAR(26) NOT NULL,
    PRIMARY KEY (rule_id, channel_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS alert_notifications (
    id VARCHAR(26) PRIMARY KEY,
    tenant_id VARCHAR(26),
    incident_id VARCHAR(26) NOT NULL,
    title VARCHAR(255) NOT NULL,
    message TEXT NOT NULL,
    severity VARCHAR(20) NOT NULL,
    is_read BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_alert_notifications_tenant (tenant_id, is_read)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
