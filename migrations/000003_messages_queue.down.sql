-- Mailvex Phase 3: Rollback

DROP TABLE IF EXISTS delivery_attempts;
DROP TABLE IF EXISTS message_events;
DROP TABLE IF EXISTS message_recipients;
DROP TABLE IF EXISTS messages;
DROP TABLE IF EXISTS outbox;
DROP TABLE IF EXISTS devices;
DROP TABLE IF EXISTS trusted_networks;
DROP TABLE IF EXISTS smtp_credentials;
