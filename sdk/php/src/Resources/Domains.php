<?php

declare(strict_types=1);

namespace Mailvex\Resources;

class Domains extends AbstractResource
{
    public function list(): array
    {
        return $this->client->request('GET', '/v1/domains');
    }

    public function create(string $name, string $type = 'public'): array
    {
        return $this->client->request('POST', '/v1/domains', [
            'name' => $name,
            'type' => $type,
        ]);
    }

    public function get(string $domainId): array
    {
        return $this->client->request('GET', "/v1/domains/{$domainId}");
    }

    public function verify(string $domainId): array
    {
        return $this->client->request('POST', "/v1/domains/{$domainId}/verify");
    }

    public function rotateDkim(string $domainId): array
    {
        return $this->client->request('POST', "/v1/domains/{$domainId}/dkim/rotate");
    }
}
