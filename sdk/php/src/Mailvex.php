<?php

declare(strict_types=1);

namespace Mailvex;

use Mailvex\Models\Attachment;
use Mailvex\Models\BatchMessage;
use Mailvex\Models\EmailMessage;
use Mailvex\Webhooks\Webhook;
use Mailvex\Webhooks\WebhookEvent;

class Mailvex
{
    /**
     * Create a new Mailvex client instance.
     */
    public static function client(
        string $apiKey,
        string $baseUrl = Config::DEFAULT_BASE_URL,
        ?string $tenantId = null,
        ?string $webhookSecret = null
    ): Client {
        $config = new Config($apiKey, $baseUrl, $tenantId, $webhookSecret);
        return new Client($config);
    }

    /**
     * Create a fluent EmailMessage builder.
     */
    public static function email(): EmailMessage
    {
        return new EmailMessage();
    }

    /**
     * Create a fluent BatchMessage builder.
     */
    public static function batch(): BatchMessage
    {
        return new BatchMessage();
    }

    /**
     * Create an attachment from a file path.
     */
    public static function attachmentFromPath(
        string $filePath,
        ?string $filename = null,
        ?string $contentType = null,
        ?string $contentId = null
    ): Attachment {
        return Attachment::fromPath($filePath, $filename, $contentType, $contentId);
    }

    /**
     * Create an attachment from string/binary data.
     */
    public static function attachmentFromData(
        string $data,
        string $filename,
        string $contentType = 'application/octet-stream',
        ?string $contentId = null
    ): Attachment {
        return Attachment::fromData($data, $filename, $contentType, $contentId);
    }

    /**
     * Verify a webhook signature.
     */
    public static function verifyWebhook(
        string $payload,
        string $signatureHeader,
        string $secret,
        int $tolerance = Webhook::DEFAULT_TOLERANCE
    ): bool {
        return Webhook::verifySignature($payload, $signatureHeader, $secret, $tolerance);
    }

    /**
     * Construct a WebhookEvent object from incoming webhook payload.
     */
    public static function constructWebhookEvent(
        string $payload,
        string $signatureHeader,
        string $secret,
        int $tolerance = Webhook::DEFAULT_TOLERANCE
    ): WebhookEvent {
        return Webhook::constructEvent($payload, $signatureHeader, $secret, $tolerance);
    }
}
