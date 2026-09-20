# ADR 0002: SMTP Edge Engine Selection

## Status
Accepted

## Context
Mailvex requires an SMTP edge component that handles:
- Authenticated submission on port 587 (STARTTLS) and 465 (SMTPS)
- Inbound MX reception and bounce/FBL/abuse intake on port 25
- Strict tenant-aware authentication and sender authorization
- High concurrency and low resource footprint (suitable for home lab profile)

As per §4 of the Master Prompt, the candidates are:
1. Haraka (Node.js)
2. KumoMTA (Rust / Lua)
3. Postfix (C / Milters / Policy daemons)
4. Go library (`emersion/go-smtp`)

## Evaluation Matrix

| Criteria | Haraka | KumoMTA | Postfix | `emersion/go-smtp` |
|---|---|---|---|---|
| **Single-Runtime Alignment** | ❌ Requires Node.js | ❌ Separate Rust daemon | ❌ C / OS daemon | ✅ Native Go (same binary/runtime) |
| **Tenant-Aware Auth & Policy** | ⚠️ Custom JS plugins | ⚠️ Lua policy scripts | ❌ Complex external policy daemons | ✅ Direct Go interface to Core repos |
| **Memory Footprint** | ~150 MB (V8 VM) | ~80 MB | ~30 MB | **< 20 MB** (Go goroutines) |
| **Operational Simplicity** | Node packages, npm | Rust binary compilation | OS package configs | Single static binary in Docker |
| **TLS & STARTTLS Support** | ✅ Built-in | ✅ Built-in | ✅ Built-in | ✅ Built-in via Go standard `crypto/tls` |
| **MIME Normalization** | ✅ plugins | ✅ Lua filters | ⚠️ Milters | ✅ Direct integration with Go MIME parser |

## Decision
We select **`emersion/go-smtp`**.

### Rationale
1. **Single Runtime**: Enables the SMTP edge to run either as a dedicated container in the `standard`/`cluster` profile, or directly inside the `mailvex-allinone` process for the `homelab` profile without external runtime dependencies.
2. **Direct Authentication & Authorization**: The Go interface (`Backend`, `Session`, `AuthSession`) connects directly to Mailvex Core's credential store, rate limiters, trusted network CIDRs, and brute-force defenses without IPC or policy daemon latency.
3. **Open-Relay Safety**: Complete control over session lifecycle guarantees that unauthenticated sessions can never submit mail for external delivery.

## Consequences
- The SMTP edge will be implemented in `smtp/` as a clean Go service wrapping `emersion/go-smtp`.
- Outbound delivery remains strictly separated in `worker/`.
