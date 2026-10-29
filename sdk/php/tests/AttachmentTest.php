<?php

declare(strict_types=1);

namespace Mailvex\Tests;

use Mailvex\Models\Attachment;
use PHPUnit\Framework\TestCase;

class AttachmentTest extends TestCase
{
    public function testFromData(): void
    {
        $attachment = Attachment::fromData('pdf binary content', 'invoice.pdf', 'application/pdf', 'inv_1');
        $data = $attachment->toArray();

        $this->assertSame('invoice.pdf', $data['filename']);
        $this->assertSame('application/pdf', $data['content_type']);
        $this->assertSame(base64_encode('pdf binary content'), $data['content']);
        $this->assertSame('inv_1', $data['content_id']);
    }

    public function testFromBase64(): void
    {
        $b64 = base64_encode('image data');
        $attachment = Attachment::fromBase64($b64, 'photo.png', 'image/png');
        $data = $attachment->toArray();

        $this->assertSame('photo.png', $data['filename']);
        $this->assertSame('image/png', $data['content_type']);
        $this->assertSame($b64, $data['content']);
        $this->assertArrayNotHasKey('content_id', $data);
    }

    public function testFromPath(): void
    {
        $tmp = tempnam(sys_get_temp_dir(), 'mvx_');
        file_put_contents($tmp, 'sample file text');

        $attachment = Attachment::fromPath($tmp, 'sample.txt', 'text/plain');
        $data = $attachment->toArray();

        $this->assertSame('sample.txt', $data['filename']);
        $this->assertSame('text/plain', $data['content_type']);
        $this->assertSame(base64_encode('sample file text'), $data['content']);

        unlink($tmp);
    }
}
