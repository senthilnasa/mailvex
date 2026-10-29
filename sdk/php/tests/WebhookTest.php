<?php

declare(strict_types=1);

namespace Mailvex\Tests;

use Mailvex\Exceptions\WebhookVerificationException;
use Mailvex\Webhooks\Webhook;
use PHPUnit\Framework\TestCase;

class WebhookTest extends TestCase
{
    protected string $secret = 'whsec_test_secret_abc123';

    public function testValidSignatureVerification(): void
    {
        $payload = json_encode([
            'id' => 'whk_01HZX',
            'type' => 'message.bounced',
            'data' => [
                'message_id' => 'msg_123',
                'recipient' => 'bad@example.com',
                'reason' => 'mailbox full',
            ],
        ]);

        $t = time();
        $v1 = hash_hmac('sha256', "{$t}.{$payload}", $this->secret);
        $header = "t={$t},v1={$v1}";

        $this->assertTrue(Webhook::verifySignature($payload, $header, $this->secret));

        $event = Webhook::constructEvent($payload, $header, $this->secret);
        $this->assertSame('message.bounced', $event->getType());
        $this->assertTrue($event->isBounced());
        $this->assertFalse($event->isDelivered());
        $this->assertSame('msg_123', $event->getMessageId());
        $this->assertSame('bad@example.com', $event->getRecipient());
    }

    public function testTamperedPayloadThrowsException(): void
    {
        $payload = json_encode(['data' => 'original']);
        $t = time();
        $v1 = hash_hmac('sha256', "{$t}.{$payload}", $this->secret);
        $header = "t={$t},v1={$v1}";

        $tampered = json_encode(['data' => 'tampered']);

        $this->expectException(WebhookVerificationException::class);
        Webhook::verifySignature($tampered, $header, $this->secret);
    }

    public function testMissingHeaderThrowsException(): void
    {
        $this->expectException(WebhookVerificationException::class);
        Webhook::verifySignature('{}', '', $this->secret);
    }
}
