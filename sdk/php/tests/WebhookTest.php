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

    public function testBounceClassificationHelpers(): void
    {
        $payload = json_encode([
            'id' => 'evt_01HZX_HARD',
            'type' => 'message.bounced',
            'data' => [
                'message_id' => 'msg_456',
                'recipient' => 'nobody@example.com',
                'classification' => 'hard_bounce',
                'reason' => '550 5.1.1 User unknown',
                'smtp_code' => '550',
                'enhanced_code' => '5.1.1',
            ],
        ]);

        $t = time();
        $v1 = hash_hmac('sha256', "{$t}.{$payload}", $this->secret);
        $header = "t={$t},v1={$v1}";

        $event = Webhook::constructEvent($payload, $header, $this->secret);
        $this->assertTrue($event->isBounced());
        $this->assertTrue($event->isHardBounce());
        $this->assertFalse($event->isSoftBounce());
        $this->assertSame('hard_bounce', $event->getBounceType());
        $this->assertSame('550 5.1.1 User unknown', $event->getBounceReason());
        $this->assertSame('550', $event->getSMTPCode());
        $this->assertSame('5.1.1', $event->getEnhancedCode());
    }
}
