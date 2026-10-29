<?php

declare(strict_types=1);

namespace Mailvex\Exceptions;

class ApiException extends MailvexException
{
    protected int $statusCode;
    protected ?string $errorCode;
    protected ?array $responseBody;

    public function __construct(
        string $message,
        int $statusCode = 0,
        ?string $errorCode = null,
        ?array $responseBody = null,
        ?\Throwable $previous = null
    ) {
        parent::__construct($message, $statusCode, $previous);
        $this->statusCode = $statusCode;
        $this->errorCode = $errorCode;
        $this->responseBody = $responseBody;
    }

    public function getStatusCode(): int
    {
        return $this->statusCode;
    }

    public function getErrorCode(): ?string
    {
        return $this->errorCode;
    }

    public function getResponseBody(): ?array
    {
        return $this->responseBody;
    }
}
