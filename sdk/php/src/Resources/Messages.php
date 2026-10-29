<?php

declare(strict_types=1);

namespace Mailvex\Resources;

class Messages extends AbstractResource
{
    /**
     * Retrieve details of a specific message.
     */
    public function get(string $messageId): array
    {
        return $this->client->request('GET', "/v1/messages/{$messageId}");
    }

    /**
     * Query messages with optional filters.
     *
     * @param array $query ['status' => ..., 'recipient' => ..., 'from' => ..., 'limit' => ...]
     */
    public function list(array $query = []): array
    {
        $queryString = !empty($query) ? '?' . http_build_query($query) : '';
        return $this->client->request('GET', "/v1/messages{$queryString}");
    }

    /**
     * Get delivery events timeline for a message.
     */
    public function events(string $messageId): array
    {
        return $this->client->request('GET', "/v1/messages/{$messageId}/events");
    }

    /**
     * Cancel a queued message.
     */
    public function cancel(string $messageId): array
    {
        return $this->client->request('POST', "/v1/messages/{$messageId}/cancel");
    }

    /**
     * Retry a failed or bounced message.
     */
    public function retry(string $messageId): array
    {
        return $this->client->request('POST', "/v1/messages/{$messageId}/retry");
    }

    /**
     * Inspect messages captured in the development sink.
     */
    public function development(array $query = []): array
    {
        $queryString = !empty($query) ? '?' . http_build_query($query) : '';
        return $this->client->request('GET', "/v1/messages/development{$queryString}");
    }
}
