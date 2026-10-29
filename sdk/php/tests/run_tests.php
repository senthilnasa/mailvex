<?php

declare(strict_types=1);

require_once __DIR__ . '/../vendor/autoload.php';

use Mailvex\Client;
use Mailvex\Config;
use Mailvex\Http\HttpClientInterface;
use Mailvex\Http\Response;
use Mailvex\Mailvex;
use Mailvex\Models\Attachment;
use Mailvex\Models\BatchMessage;
use Mailvex\Models\BatchRecipient;
use Mailvex\Models\EmailMessage;
use Mailvex\Webhooks\Webhook;
use Mailvex\Exceptions\WebhookVerificationException;

// Mock HTTP Client for Testing
class MockHttpClient implements HttpClientInterface
{
    public array $lastRequest = [];
    public ?Response $nextResponse = null;

    public function request(
        string $method,
        string $url,
        array $headers = [],
        ?string $body = null,
        int $timeout = 30
    ): Response {
        $this->lastRequest = [
            'method' => $method,
            'url' => $url,
            'headers' => $headers,
            'body' => $body ? json_decode($body, true) : null,
            'timeout' => $timeout,
        ];

        return $this->nextResponse ?? new Response(200, [], json_encode(['status' => 'ok']));
    }
}

function test(string $name, callable $callback): void
{
    try {
        $callback();
        echo "[PASS] {$name}\n";
    } catch (\Throwable $e) {
        echo "[FAIL] {$name}: {$e->getMessage()}\n";
        echo $e->getTraceAsString() . "\n";
        exit(1);
    }
}

// 1. Test Attachment
test('Attachment fromData, fromBase64, and fromPath', function () {
    $att1 = Attachment::fromData('hello world', 'test.txt', 'text/plain', 'cid_123');
    $arr1 = $att1->toArray();
    assert($arr1['filename'] === 'test.txt');
    assert($arr1['content_type'] === 'text/plain');
    assert($arr1['content'] === base64_encode('hello world'));
    assert($arr1['content_id'] === 'cid_123');

    $tempFile = sys_get_temp_dir() . '/test_mailvex_' . uniqid() . '.txt';
    file_put_contents($tempFile, 'file contents test');
    $att2 = Attachment::fromPath($tempFile, 'renamed.txt');
    $arr2 = $att2->toArray();
    assert($arr2['filename'] === 'renamed.txt');
    assert($arr2['content'] === base64_encode('file contents test'));
    unlink($tempFile);
});

// 2. Test EmailMessage Builder (Single and Multiple Recipients with Attachments)
test('EmailMessage Builder with multiple recipients and attachments', function () {
    $msg = EmailMessage::create()
        ->from('Sender <sender@example.com>')
        ->to(['alice@example.com', 'bob@example.com'])
        ->cc('carol@example.com')
        ->subject('Meeting Notes')
        ->html('<p>Attached are the notes</p>')
        ->text('Attached are the notes')
        ->attachFromData('PDF_BYTES_HERE', 'notes.pdf', 'application/pdf')
        ->header('X-Campaign', 'weekly-all-hands')
        ->tag('notes')
        ->metadata('meeting_id', 'mtg_001')
        ->idempotencyKey('idem_single_001');

    $payload = $msg->toArray();
    assert($payload['from'] === 'Sender <sender@example.com>');
    assert($payload['to'] === ['alice@example.com', 'bob@example.com']);
    assert($payload['cc'] === ['carol@example.com']);
    assert(count($payload['attachments']) === 1);
    assert($payload['attachments'][0]['filename'] === 'notes.pdf');
    assert($payload['headers']['X-Campaign'] === 'weekly-all-hands');
    assert($payload['tags'] === ['notes']);
    assert($payload['metadata']['meeting_id'] === 'mtg_001');
    assert($msg->getIdempotencyKey() === 'idem_single_001');
});

// 3. Test BatchMessage Builder (Multiple Distinct Emails in One Call)
test('BatchMessage Builder with multiple recipients and per-recipient variables', function () {
    $batch = BatchMessage::create()
        ->from('System <no-reply@example.com>')
        ->subject('Monthly Statement')
        ->template('tmpl_statement')
        ->addRecipient('user1@example.com', ['name' => 'Alice', 'balance' => '$100'])
        ->addRecipient('user2@example.com', ['name' => 'Bob', 'balance' => '$250'], ['tier' => 'gold'])
        ->idempotencyKey('idem_batch_001');

    $payload = $batch->toArray();
    assert($payload['from'] === 'System <no-reply@example.com>');
    assert(count($payload['recipients']) === 2);
    assert($payload['recipients'][0]['to'] === 'user1@example.com');
    assert($payload['recipients'][0]['template_variables']['name'] === 'Alice');
    assert($payload['recipients'][1]['metadata']['tier'] === 'gold');
});

// 4. Test Client Send & SendBatch via Mock HTTP Client
test('Client send single email and batch email', function () {
    $mockHttp = new MockHttpClient();
    $config = new Config('mvx_test_mock_key', 'https://api.mailvex.example.com', 'tenant_123');
    $client = new Client($config, $mockHttp);

    // Send single
    $mockHttp->nextResponse = new Response(202, [], json_encode([
        'id' => 'msg_01HZX876543210ABCDEFGHJKMN',
        'status' => 'queued',
        'environment' => 'development',
    ]));

    $email = EmailMessage::create()
        ->from('sender@example.com')
        ->to('recipient@example.com')
        ->subject('Test Subject')
        ->text('Hello!')
        ->idempotencyKey('idem_test_single');

    $resp = $client->send($email);
    assert($resp['id'] === 'msg_01HZX876543210ABCDEFGHJKMN');
    assert($mockHttp->lastRequest['method'] === 'POST');
    assert($mockHttp->lastRequest['url'] === 'https://api.mailvex.example.com/v1/email');
    assert($mockHttp->lastRequest['headers']['Authorization'] === 'Bearer mvx_test_mock_key');
    assert($mockHttp->lastRequest['headers']['X-Tenant-ID'] === 'tenant_123');
    assert($mockHttp->lastRequest['headers']['Idempotency-Key'] === 'idem_test_single');

    // Send batch
    $mockHttp->nextResponse = new Response(200, [], json_encode([
        'batch_id' => 'batch_01HZX876543210ABCDEFGHJK',
        'total' => 2,
        'accepted' => 2,
        'failed' => 0,
        'message_ids' => ['msg_1', 'msg_2'],
    ]));

    $batch = BatchMessage::create()
        ->from('sender@example.com')
        ->subject('Batch Subject')
        ->recipients(['user1@test.com', 'user2@test.com']);

    $batchResp = $client->sendBatch($batch, 'idem_test_batch');
    assert($batchResp['total'] === 2);
    assert($mockHttp->lastRequest['method'] === 'POST');
    assert($mockHttp->lastRequest['url'] === 'https://api.mailvex.example.com/v1/email/batch');
    assert($mockHttp->lastRequest['headers']['Idempotency-Key'] === 'idem_test_batch');
});

// 5. Test Webhook Signature Verification and Event Construction
test('Webhook signature verification and event parsing', function () {
    $secret = 'whsec_test_secret_key_12345';
    $payload = json_encode([
        'id' => 'evt_01HZX876543210',
        'type' => 'message.delivered',
        'data' => [
            'message_id' => 'msg_01HZX876543210ABCDEFGHJKMN',
            'recipient' => 'alice@example.com',
            'timestamp' => '2026-10-29T23:00:00Z',
        ],
    ]);

    $t = time();
    $expectedSig = hash_hmac('sha256', "{$t}.{$payload}", $secret);
    $sigHeader = "t={$t},v1={$expectedSig}";

    // Verify valid signature
    $valid = Webhook::verifySignature($payload, $sigHeader, $secret);
    assert($valid === true);

    // Parse event
    $event = Webhook::constructEvent($payload, $sigHeader, $secret);
    assert($event->getType() === 'message.delivered');
    assert($event->isDelivered() === true);
    assert($event->getMessageId() === 'msg_01HZX876543210ABCDEFGHJKMN');
    assert($event->getRecipient() === 'alice@example.com');

    // Tampered payload throws exception
    $tamperedPayload = json_encode(['id' => 'evt_tampered']);
    $threwTamper = false;
    try {
        Webhook::verifySignature($tamperedPayload, $sigHeader, $secret);
    } catch (WebhookVerificationException $e) {
        $threwTamper = true;
    }
    assert($threwTamper === true, 'Tampered payload must throw WebhookVerificationException');

    // Expired timestamp outside tolerance window throws exception
    $oldTimestamp = time() - 600; // 10 minutes ago
    $oldSig = hash_hmac('sha256', "{$oldTimestamp}.{$payload}", $secret);
    $oldSigHeader = "t={$oldTimestamp},v1={$oldSig}";
    $threwExpired = false;
    try {
        Webhook::verifySignature($payload, $oldSigHeader, $secret, 300);
    } catch (WebhookVerificationException $e) {
        $threwExpired = true;
    }
    assert($threwExpired === true, 'Expired timestamp must throw WebhookVerificationException');
});

echo "\nAll Mailvex SDK tests passed successfully!\n";
