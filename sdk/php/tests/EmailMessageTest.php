<?php

declare(strict_types=1);

namespace Mailvex\Tests;

use Mailvex\Models\EmailMessage;
use PHPUnit\Framework\TestCase;

class EmailMessageTest extends TestCase
{
    public function testFluentBuilderWithAttachmentsAndMultipleRecipients(): void
    {
        $message = EmailMessage::create()
            ->from('Alerts <alerts@example.com>')
            ->to(['alice@example.com', 'bob@example.com'])
            ->cc(['carol@example.com'])
            ->bcc(['audit@example.com'])
            ->replyTo('support@example.com')
            ->subject('Security Notice')
            ->html('<h1>Notice</h1>')
            ->text('Notice text')
            ->stream('transactional')
            ->attachFromData('file-content', 'doc.txt', 'text/plain')
            ->header('X-App-Env', 'production')
            ->tag('security')
            ->metadata('user_id', '123')
            ->trackOpens(true)
            ->trackClicks(false)
            ->idempotencyKey('idem_key_123');

        $arr = $message->toArray();

        $this->assertSame('Alerts <alerts@example.com>', $arr['from']);
        $this->assertSame(['alice@example.com', 'bob@example.com'], $arr['to']);
        $this->assertSame(['carol@example.com'], $arr['cc']);
        $this->assertSame(['audit@example.com'], $arr['bcc']);
        $this->assertSame('support@example.com', $arr['reply_to']);
        $this->assertSame('Security Notice', $arr['subject']);
        $this->assertSame('<h1>Notice</h1>', $arr['html']);
        $this->assertSame('Notice text', $arr['text']);
        $this->assertSame('transactional', $arr['stream']);
        $this->assertCount(1, $arr['attachments']);
        $this->assertSame('doc.txt', $arr['attachments'][0]['filename']);
        $this->assertSame('production', $arr['headers']['X-App-Env']);
        $this->assertSame(['security'], $arr['tags']);
        $this->assertSame(['user_id' => '123'], $arr['metadata']);
        $this->assertTrue($arr['track_opens']);
        $this->assertFalse($arr['track_clicks']);
        $this->assertSame('idem_key_123', $message->getIdempotencyKey());
    }
}
