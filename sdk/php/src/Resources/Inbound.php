<?php

declare(strict_types=1);

namespace Mailvex\Resources;

class Inbound extends AbstractResource
{
    public function listRoutes(): array
    {
        return $this->client->request('GET', '/v1/inbound/routes');
    }

    public function createRoute(array $route): array
    {
        return $this->client->request('POST', '/v1/inbound/routes', $route);
    }

    public function testRoute(string $email): array
    {
        return $this->client->request('POST', '/v1/inbound/routes/test', [
            'email' => $email,
        ]);
    }

    public function listMessages(array $query = []): array
    {
        $queryString = !empty($query) ? '?' . http_build_query($query) : '';
        return $this->client->request('GET', "/v1/inbound/messages{$queryString}");
    }
}
