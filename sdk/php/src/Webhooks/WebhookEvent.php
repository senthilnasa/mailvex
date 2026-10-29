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

    /**
     * Unique Event ULID (e.g. evt_01HZX876543210ABCDEFGHJKMN).
     * Stable across delivery retries; use for de-duplication.
     */
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

    /**
     * Get the target message ID.
     */
    public function getMessageId(): ?string
    {
        return $this->data['message_id'] ?? $this->data['id'] ?? null;
    }

    /**
     * Get the recipient email address.
     */
    public function getRecipient(): ?string
    {
        return $this->data['recipient'] ?? $this->data['to'] ?? $this->data['address'] ?? null;
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

    /**
     * Get bounce classification type: "hard_bounce", "soft_bounce", "rate_limited",
     * "mailbox_unavailable", "spam_rejection", etc.
     */
    public function getBounceType(): ?string
    {
        return $this->data['bounce_type'] ?? $this->data['classification'] ?? null;
    }

    /**
     * Whether this is an unrecoverable hard bounce (e.g. invalid mailbox or bad domain).
     * Addresses should be blocked / suppressed.
     */
    public function isHardBounce(): bool
    {
        if (!$this->isBounced()) {
            return false;
        }

        $type = strtolower((string)$this->getBounceType());
        return in_array($type, ['hard', 'hard_bounce', 'permanent', 'suppress_immediate'], true);
    }

    /**
     * Whether this is a transient soft bounce (e.g. mailbox full, server rate limited).
     * Mailvex retries these automatically; do NOT block address immediately.
     */
    public function isSoftBounce(): bool
    {
        if (!$this->isBounced()) {
            return false;
        }

        $type = strtolower((string)$this->getBounceType());
        return in_array($type, ['soft', 'soft_bounce', 'transient', 'rate_limited', 'mailbox_unavailable'], true);
    }

    /**
     * Human-readable bounce reason or SMTP response text.
     */
    public function getBounceReason(): ?string
    {
        return $this->data['reason'] ?? $this->data['response'] ?? null;
    }

    /**
     * Complaint feedback type (e.g. "abuse", "spam", "fraud", "misc").
     */
    public function getFeedbackType(): ?string
    {
        return $this->data['feedback_type'] ?? $this->data['complaint_type'] ?? null;
    }

    /**
     * Raw SMTP status code (e.g. "250", "550", "421").
     */
    public function getSMTPCode(): ?string
    {
        return isset($this->data['smtp_code']) ? (string)$this->data['smtp_code'] : null;
    }

    /**
     * Enhanced status code (e.g. "2.0.0", "5.1.1", "4.2.2").
     */
    public function getEnhancedCode(): ?string
    {
        return isset($this->data['enhanced_code']) ? (string)$this->data['enhanced_code'] : null;
    }

    /**
     * Target URL for clicked events.
     */
    public function getClickUrl(): ?string
    {
        return $this->data['url'] ?? $this->data['link_url'] ?? null;
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
