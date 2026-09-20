# Mailvex — Webhook Architecture & Dispatcher

## 1. Webhook Dispatch Flow

Mailvex delivers real-time event notifications to tenant-configured HTTPS endpoints.

### 1.1 Signature & Replay Protection
Every outbound webhook delivery includes:
- `X-Mailvex-Delivery-Id`: Unique delivery attempt ID (`whd_...`)
- `X-Mailvex-Event-Id`: Unique event ID (`evt_...`)
- `X-Mailvex-Timestamp`: Unix epoch timestamp `t`
- `X-Mailvex-Signature`: `t=<unix>,v1=<hex hmac-sha256(secret, t + "." + body)>`

Receivers verify authenticity using their webhook secret and reject deliveries where `|now - t| > 300` seconds to prevent replay attacks.

### 1.2 Secret Rotation with 24-Hour Overlap
When a secret is rotated via `/v1/webhooks/{id}/rotate-secret`:
- For 24 hours, deliveries include signatures for both the old and new secrets:
  `X-Mailvex-Signature: t=...,v1=<new_sig>,v1=<old_sig>`
- This enables zero-downtime secret rollover without dropped payloads.

### 1.3 Automatic Circuit Breaking
- Endpoints that repeatedly fail (50 consecutive errors over >= 24h) are set to `disabled_failing`.
- An incident alert is triggered and the tenant is notified in the Console.
