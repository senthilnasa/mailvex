<?php

declare(strict_types=1);

namespace Mailvex;

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
        $this->apiKey = $apiKey;
        $this->baseUrl = rtrim($baseUrl, '/');
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
