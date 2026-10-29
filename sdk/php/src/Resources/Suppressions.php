<?php

declare(strict_types=1);

namespace Mailvex\Resources;

class Suppressions extends AbstractResource
{
    public function list(array $query = []): array
    {
        $queryString = !empty($query) ? '?' . http_build_query($query) : '';
        return $this->client->request('GET', "/v1/suppressions{$queryString}");
    }

    public function create(string $address, string $type = 'manual', ?string $reason = null): array
    {
        $payload = [
            'address' => $address,
            'type' => $type,
        ];
        if ($reason !== null) {
            $payload['reason'] = $reason;
        }

        return $this->client->request('POST', '/v1/suppressions', $payload);
    }

    public function delete(string $suppressionId): array
    {
        return $this->client->request('DELETE', "/v1/suppressions/{$suppressionId}");
    }
}
