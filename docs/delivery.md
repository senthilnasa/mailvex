# Mailvex — Delivery Engine & Transport Architecture

## 1. Outbound Worker Architecture

The **Mailvex Worker** is the sole subsystem that executes outbound SMTP connections.

### 1.1 Delivery Lifecycle
1. **Claim Task**: Pull recipient job from priority stream.
2. **Pre-flight Checks**: Verify message is not cancelled, expired, or suppressed.
3. **Suppression Guard**: Check `suppressions` for recipient address, domain block, or unsubscribe (for broadcast streams).
4. **IP & Routing Selection**: Select IP from assigned pool using configured strategy (`round_robin`, `least_used`, `weighted`).
5. **Transport Selection**:
   - **Direct MX**: Resolve destination domain MX records with caching and fallback to A/AAAA.
   - **Relay (Smart-Host)**: Route through configured tenant/platform relay.
6. **MTA-STS & DANE**: Perform TLS policy evaluation.
7. **Header Hygiene**:
   - Strip internal routing headers (`X-Mailvex-*`, private IPs).
   - Ensure RFC-mandated headers (`Message-ID`, `Date`, `MIME-Version`).
   - Add `Feedback-ID: <tenant>:<stream>:<campaign>:mailvex`.
   - Add `List-Unsubscribe` & `List-Unsubscribe-Post` for broadcast streams.
   - Set VERP `Return-Path: bounce+<token>@<bounce_domain>`.
8. **DKIM Signing**: Canonicalize and sign with domain's active selector.
9. **SMTP Transmission**: Establish TLS session, authenticate (if relay), and deliver.
10. **Result Classification**: Classify response (`delivered`, `deferred`, `hard_bounce`, `soft_bounce`, `rate_limited`).
11. **Event & Retry**: Emit `message.delivered` or schedule backoff in `mvx:delayed`.
