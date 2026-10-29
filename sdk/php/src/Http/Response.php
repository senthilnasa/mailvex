<?php

declare(strict_types=1);

namespace Mailvex\Http;

class Response
{
    protected int $statusCode;
    protected array $headers;
    protected string $body;
    protected ?array $jsonDecoded = null;

    public function __construct(int $statusCode, array $headers, string $body)
    {
        $this->statusCode = $statusCode;
        $this->headers = $headers;
        $this->body = $body;
    }

    public function getStatusCode(): int
    {
        return $this->statusCode;
    }

    public function getHeaders(): array
    {
        return $this->headers;
    }

    public function getHeader(string $name): ?string
    {
        $nameLower = strtolower($name);
        foreach ($this->headers as $key => $value) {
            if (strtolower($key) === $nameLower) {
                return is_array($value) ? implode(', ', $value) : (string)$value;
            }
        }
        return null;
    }

    public function getBody(): string
    {
        return $this->body;
    }

    public function json(): ?array
    {
        if ($this->jsonDecoded !== null) {
            return $this->jsonDecoded;
        }

        if (trim($this->body) === '') {
            return null;
        }

        $decoded = json_decode($this->body, true);
        if (json_last_error() === JSON_ERROR_NONE && is_array($decoded)) {
            $this->jsonDecoded = $decoded;
            return $this->jsonDecoded;
        }

        return null;
    }

    public function isSuccessful(): bool
    {
        return $this->statusCode >= 200 && $this->statusCode < 300;
    }
}
