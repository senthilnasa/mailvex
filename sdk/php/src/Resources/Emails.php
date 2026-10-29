<?php

declare(strict_types=1);

namespace Mailvex\Resources;

use Mailvex\Models\BatchMessage;
use Mailvex\Models\EmailMessage;

class Emails extends AbstractResource
{
    /**
     * Send a single transactional or broadcast email.
     *
     * @param EmailMessage|array $message
     * @param string|null $idempotencyKey
     * @return array
     */
    public function send(EmailMessage|array $message, ?string $idempotencyKey = null): array
    {
        $headers = [];
        $payload = [];

        if ($message instanceof EmailMessage) {
            $payload = $message->toArray();
            $key = $idempotencyKey ?? $message->getIdempotencyKey();
            if ($key) {
                $headers['Idempotency-Key'] = $key;
            }
        } else {
            $payload = $message;
            if ($idempotencyKey) {
                $headers['Idempotency-Key'] = $idempotencyKey;
            }
        }

        return $this->client->request('POST', '/v1/email', $payload, $headers);
    }

    /**
     * Send batch emails to multiple recipients in a single API call (up to 500 recipients).
     *
     * @param BatchMessage|array $batch
     * @param string|null $idempotencyKey
     * @return array
     */
    public function sendBatch(BatchMessage|array $batch, ?string $idempotencyKey = null): array
    {
        $headers = [];
        $payload = [];

        if ($batch instanceof BatchMessage) {
            $payload = $batch->toArray();
            $key = $idempotencyKey ?? $batch->getIdempotencyKey();
            if ($key) {
                $headers['Idempotency-Key'] = $key;
            }
        } else {
            $payload = $batch;
            if ($idempotencyKey) {
                $headers['Idempotency-Key'] = $idempotencyKey;
            }
        }

        return $this->client->request('POST', '/v1/email/batch', $payload, $headers);
    }
}
