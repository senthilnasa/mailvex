# Mailvex PHP & Laravel SDK

Official PHP & Laravel client library for **Mailvex Email Infrastructure Engine**.

[![Latest Version on Packagist](https://img.shields.io/packagist/v/mailvex/mailvex-php.svg?style=flat-square)](https://packagist.org/packages/mailvex/mailvex-php)
[![PHP Version](https://img.shields.io/badge/php-%3E%3D8.1-8892BF.svg?style=flat-square)](https://php.net)
[![Laravel Version](https://img.shields.io/badge/laravel-9.x%20%7C%2010.x%20%7C%2011.x%20%7C%2012.x-FF2D20.svg?style=flat-square)](https://laravel.com)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg?style=flat-square)](LICENSE)

---

## Features

- 🚀 **Full PHP & Laravel Support**: Works as a standalone PHP SDK and features seamless Laravel auto-discovery.
- 📎 **Attachments**: Attach files easily via local file path, raw binary/string data, or pre-encoded Base64.
- 👥 **Multiple Recipients & Batch Email**:
  - Send to multiple recipients in a single `send()` call.
  - Send up to 500 individualized messages in a single `sendBatch()` call with per-recipient template variables and metadata.
- 🛡️ **Webhook Verification**: Built-in HMAC-SHA256 signature verification (`X-Mailvex-Signature: t=...,v1=...`) with configurable replay tolerance window.
- ✉️ **Native Laravel Mail Transport**: Use Laravel's standard `Mail::to()->send()` driver, automatically converting Mime attachments and headers to Mailvex REST API calls.
- ⚡ **Zero External Dependencies**: Built with native PHP `ext-curl` and `ext-json` with automatic retry and exponential backoff.

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
    baseUrl: 'https://api.mailvex.yourdomain.com' // or 'http://localhost:8080' for local dev
);
```

---

### 2. Send Email with Attachments

You can attach files from paths, raw data, or Base64 strings:

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
    ->attachFromData('Invoice CSV data...', 'invoice.csv', 'text/csv')
    ->header('X-Agent-Origin', 'worker-4')
    ->tag('security-report')
    ->metadata('report_id', 'rep_001')
    ->idempotencyKey('b8b3d6cb-0e9e-4e31-893d-4c3d82a17f22');

$response = $client->send($message);

echo "Message Queued! ID: " . $response['id'];
```

---

### 3. Send to Multiple Recipients in One Call

You can provide an array of email addresses to the `to()`, `cc()`, or `bcc()` methods:

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

### 4. Send Batch Emails (Up to 500 Distinct Messages at Once)

Use `sendBatch()` to dispatch distinct messages to multiple recipients with personalized template variables in a single HTTP request:

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

### 5. Webhook Signature Verification

Mailvex signs incoming webhook deliveries using HMAC-SHA256 with the header `X-Mailvex-Signature: t=<timestamp>,v1=<signature>`.

```php
use Mailvex\Webhooks\Webhook;
use Mailvex\Exceptions\WebhookVerificationException;

$payload = file_get_contents('php://input');
$signatureHeader = $_SERVER['HTTP_X_MAILVEX_SIGNATURE'] ?? '';
$secret = 'whsec_d41d8cd98f00b204e9800998ecf8427e';

try {
    // Verifies HMAC and checks timestamp drift within 300s tolerance
    $event = Webhook::constructEvent($payload, $signatureHeader, $secret);

    // Handle event types
    switch ($event->getType()) {
        case 'message.delivered':
            $messageId = $event->getMessageId();
            $recipient = $event->getRecipient();
            // Update delivery status in your database
            break;

        case 'message.bounced':
            // Suppress or flag bounced recipient
            break;

        case 'message.complained':
            // Handle spam complaint
            break;
    }

    http_response_code(200);
    echo json_encode(['status' => 'success']);
} catch (WebhookVerificationException $e) {
    http_response_code(401);
    echo json_encode(['error' => $e->getMessage()]);
}
```

---

## Laravel Integration

### 1. Configuration & Publishing

The package automatically registers its Service Provider and Facade in Laravel.

Publish the configuration file:

```bash
php artisan vendor:publish --tag=mailvex-config
```

Add your credentials to your `.env` file:

```env
MAIL_MAILER=mailvex

MAILVEX_API_KEY=mvx_live_a89f30b2c1409f921df048bbef8001a1
MAILVEX_API_URL=http://localhost:8080
MAILVEX_WEBHOOK_SECRET=whsec_d41d8cd98f00b204e9800998ecf8427e
MAILVEX_TENANT_ID=tenant_01HZX876543210ABCDEFGHJK
```

In `config/mail.php`, register the `mailvex` mailer transport:

```php
'mailers' => [
    // ...
    'mailvex' => [
        'transport' => 'mailvex',
    ],
],
```

---

### 2. Sending Mail with Laravel's Native Mailer & Attachments

Use Laravel's standard `Mail` facade or Mailable classes. Attachments and recipients are automatically transformed:

```php
use Illuminate\Support\Facades\Mail;
use App\Mail\InvoiceMailable;

// 1. Using Mailable
Mail::to(['client@example.com', 'accounting@example.com'])
    ->send(new InvoiceMailable($invoice));

// 2. Using inline Mail::raw or Mail::send
Mail::send([], [], function ($message) {
    $message->to(['user1@example.com', 'user2@example.com'])
        ->from('billing@mycompany.com', 'Billing Team')
        ->subject('Invoice #1042')
        ->html('<p>Attached is your invoice.</p>')
        ->attach('/path/to/invoice.pdf', [
            'as' => 'invoice-1042.pdf',
            'mime' => 'application/pdf',
        ]);
});
```

---

### 3. Using the Mailvex Laravel Facade

You can also access Mailvex directly via the `Mailvex` Facade:

```php
use Mailvex\Laravel\Facades\Mailvex;
use Mailvex\Models\EmailMessage;
use Mailvex\Models\BatchMessage;

// Send email with attachments
$response = Mailvex::send(
    EmailMessage::create()
        ->from('support@mycompany.com')
        ->to('user@example.com')
        ->subject('Support Ticket #8291')
        ->text('We have updated your ticket.')
        ->attachFromPath(storage_path('logs/ticket.log'))
);

// Send batch emails in one call
$batchResponse = Mailvex::sendBatch(
    BatchMessage::create()
        ->from('promo@mycompany.com')
        ->subject('Special Discount')
        ->template('tmpl_discount')
        ->addRecipient('user1@example.com', ['code' => 'SAVE10'])
        ->addRecipient('user2@example.com', ['code' => 'SAVE20'])
);

// Inspect messages or domains
$messageDetails = Mailvex::messages()->get('msg_01HZX876543210ABCDEFGHJKMN');
$domains = Mailvex::domains()->list();
```

---

### 4. Laravel Webhook Route & Event

Declare the webhook route in `routes/api.php` or `routes/web.php` using the route macro:

```php
use Illuminate\Support\Facades\Route;

// Automatically registers POST /api/mailvex/webhook pointing to WebhookController
Route::mailvexWebhooks('mailvex/webhook');
```

When a webhook arrives with a valid `X-Mailvex-Signature`, it fires the `Mailvex\Laravel\Events\WebhookReceived` event.

Listen for it in your `EventServiceProvider` or listener:

```php
namespace App\Listeners;

use Mailvex\Laravel\Events\WebhookReceived;

class HandleMailvexWebhook
{
    public function handle(WebhookReceived $event): void
    {
        $webhookEvent = $event->event;
        $type = $webhookEvent->getType(); // e.g. "message.delivered", "message.bounced"
        $data = $webhookEvent->getData();

        if ($webhookEvent->isBounced()) {
            \Log::warning("Email bounced for recipient: " . $webhookEvent->getRecipient());
        }
    }
}
```

> **Note:** Remember to exclude `mailvex/webhook` from CSRF verification in `bootstrap/app.php` (Laravel 11+) or `app/Http/Middleware/VerifyCsrfToken.php` (Laravel 9/10).

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
