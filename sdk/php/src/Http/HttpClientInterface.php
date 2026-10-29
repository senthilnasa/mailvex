<?php

declare(strict_types=1);

namespace Mailvex\Http;

interface HttpClientInterface
{
    /**
     * Send an HTTP request and return the Response.
     *
     * @param string $method GET, POST, DELETE, etc.
     * @param string $url Full request URL.
     * @param array<string, string> $headers Request headers.
     * @param string|null $body Request payload body.
     * @param int $timeout Request timeout in seconds.
     * @return Response
     * @throws \Mailvex\Exceptions\NetworkException
     */
    public function request(
        string $method,
        string $url,
        array $headers = [],
        ?string $body = null,
        int $timeout = 30
    ): Response;
}
