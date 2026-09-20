# ADR 0003: Reverse Proxy & On-Demand TLS Selection

## Status
Accepted

## Context
Mailvex requires an HTTP reverse proxy that can:
- Terminate TLS for Console and API
- Route traffic to appropriate backend microservices
- Provide **On-Demand ACME TLS** for custom tenant tracking domains (e.g. `track.tenant-domain.com`) with validation against Mailvex API (`/v1/system/ask-domain`)
- Support lightweight single-server deployments and home labs

Options evaluated: Caddy vs Traefik.

## Evaluation Matrix

| Feature | Caddy 2 | Traefik v3 |
|---|---|---|
| **On-Demand ACME TLS** | Native `on_demand_tls` with `ask` endpoint | Requires Enterprise or plugins |
| **Config Simplicity** | Declarative `Caddyfile` | Complex YAML/Labels |
| **Memory Footprint** | ~30 MB | ~60 MB |
| **HTTP/3 Support** | Native | Native |

## Decision
We select **Caddy 2** as the primary reverse proxy.

### Rationale
Caddy provides native `on_demand_tls` directives that query an internal HTTP endpoint (`http://api:8080/v1/system/ask-domain?domain=...`). When an external request arrives for a custom tracking or hosted unsubscribe domain, Caddy checks with Mailvex API; if the domain is verified and owned by an active tenant, Caddy automatically obtains and manages a Let's Encrypt / ZeroSSL TLS certificate.

## Consequences
- `deploy/caddy/Caddyfile` will be shipped as the standard reverse proxy configuration.
- Mailvex API provides an internal `/v1/system/ask-domain` handler returning HTTP 200 for valid custom domains and 404 for unknown domains.
