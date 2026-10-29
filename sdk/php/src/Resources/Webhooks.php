<?php

declare(strict_types=1);

namespace Mailvex\Resources;

class Webhooks extends AbstractResource
{
    public function list(): array
    {
        return $this->client->request('GET', '/v1/webhooks');
    }

    public function create(string $url, array $events, ?string $description = null): array
    {
        $payload = [
            'url' => $url,
            'events' => $events,
        ];
        if ($description !== null) {
            $payload['description'] = $description;
        }

        return $this->client->request('POST', '/v1/webhooks', $payload);
    }

    public function delete(string $webhookId): array
    {
        return $this->client->request('DELETE', "/v1/webhooks/{$webhookId}");
    }
}
