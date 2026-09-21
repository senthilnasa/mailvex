-- Mailvex Phase 8: Webhooks Delivery Engine (Endpoints, Secret Overlap, Delivery Logs)

CREATE TABLE IF NOT EXISTS webhook_endpoints (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id VARCHAR(36) NOT NULL,
    url VARCHAR(512) NOT NULL,
    description VARCHAR(255) NULL,
    events JSON NOT NULL, -- array of subscribed events or ["*"]
    secret VARCHAR(128) NOT NULL, -- whsec_...
    previous_secret VARCHAR(128) NULL,
    previous_secret_expires_at DATETIME(6) NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'active', -- active, disabled_failing, disabled_manual
    consecutive_failures INT NOT NULL DEFAULT 0,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),

    CONSTRAINT fk_we_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
    INDEX idx_we_tenant_status (tenant_id, status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS webhook_deliveries (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id VARCHAR(36) NOT NULL,
    endpoint_id VARCHAR(36) NOT NULL,
    event_id VARCHAR(64) NOT NULL,
    event_type VARCHAR(64) NOT NULL,
    payload MEDIUMTEXT NOT NULL,
    status_code INT NULL,
    response_body TEXT NULL,
    error_message TEXT NULL,
    attempt INT NOT NULL DEFAULT 1,
    duration_ms INT NOT NULL DEFAULT 0,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),

    CONSTRAINT fk_wd_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
    CONSTRAINT fk_wd_endpoint FOREIGN KEY (endpoint_id) REFERENCES webhook_endpoints(id) ON DELETE CASCADE,
    INDEX idx_wd_tenant_endpoint (tenant_id, endpoint_id, created_at),
    INDEX idx_wd_event (event_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
