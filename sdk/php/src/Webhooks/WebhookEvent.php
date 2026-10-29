<?php

declare(strict_types=1);

namespace Mailvex\Webhooks;

class WebhookEvent
{
    protected ?string $id;
    protected string $type;
    protected array $data;
    protected ?int $timestamp;
    protected string $rawPayload;

    public function __construct(
        string $type,
        array $data,
        ?string $id = null,
        ?int $timestamp = null,
        string $rawPayload = ''
    ) {
        $this->type = $type;
        $this->data = $data;
        $this->id = $id;
        $this->timestamp = $timestamp;
        $this->rawPayload = $rawPayload;
    }

    public function getId(): ?string
    {
        return $this->id;
    }

    public function getType(): string
    {
        return $this->type;
    }

    public function getData(): array
    {
        return $this->data;
    }

    public function getTimestamp(): ?int
    {
        return $this->timestamp;
    }

    public function getRawPayload(): string
    {
        return $this->rawPayload;
    }

    public function getMessageId(): ?string
    {
        return $this->data['message_id'] ?? $this->data['id'] ?? null;
    }

    public function getRecipient(): ?string
    {
        return $this->data['recipient'] ?? $this->data['to'] ?? null;
    }

    public function isDelivered(): bool
    {
        return $this->type === 'message.delivered';
    }

    public function isBounced(): bool
    {
        return $this->type === 'message.bounced';
    }

    public function isComplained(): bool
    {
        return $this->type === 'message.complained';
    }

    public function isOpened(): bool
    {
        return $this->type === 'message.opened';
    }

    public function isClicked(): bool
    {
        return $this->type === 'message.clicked';
    }

    public function toArray(): array
    {
        return [
            'id' => $this->id,
            'type' => $this->type,
            'data' => $this->data,
            'timestamp' => $this->timestamp,
        ];
    }
}
