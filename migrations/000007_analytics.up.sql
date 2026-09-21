-- Mailvex Phase 7: Analytics & Reporting Engine (Pre-aggregated Rollups, Latency Histograms, Exports)

CREATE TABLE IF NOT EXISTS analytics_rollups (
    id VARCHAR(64) PRIMARY KEY,
    tenant_id VARCHAR(36) NOT NULL,
    time_bucket VARCHAR(8) NOT NULL, -- 1m, 1h, 1d, 1mo
    bucket_start DATETIME(0) NOT NULL,
    stream_id VARCHAR(36) NULL,
    domain_id VARCHAR(36) NULL,
    provider VARCHAR(64) NOT NULL DEFAULT 'other', -- google, microsoft, yahoo, apple, other
    campaign_tag VARCHAR(64) NOT NULL DEFAULT 'default',
    
    -- Event Counters
    sent_count BIGINT NOT NULL DEFAULT 0,
    delivered_count BIGINT NOT NULL DEFAULT 0,
    bounced_hard_count BIGINT NOT NULL DEFAULT 0,
    bounced_soft_count BIGINT NOT NULL DEFAULT 0,
    deferred_count BIGINT NOT NULL DEFAULT 0,
    complained_count BIGINT NOT NULL DEFAULT 0,
    opened_unique_count BIGINT NOT NULL DEFAULT 0,
    opened_total_count BIGINT NOT NULL DEFAULT 0,
    clicked_unique_count BIGINT NOT NULL DEFAULT 0,
    clicked_total_count BIGINT NOT NULL DEFAULT 0,
    unsubscribed_count BIGINT NOT NULL DEFAULT 0,
    
    -- Latency Histogram Distribution
    latency_under_5s BIGINT NOT NULL DEFAULT 0,
    latency_under_30s BIGINT NOT NULL DEFAULT 0,
    latency_under_1m BIGINT NOT NULL DEFAULT 0,
    latency_under_5m BIGINT NOT NULL DEFAULT 0,
    latency_under_15m BIGINT NOT NULL DEFAULT 0,
    latency_over_15m BIGINT NOT NULL DEFAULT 0,

    CONSTRAINT fk_ar_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
    INDEX idx_ar_tenant_time (tenant_id, time_bucket, bucket_start),
    INDEX idx_ar_tenant_stream (tenant_id, stream_id, bucket_start),
    INDEX idx_ar_tenant_domain (tenant_id, domain_id, bucket_start),
    INDEX idx_ar_tenant_provider (tenant_id, provider, bucket_start),
    INDEX idx_ar_tenant_tag (tenant_id, campaign_tag, bucket_start),
    UNIQUE KEY uq_ar_rollup (tenant_id, time_bucket, bucket_start, stream_id, domain_id, provider, campaign_tag)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS analytics_exports (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id VARCHAR(36) NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'pending', -- pending, completed, failed
    format VARCHAR(16) NOT NULL DEFAULT 'csv',
    filter_criteria JSON NULL,
    file_url VARCHAR(255) NULL,
    file_data MEDIUMBLOB NULL,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    completed_at DATETIME(6) NULL,
    CONSTRAINT fk_ae_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
    INDEX idx_ae_tenant (tenant_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
