# Mailvex PHP & Laravel SDK

Official PHP & Laravel client library for **Mailvex Email Infrastructure Engine**.

[![Latest Version on Packagist](https://img.shields.io/packagist/v/mailvex/mailvex-php.svg?style=flat-square)](https://packagist.org/packages/mailvex/mailvex-php)
[![PHP Version](https://img.shields.io/badge/php-%3E%3D8.1-8892BF.svg?style=flat-square)](https://php.net)
[![Laravel Version](https://img.shields.io/badge/laravel-9.x%20%7C%2010.x%20%7C%2011.x%20%7C%2012.x%20%7C%2013.x-FF2D20.svg?style=flat-square)](https://laravel.com)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg?style=flat-square)](LICENSE)

---

## Features

- 🚀 **Full PHP & Laravel Support**: Zero external dependencies (native `ext-curl`, `ext-json`), compatible with PHP 8.1 through 8.4+ and Laravel 9 through 13.
- 📎 **Attachments**: Attach files via local file path, raw binary/string data, or pre-encoded Base64.
- 👥 **Multiple Recipients & Batch Dispatch**:
  - Send to multiple recipients in a single `send()` call.
  - Send up to 500 individualized messages in a single `sendBatch()` call with per-recipient template variables and metadata.
- 🛡️ **Webhook Signature Verification**: HMAC-SHA256 signature verification (`X-Mailvex-Signature: t=...,v1=...`) with 5-minute replay tolerance, dual-key rotation overlap, and typed helper methods for bounce and complaint classifications.
- ✉️ **Native Laravel Mail Transport**: Fully integrates with Laravel's mail system (`Mail::to()->send()`), auto-converting attachments, headers, and metadata to Mailvex REST API payloads.
- ⚡ **Resilient Delivery**: Built-in exponential backoff retry and RFC 8058 One-Click Unsubscribe automation.

---

## Installation

Install the package via Composer:

```bash
composer require mailvex/mailvex-php
```

---

## Standalone PHP Usage

### 1. Initialize Client

```php
use Mailvex\Mailvex;
use Mailvex\Client;

$client = Mailvex::client(
    apiKey: 'mvx_live_a89f30b2c1409f921df048bbef8001a1',
    baseUrl: 'https://api.mailvex.yourdomain.com' // Explicit URL required in production
);
```

> [!IMPORTANT]
> **Base URL in Production**: In development, `baseUrl` can default to `http://localhost:8080`. In production, leaving the base URL empty or pointing to localhost will throw an `InvalidArgumentException`.

---

### 2. Send Email with Attachments

Attach files from paths, raw data, or Base64 strings:

```php
use Mailvex\Models\EmailMessage;
use Mailvex\Models\Attachment;

$message = EmailMessage::create()
    ->from('Security Alert <alerts@mycompany.com>')
    ->to('alice@example.com')
    ->cc('auditor@example.com')
    ->replyTo('support@mycompany.com')
    ->subject('Action Required: Security Audit Report')
    ->html('<h2>Security Audit Complete</h2><p>Please find your security report attached.</p>')
    ->text("Security Audit Complete\nPlease find your security report attached.")
    ->attachFromPath('/path/to/report.pdf', 'security-audit.pdf', 'application/pdf')
    ->attachFromData('Col1,Col2\nVal1,Val2', 'metrics.csv', 'text/csv')
    ->header('X-Agent-Origin', 'worker-4')
    ->tag('security-report')
    ->metadata('report_id', 'rep_001')
    ->idempotencyKey('b8b3d6cb-0e9e-4e31-893d-4c3d82a17f22');

$response = $client->send($message);

echo "Message Queued! ID: " . $response['id'];
```

---

### 3. Send to Multiple Recipients in One Call

Provide an array of email addresses to `to()`, `cc()`, or `bcc()`:

```php
$message = EmailMessage::create()
    ->from('Team <team@mycompany.com>')
    ->to([
        'alice@example.com',
        'bob@example.com',
        'carol@example.com'
    ])
    ->subject('Project Update')
    ->html('<p>Hi Team, weekly sprint update is ready.</p>');

$response = $client->send($message);
```

---

### 4. Send Batch Emails (Up to 500 Distinct Messages in One Call)

Use `sendBatch()` to dispatch distinct, personalized messages to multiple recipients in a single HTTP request:

```php
use Mailvex\Models\BatchMessage;
use Mailvex\Models\BatchRecipient;

$batch = BatchMessage::create()
    ->from('Billing <billing@mycompany.com>')
    ->subject('Your Monthly Statement')
    ->template('tmpl_monthly_statement')
    ->stream('broadcast')
    ->addRecipient('alice@example.com', [
        'name' => 'Alice',
        'amount' => '$49.00',
        'due_date' => 'Nov 1, 2026'
    ], ['customer_tier' => 'pro'])
    ->addRecipient('bob@example.com', [
        'name' => 'Bob',
        'amount' => '$99.00',
        'due_date' => 'Nov 3, 2026'
    ], ['customer_tier' => 'enterprise'])
    ->attachFromPath('/path/to/terms.pdf');

$response = $client->sendBatch($batch);

echo "Batch queued! Total: {$response['total']}, Accepted: {$response['accepted']}\n";
print_r($response['message_ids']);
```

---

## Message Streams Guide

Mailvex organizes outbound traffic into **Message Streams**. Streams provide physical isolation between critical operational notifications and high-volume broadcast campaigns.

| Stream | Intended Use | Queue Priority | Unsubscribe Handling | Suppression Scope |
|---|---|---|---|---|
| `transactional` *(Default)* | OTPs, password resets, admission notices, invoices, account alerts | **P1 (Expedited)** | No marketing unsubscribe headers inserted | Not affected by marketing / broadcast unsubscribes |
| `broadcast` | Marketing newsletters, student announcements, promotional broadcasts | **P2 (Bulk)** | Automatically injects RFC 8058 One-Click `List-Unsubscribe` headers | Scoped to broadcast; recipient unsubscribe blocks broadcast only |
| *Custom Streams* | Created via `POST /v1/streams` (e.g. `admissions`, `billing`) | Configurable | Configurable per stream type | Partitioned by stream ID |

### Stream Behavior & Guarantees
1. **Queue Priority & Throughput Protection**: `transactional` emails are dispatched to Priority 1 (`mvx:delivery:p1`), ensuring critical transactional emails are never delayed behind a 50,000-recipient newsletter blast.
2. **Independent Suppression Lists**: Suppression checks in Mailvex are scoped by `(tenant_id, stream_id, address)`. If an applicant unsubscribes from marketing or broadcast updates, their address is **only** suppressed on the broadcast stream. Critical transactional messages (e.g. password resets, admission offers) will continue to deliver normally.
3. **Automatic RFC 8058 Headers**: When `stream` is `broadcast`, Mailvex automatically generates signed VERP tokens and embeds RFC 8058 `List-Unsubscribe` and `List-Unsubscribe-Post: List-Unsubscribe=One-Click` headers into the raw MIME message.
4. **Setting Stream in Laravel**:
   - `MailvexTransport` defaults to `stream: transactional`.
   - To send as `broadcast` via Laravel's native mailer, set the `X-Mailvex-Stream` header:
     ```php
     // In a Mailable:
     public function headers(): Headers
     {
         return new Headers(text: ['X-Mailvex-Stream' => 'broadcast']);
     }

     // Or via Mail::send() callback:
     Mail::send([], [], function ($message) {
         $message->getHeaders()->addTextHeader('X-Mailvex-Stream', 'broadcast');
     });
     ```
   - When using the `Mailvex` Facade or SDK directly:
     ```php
     $email->stream('broadcast');
     ```

---

## Webhook Engine & Event Specifications

Mailvex delivers real-time HTTP POST notifications for delivery outcomes, recipient engagement, and reputation events.

### Security & Signature Algorithm
Each delivery includes the header:
```http
X-Mailvex-Signature: t=1730107200,v1=3c6f8b9e4a2...
```
- **Signature verification**: `v1 = HMAC-SHA256(secret, "$timestamp.$rawPayload")`.
- **Clock drift defense**: Verifies `|current_time - t| <= 300` seconds (5-minute tolerance window).
- **Dual-Key Rotation Overlap**: When rotating secrets, Mailvex includes signatures for both current and previous secrets for 24 hours.
- **De-duplication**: The top-level `id` (`evt_...`) is **unique per event and stable across retries**. Use `$event->getId()` to de-duplicate events in your database. The header `X-Mailvex-Delivery-Id` (`whd_...`) identifies the individual transmission attempt.

### Webhook Event Catalog

#### 1. `message.delivered`
Fired when the destination MX server accepts the message with SMTP `250 2.0.0 OK`.
```json
{
  "id": "evt_01HZX876543210ABCDEFGHJKMN",
  "event": "message.delivered",
  "tenant_id": "tenant_01HZX876543210ABCDEFGHJK",
  "occurred_at": "2026-10-30T00:15:00Z",
  "data": {
    "message_id": "msg_01HZX876543210ABCDEFGHJKMN",
    "recipient": "applicant@example.com",
    "smtp_code": "250",
    "response": "250 2.0.0 OK: queued as 482910"
  }
}
```

#### 2. `message.bounced`
Fired when delivery fails with an SMTP rejection code (5xx or terminal 4xx).
```json
{
  "id": "evt_01HZX876543210ABCDEFGHJK01",
  "event": "message.bounced",
  "tenant_id": "tenant_01HZX876543210ABCDEFGHJK",
  "occurred_at": "2026-10-30T00:15:05Z",
  "data": {
    "message_id": "msg_01HZX876543210ABCDEFGHJKMN",
    "recipient": "invalid-address@example.com",
    "classification": "hard_bounce",
    "action": "suppress_immediate",
    "reason": "550 5.1.1 User unknown",
    "smtp_code": "550",
    "enhanced_code": "5.1.1"
  }
}
```

**Bounce Classifications & Recommended Handling:**
| Classification (`$event->getBounceType()`) | Action (`data.action`) | Meaning | Recommended Action |
|---|---|---|---|
| `hard_bounce` | `suppress_immediate` | Mailbox does not exist (5.1.1) or domain unrouteable | **Block / Suppress address permanently** (`$event->isHardBounce() === true`) |
| `soft_bounce` | `retry` | Mailbox full / quota exceeded (4.2.2) | **Do not block.** Mailvex auto-retries |
| `rate_limited` | `retry` | Too many connections / throttled (4.7.28) | **Do not block.** Mailvex auto-retries |
| `mailbox_unavailable` | `retry` | Target system temporary downtime (4.2.1) | **Do not block.** Mailvex auto-retries |
| `spam_rejection` | `dead_letter` | Remote server blocked content/IP | Inspect sender reputation or content |

#### 3. `message.complained`
Fired when a recipient reports the message as spam via an ISP Feedback Loop (ARF) or RFC 8058 one-click list unsubscribe.
```json
{
  "id": "evt_01HZX876543210ABCDEFGHJK02",
  "event": "message.complained",
  "tenant_id": "tenant_01HZX876543210ABCDEFGHJK",
  "occurred_at": "2026-10-30T00:20:00Z",
  "data": {
    "message_id": "msg_01HZX876543210ABCDEFGHJKMN",
    "recipient": "user@example.com",
    "feedback_type": "abuse",
    "user_agent": "Mozilla/5.0..."
  }
}
```
*Feedback Types (`$event->getFeedbackType()`):* `abuse`, `spam`, `fraud`, `misc`. Mailvex automatically suppresses complained recipients.

#### 4. `message.opened` & `message.clicked`
Fired when open-tracking pixel is requested or link redirection occurs (bot hits are filtered automatically).
- `message.opened`: `{"message_id": "...", "recipient": "...", "ip": "198.51.100.x", "user_agent": "..."}`
- `message.clicked`: `{"message_id": "...", "recipient": "...", "url": "https://krea.edu.in/apply", "ip": "...", "user_agent": "..."}`

### Outbound Delivery Retry Policy
When receiving temporary errors (4xx / `soft_bounce` / `rate_limited`), Mailvex's delivery queue schedules retries using a 9-tier schedule with **±20% randomized jitter**:
- Attempt 1: 1 minute
- Attempt 2: 5 minutes
- Attempt 3: 15 minutes
- Attempt 4: 30 minutes
- Attempt 5: 1 hour
- Attempt 6: 2 hours
- Attempt 7: 6 hours
- Attempt 8: 12 hours
- Attempt 9: 24 hours

Maximum delivery lifetime is **72 hours**. If a soft bounce cannot be delivered after 9 attempts or 72 hours, its status transitions to `failed` and it is routed to the Dead Letter Queue (DLQ).

---

## Laravel Integration (Laravel 9, 10, 11, 12, 13)

### 1. Configuration & Publishing

The package includes auto-discovery for `MailvexServiceProvider` and the `Mailvex` Facade.

Publish configuration:
```bash
php artisan vendor:publish --tag=mailvex-config
```

Add your credentials to `.env`:
```env
MAIL_MAILER=mailvex

MAILVEX_API_KEY=mvx_live_a89f30b2c1409f921df048bbef8001a1
MAILVEX_API_URL=https://api.mailvex.yourdomain.com
MAILVEX_WEBHOOK_SECRET=whsec_d41d8cd98f00b204e9800998ecf8427e
MAILVEX_TENANT_ID=tenant_01HZX876543210ABCDEFGHJK
```

In `config/mail.php`:
```php
'mailers' => [
    'mailvex' => [
        'transport' => 'mailvex',
    ],
],
```

---

### 2. Native Laravel Mailer with Attachments

```php
use Illuminate\Support\Facades\Mail;
use App\Mail\AdmissionOfferMailable;

// 1. Using Mailable
Mail::to(['applicant@example.com', 'guardian@example.com'])
    ->send(new AdmissionOfferMailable($offer));

// 2. Using inline Mail::send with attachment
Mail::send([], [], function ($message) {
    $message->to('student@example.com')
        ->from('admissions@krea.edu.in', 'Krea Admissions')
        ->subject('Offer Letter - Krea University')
        ->html('<p>Congratulations! Please find your offer letter attached.</p>')
        ->attach('/path/to/offer-letter.pdf', [
            'as' => 'Krea-Offer-Letter.pdf',
            'mime' => 'application/pdf',
        ]);
});
```

---

### 3. Handling Webhooks in Laravel

Register the webhook route in `routes/api.php`:
```php
use Illuminate\Support\Facades\Route;

// Registers POST /api/mailvex/webhook with signature verification
Route::mailvexWebhooks('mailvex/webhook');
```

Listen for `Mailvex\Laravel\Events\WebhookReceived`:

```php
namespace App\Listeners;

use Mailvex\Laravel\Events\WebhookReceived;
use Illuminate\Support\Facades\Log;

class HandleMailvexWebhook
{
    public function handle(WebhookReceived $event): void
    {
        $webhook = $event->event;

        // 1. De-duplicate using stable Event ID
        $eventId = $webhook->getId(); // "evt_01HZX..."

        // 2. Handle outcome
        if ($webhook->isBounced()) {
            $email = $webhook->getRecipient();
            $type = $webhook->getBounceType(); // "hard_bounce", "soft_bounce", etc.

            if ($webhook->isHardBounce()) {
                // Unrecoverable hard bounce: Mark contact as invalid in CRM
                Log::warning("Hard bounce for {$email}: " . $webhook->getBounceReason());
            } elseif ($webhook->isSoftBounce()) {
                // Transient bounce: Mailvex will auto-retry up to 72 hours
                Log::info("Transient soft bounce ({$type}) for {$email}");
            }
        } elseif ($webhook->isDelivered()) {
            $messageId = $webhook->getMessageId();
            // Update CRM message delivery record
        } elseif ($webhook->isComplained()) {
            // Unsubscribe user immediately
        }
    }
}
```

---

## Testing

Run tests using PHP:

```bash
php tests/run_tests.php
```

Or run PHPUnit:

```bash
./vendor/bin/phpunit
```

---

## License

The Mailvex PHP & Laravel SDK is open-source software licensed under the [MIT license](LICENSE).
