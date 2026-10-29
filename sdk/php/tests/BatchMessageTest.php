<?php

declare(strict_types=1);

namespace Mailvex\Tests;

use Mailvex\Models\BatchMessage;
use Mailvex\Models\BatchRecipient;
use PHPUnit\Framework\TestCase;

class BatchMessageTest extends TestCase
{
    public function testBatchMessageCreation(): void
    {
        $batch = BatchMessage::create()
            ->from('news@example.com')
            ->subject('Newsletter')
            ->html('<p>Hello {{name}}</p>')
            ->template('tmpl_news')
            ->addRecipient('user1@example.com', ['name' => 'Alice'])
            ->addRecipient(new BatchRecipient('user2@example.com', ['name' => 'Bob'], ['group' => 'beta']))
            ->recipients(['user3@example.com']);

        $payload = $batch->toArray();

        $this->assertSame('news@example.com', $payload['from']);
        $this->assertSame('Newsletter', $payload['subject']);
        $this->assertCount(3, $payload['recipients']);
        $this->assertSame('user1@example.com', $payload['recipients'][0]['to']);
        $this->assertSame('Alice', $payload['recipients'][0]['template_variables']['name']);
        $this->assertSame('user2@example.com', $payload['recipients'][1]['to']);
        $this->assertSame('beta', $payload['recipients'][1]['metadata']['group']);
        $this->assertSame('user3@example.com', $payload['recipients'][2]['to']);
    }

    public function testBatchRequiresRecipients(): void
    {
        $this->expectException(\InvalidArgumentException::class);
        $batch = BatchMessage::create()->from('a@b.com')->subject('test');
        $batch->toArray();
    }
}
