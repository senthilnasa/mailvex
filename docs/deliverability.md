# Mailvex — Deliverability Architecture

## 1. Core Deliverability Subsystems

Mailvex includes comprehensive deliverability tooling required for high-reputation inbox placement:

### 1.1 Variable Envelope Return Path (VERP)
- Outbound messages receive unique `Return-Path: bounce+<signed_token>@bounce.<domain>`.
- When an asynchronous Delivery Status Notification (DSN) arrives at the bounce domain, the SMTP edge parses the token, maps it back to the exact recipient, records `message.bounced`, and updates suppressions.

### 1.2 Feedback Loops (ARF & Postmaster)
- Handles Abuse Reporting Format (ARF) reports from Yahoo, Microsoft JMRP, and postmaster addresses.
- Auto-suppresses complainers and calculates tenant/stream complaint rates.

### 1.3 RFC 8058 One-Click Unsubscribe
- Broadcast streams mandate:
  - `List-Unsubscribe: <https://track.domain.com/u/<token>>, <mailto:unsubscribe+<token>@bounce.domain.com>`
  - `List-Unsubscribe-Post: List-Unsubscribe=One-Click`
- `POST /u/<token>` instantly suppresses the recipient for the stream and returns HTTP 200 without user interaction.

### 1.4 IP Warmup Plans
- Automated daily sending volume ramps with per-provider caps (e.g. Gmail 50/day on Day 1, doubling daily).
- Holds warmup ramp if bounce or complaint thresholds are breached.
