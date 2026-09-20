# Mailvex — Networking & IP Binding

## 1. Outbound IP Management

Mailvex supports sending from multiple public IPv4 and IPv6 addresses across pools.

### 1.1 Source IP Binding in Go
To transmit from a specific public IP:
- Outbound SMTP connections bind `net.Dialer.LocalAddr` to the specified IP.
- The `EHLO` command matches the IP's Forward-Confirmed Reverse DNS (FCrDNS) hostname.

### 1.2 Docker Networking Requirements
Standard Docker bridge networking (`bridge`) performs SNAT (Source Network Address Translation) using the host's primary outbound interface. This prevents containers from selecting arbitrary public source IPs.

Solutions:
1. **`network_mode: host` (Recommended)**: Worker containers attach directly to host network interfaces, allowing direct binding to all assigned IP addresses.
2. **`macvlan` / `ipvlan`**: Containers receive dedicated virtual MAC/IP attachments on the physical network.

---

## 2. Inbound Port Assignments

| Port | Protocol | Purpose | Authentication |
|---|---|---|---|
| **25** | SMTP | Inbound MX reception, DSN bounce intake, abuse intake, LAN IoT IP auth | Optional (inbound routes only) |
| **587** | SMTP | Submission (STARTTLS) | Required (AUTH PLAIN/LOGIN) |
| **465** | SMTPS | Implicit TLS submission | Required (AUTH PLAIN/LOGIN) |
| **80** | HTTP | ACME HTTP-01 challenges, HTTP -> HTTPS redirects | Public |
| **443** | HTTPS | Mailvex API, Console, Webhooks, Tracking links | TLS termination |
