<?php

declare(strict_types=1);

namespace Mailvex\Laravel\Facades;

use Illuminate\Support\Facades\Facade;
use Mailvex\Client;
use Mailvex\Models\BatchMessage;
use Mailvex\Models\EmailMessage;
use Mailvex\Resources\Domains;
use Mailvex\Resources\Emails;
use Mailvex\Resources\Inbound;
use Mailvex\Resources\Messages;
use Mailvex\Resources\Reputation;
use Mailvex\Resources\Suppressions;
use Mailvex\Resources\Webhooks;

/**
 * @method static array send(EmailMessage|array $message, ?string $idempotencyKey = null)
 * @method static array sendBatch(BatchMessage|array $batch, ?string $idempotencyKey = null)
 * @method static Emails emails()
 * @method static Messages messages()
 * @method static Domains domains()
 * @method static Suppressions suppressions()
 * @method static Webhooks webhooks()
 * @method static Inbound inbound()
 * @method static Reputation reputation()
 * @method static bool verifyWebhook(string $payload, string $signatureHeader, ?string $secret = null)
 * @method static \Mailvex\Webhooks\WebhookEvent constructWebhookEvent(string $payload, string $signatureHeader, ?string $secret = null)
 * @method static array request(string $method, string $path, array $data = [], array $headers = [])
 *
 * @see \Mailvex\Client
 */
class Mailvex extends Facade
{
    protected static function getFacadeAccessor(): string
    {
        return 'mailvex';
    }
}
