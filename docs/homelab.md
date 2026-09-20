# Mailvex — Home Lab & Local Server Deployment Guide

## 1. Overview

Mailvex provides a dedicated **`homelab`** profile designed for small servers, Raspberry Pis (4/5), and home NAS devices running on residential internet.

Residential ISPs typically block outbound port 25 and assign dynamic IPs with poor deliverability reputation. The `homelab` profile solves this via:
1. **Smart-Host Relay**: Outbound mail is routed through external authenticated relays (Amazon SES, Gmail Workspace, Mailgun, SendGrid, Brevo) with automatic `From` header rewriting.
2. **Device Registry**: Collects and normalizes mail from local network devices (security cameras, printers, TrueNAS, Home Assistant, Proxmox, UPS monitors).
3. **Trusted Network Security**: Allows unauthenticated or IP-authenticated SMTP submission strictly from verified private subnets (`192.168.0.0/16`, `10.0.0.0/8`).
4. **Mail to Push Routing**: Converts incoming device notifications to push alerts (ntfy, Telegram, Home Assistant, Gotify, Pushover).
5. **Ultra-Low Resource Footprint**: The `mailvex-allinone` image combines API, Worker, Webhooks, Scheduler, and SMTP edge into a single lightweight process idling under 512 MB RAM.

---

## 2. IoT Device Ingestion

In Console → Devices, users can register appliances:
- Enter device name, model preset, and LAN IP address (e.g. `192.168.1.50`).
- Generates a configuration card with SMTP host, port 587/25, username, and password.
- If a device lacks authentication capabilities, IP-authenticated submission from the registered IP is permitted within the trusted subnet.
