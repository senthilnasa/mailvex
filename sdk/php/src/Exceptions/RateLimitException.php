<?php

declare(strict_types=1);

namespace Mailvex\Exceptions;

class RateLimitException extends ApiException
{
    protected ?int $resetTimestamp;

    public function __construct(
        string $message = 'Rate limit exceeded',
        int $statusCode = 429,
        ?string $errorCode = 'rate_limit_exceeded',
        ?array $responseBody = null,
        ?int $resetTimestamp = null,
        ?\Throwable $previous = null
    ) {
        parent::__construct($message, $statusCode, $errorCode, $responseBody, $previous);
        $this->resetTimestamp = $resetTimestamp;
    }

    public function getResetTimestamp(): ?int
    {
        return $this->resetTimestamp;
    }
}
