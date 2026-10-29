<?php

declare(strict_types=1);

namespace Mailvex\Resources;

class Reputation extends AbstractResource
{
    public function overview(): array
    {
        return $this->client->request('GET', '/v1/reputation/overview');
    }

    public function checkDnsbl(string $ip): array
    {
        return $this->client->request('POST', '/v1/dnsbl/check', [
            'ip' => $ip,
        ]);
    }
}
