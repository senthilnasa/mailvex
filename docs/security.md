# Mailvex — Security Architecture & Threat Model

## 1. Envelope Encryption

Mailvex uses a two-tier envelope encryption architecture:

```text
Key Encryption Key (KEK)
  └── Stored in Environment (MVX_MASTER_KEY), file, or Vault
        └── Encrypts per-tenant Data Encryption Keys (DEKs)
              └── Stored in `tenant_keys` table
                    └── Encrypts tenant secrets (DKIM private keys, webhook secrets, bodies)
```

1. **KEK (Key Encryption Key)**:
   - AES-256 (32 bytes base64).
   - Never stored in the database.
   - Configured via `MVX_MASTER_KEY` (env) or `/etc/mailvex/master.key` (file).
2. **DEK (Data Encryption Key)**:
   - 256-bit AES key generated at tenant creation.
   - Encrypted with KEK using AES-256-GCM.
   - Ciphertext stored in `tenant_keys` with `key_version`.
3. **Secret Encryption**:
   - DKIM private keys, webhook secrets, stored message bodies are encrypted at rest using the tenant DEK with AES-256-GCM.

---

## 2. Authentication & Passwords

1. **User Passwords**:
   - Hashed using **Argon2id** (`time=1, memory=64MB, threads=4, keyLen=32, saltLen=16`).
   - Plaintext passwords are never logged or stored.
2. **API Keys**:
   - Format: `mvx_live_<32_random_bytes>` or `mvx_test_<32_random_bytes>`.
   - Stored as `sha256(secret)` in the database.
   - Key prefixes and the last 4 characters are kept for display purposes.
3. **SMTP Passwords**:
   - Generated randomly and displayed once.
   - Stored as Argon2id hashes in `smtp_credentials`.

---

## 3. Application Security Checklist

- **CRLF Injection Prevention**: Header names and values are strictly validated against `\r` and `\n`. Custom headers are restricted to allowlisted formats.
- **SSRF Defense**: All outbound HTTP requests (webhooks, avatar lookups) resolve destination IPs and strictly reject loopback, link-local, and private RFC 1918 ranges.
- **Message Inspector Sandboxing**: Message HTML bodies are displayed exclusively in an isolated `iframe` with `sandbox=""` and a strict Content Security Policy (`default-src 'none'; img-src data:; style-src 'unsafe-inline'`). Remote images are blocked by default.
- **Open Relay Prevention**: The SMTP edge enforces authentication on ports 587/465, permits unauthenticated delivery only to verified inbound recipient routes on port 25, and never relays outbound mail directly.
