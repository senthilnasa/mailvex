-- 000012_sso_impersonation_compliance.down.sql
-- Mailvex Phase 12 Rollback

DROP TABLE IF EXISTS compliance_checks CASCADE;
DROP TABLE IF EXISTS impersonation_sessions CASCADE;
DROP TABLE IF EXISTS sso_providers CASCADE;
