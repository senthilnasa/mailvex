<?php

declare(strict_types=1);

namespace Mailvex;

use Mailvex\Exceptions\ApiException;
use Mailvex\Exceptions\AuthenticationException;
use Mailvex\Exceptions\NetworkException;
use Mailvex\Exceptions\RateLimitException;
use Mailvex\Exceptions\ValidationException;
use Mailvex\Http\CurlHttpClient;
use Mailvex\Http\HttpClientInterface;
use Mailvex\Models\BatchMessage;
use Mailvex\Models\EmailMessage;
use Mailvex\Resources\Domains;
use Mailvex\Resources\Emails;
use Mailvex\Resources\Inbound;
use Mailvex\Resources\Messages;
use Mailvex\Resources\Reputation;
use Mailvex\Resources\Suppressions;
use Mailvex\Resources\Webhooks;
use Mailvex\Webhooks\Webhook;
use Mailvex\Webhooks\WebhookEvent;

class Client
{
    public const VERSION = '1.0.0';

    protected Config $config;
    protected HttpClientInterface $httpClient;

    protected ?Emails $emails = null;
    protected ?Messages $messages = null;
    protected ?Domains $domains = null;
    protected ?Suppressions $suppressions = null;
    protected ?Webhooks $webhooks = null;
    protected ?Inbound $inbound = null;
    protected ?Reputation $reputation = null;

    public function __construct(Config|string $configOrApiKey, ?HttpClientInterface $httpClient = null)
    {
        if (is_string($configOrApiKey)) {
            $this->config = new Config($configOrApiKey);
        } else {
            $this->config = $configOrApiKey;
        }

        $this->httpClient = $httpClient ?? new CurlHttpClient();
    }

    public function getConfig(): Config
    {
        return $this->config;
    }

    public function emails(): Emails
    {
        if ($this->emails === null) {
            $this->emails = new Emails($this);
        }
        return $this->emails;
    }

    public function messages(): Messages
    {
        if ($this->messages === null) {
            $this->messages = new Messages($this);
        }
        return $this->messages;
    }

    public function domains(): Domains
    {
        if ($this->domains === null) {
            $this->domains = new Domains($this);
        }
        return $this->domains;
    }

    public function suppressions(): Suppressions
    {
        if ($this->suppressions === null) {
            $this->suppressions = new Suppressions($this);
        }
        return $this->suppressions;
    }

    public function webhooks(): Webhooks
    {
        if ($this->webhooks === null) {
            $this->webhooks = new Webhooks($this);
        }
        return $this->webhooks;
    }

    public function inbound(): Inbound
    {
        if ($this->inbound === null) {
            $this->inbound = new Inbound($this);
        }
        return $this->inbound;
    }

    public function reputation(): Reputation
    {
        if ($this->reputation === null) {
            $this->reputation = new Reputation($this);
        }
        return $this->reputation;
    }

    /**
     * Send single email convenience method.
     */
    public function send(EmailMessage|array $message, ?string $idempotencyKey = null): array
    {
        return $this->emails()->send($message, $idempotencyKey);
    }

    /**
     * Send batch email convenience method (multiple emails in one call).
     */
    public function sendBatch(BatchMessage|array $batch, ?string $idempotencyKey = null): array
    {
        return $this->emails()->sendBatch($batch, $idempotencyKey);
    }

    /**
     * Webhook verification helper.
     */
    public function verifyWebhook(string $payload, string $signatureHeader, ?string $secret = null): bool
    {
        $signingSecret = $secret ?? $this->config->getWebhookSecret();
        return Webhook::verifySignature($payload, $signatureHeader, (string)$signingSecret);
    }

    /**
     * Webhook event construction helper.
     */
    public function constructWebhookEvent(string $payload, string $signatureHeader, ?string $secret = null): WebhookEvent
    {
        $signingSecret = $secret ?? $this->config->getWebhookSecret();
        return Webhook::constructEvent($payload, $signatureHeader, (string)$signingSecret);
    }

    /**
     * Core execution engine for API calls with automatic retries and error parsing.
     */
    public function request(string $method, string $path, array $data = [], array $headers = []): array
    {
        $url = $this->config->getBaseUrl() . $path;

        $requestHeaders = array_merge([
            'Authorization' => 'Bearer ' . $this->config->getApiKey(),
            'Content-Type' => 'application/json',
            'Accept' => 'application/json',
            'User-Agent' => 'Mailvex-PHP/' . self::VERSION,
        ], $headers);

        if ($this->config->getTenantId()) {
            $requestHeaders['X-Tenant-ID'] = $this->config->getTenantId();
        }

        $body = null;
        if (!empty($data)) {
            $body = json_encode($data, JSON_UNESCAPED_SLASHES | JSON_UNESCAPED_UNICODE);
        }

        $attempts = 0;
        $maxAttempts = max(1, $this->config->getMaxRetries() + 1);

        while ($attempts < $maxAttempts) {
            $attempts++;
            try {
                $response = $this->httpClient->request(
                    $method,
                    $url,
                    $requestHeaders,
                    $body,
                    $this->config->getTimeout()
                );

                $statusCode = $response->getStatusCode();
                $respData = $response->json() ?? [];

                if ($response->isSuccessful()) {
                    return $respData;
                }

                // Error response parsing
                $errorObj = $respData['error'] ?? [];
                $errorCode = is_array($errorObj) ? ($errorObj['code'] ?? null) : null;
                $errorMessage = is_array($errorObj) ? ($errorObj['message'] ?? $response->getBody()) : $response->getBody();

                if ($statusCode === 429) {
                    $reset = $response->getHeader('RateLimit-Reset');
                    if ($attempts < $maxAttempts) {
                        usleep(500000 * $attempts); // 500ms backoff
                        continue;
                    }
                    throw new RateLimitException(
                        $errorMessage ?: 'Rate limit exceeded',
                        429,
                        $errorCode,
                        $respData,
                        $reset ? (int)$reset : null
                    );
                }

                if ($statusCode === 401 || $statusCode === 403) {
                    throw new AuthenticationException(
                        $errorMessage ?: 'Authentication failed',
                        $statusCode,
                        $errorCode,
                        $respData
                    );
                }

                if ($statusCode === 400 || $statusCode === 422) {
                    throw new ValidationException(
                        $errorMessage ?: 'Request validation error',
                        $statusCode,
                        $errorCode,
                        $respData
                    );
                }

                if ($statusCode >= 500 && $attempts < $maxAttempts) {
                    usleep(300000 * $attempts);
                    continue;
                }

                throw new ApiException(
                    $errorMessage ?: "API request failed with status {$statusCode}",
                    $statusCode,
                    $errorCode,
                    $respData
                );
            } catch (NetworkException $e) {
                if ($attempts < $maxAttempts) {
                    usleep(300000 * $attempts);
                    continue;
                }
                throw $e;
            }
        }

        throw new ApiException("Max retries exceeded for {$method} {$url}");
    }
}
