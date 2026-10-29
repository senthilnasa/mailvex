<?php

declare(strict_types=1);

namespace Mailvex;

use InvalidArgumentException;

class Config
{
    public const DEFAULT_BASE_URL = 'http://localhost:8080';

    protected string $apiKey;
    protected string $baseUrl;
    protected ?string $tenantId;
    protected ?string $webhookSecret;
    protected int $timeout;
    protected int $maxRetries;

    public function __construct(
        string $apiKey,
        string $baseUrl = self::DEFAULT_BASE_URL,
        ?string $tenantId = null,
        ?string $webhookSecret = null,
        int $timeout = 30,
        int $maxRetries = 3
    ) {
        $trimmedKey = trim($apiKey);
        if ($trimmedKey === '') {
            throw new InvalidArgumentException('Mailvex API key is required. Pass a valid API key or set MAILVEX_API_KEY.');
        }

        $trimmedUrl = trim($baseUrl);
        if ($trimmedUrl === '') {
            throw new InvalidArgumentException('Mailvex API base URL is required. Set MAILVEX_API_URL in your environment (e.g. https://api.mailvex.yourdomain.com).');
        }

        $this->apiKey = $trimmedKey;
        $this->baseUrl = rtrim($trimmedUrl, '/');
        $this->tenantId = $tenantId;
        $this->webhookSecret = $webhookSecret;
        $this->timeout = $timeout;
        $this->maxRetries = $maxRetries;
    }

    public function getApiKey(): string
    {
        return $this->apiKey;
    }

    public function getBaseUrl(): string
    {
        return $this->baseUrl;
    }

    public function getTenantId(): ?string
    {
        return $this->tenantId;
    }

    public function getWebhookSecret(): ?string
    {
        return $this->webhookSecret;
    }

    public function getTimeout(): int
    {
        return $this->timeout;
    }

    public function getMaxRetries(): int
    {
        return $this->maxRetries;
    }
}
