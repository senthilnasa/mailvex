<?php

declare(strict_types=1);

namespace Mailvex\Models;

use InvalidArgumentException;

class BatchMessage
{
    protected string $from = '';
    protected string $subject = '';
    protected ?string $html = null;
    protected ?string $text = null;
    protected ?string $templateId = null;
    protected array $templateVariables = [];
    protected string $stream = 'broadcast';
    /** @var array<BatchRecipient|array> */
    protected array $recipients = [];
    protected array $attachments = [];
    protected array $headers = [];
    protected array $tags = [];
    protected array $metadata = [];
    protected ?string $idempotencyKey = null;

    public static function create(): self
    {
        return new self();
    }

    public function from(string $from): self
    {
        $this->from = $from;
        return $this;
    }

    public function subject(string $subject): self
    {
        $this->subject = $subject;
        return $this;
    }

    public function html(string $html): self
    {
        $this->html = $html;
        return $this;
    }

    public function text(string $text): self
    {
        $this->text = $text;
        return $this;
    }

    public function stream(string $stream): self
    {
        $this->stream = $stream;
        return $this;
    }

    public function template(string $templateId, array $defaultVariables = []): self
    {
        $this->templateId = $templateId;
        $this->templateVariables = $defaultVariables;
        return $this;
    }

    /**
     * Add a single recipient to the batch.
     * Can be a BatchRecipient instance, or an email string with optional template variables and metadata.
     */
    public function addRecipient(
        BatchRecipient|string $recipient,
        array $templateVariables = [],
        array $metadata = []
    ): self {
        if ($recipient instanceof BatchRecipient) {
            $this->recipients[] = $recipient;
        } else {
            $this->recipients[] = new BatchRecipient($recipient, $templateVariables, $metadata);
        }
        return $this;
    }

    /**
     * Set or add multiple recipients at once.
     * Accepts an array of email strings, or array of associative arrays, or array of BatchRecipient instances.
     */
    public function recipients(array $recipients): self
    {
        foreach ($recipients as $item) {
            if ($item instanceof BatchRecipient) {
                $this->recipients[] = $item;
            } elseif (is_string($item)) {
                $this->recipients[] = new BatchRecipient($item);
            } elseif (is_array($item) && isset($item['to'])) {
                $this->recipients[] = new BatchRecipient(
                    $item['to'],
                    $item['template_variables'] ?? $item['vars'] ?? [],
                    $item['metadata'] ?? []
                );
            }
        }
        return $this;
    }

    public function attach(Attachment|array $attachment): self
    {
        if ($attachment instanceof Attachment) {
            $this->attachments[] = $attachment->toArray();
        } elseif (is_array($attachment)) {
            $this->attachments[] = $attachment;
        }
        return $this;
    }

    public function attachFromPath(
        string $filePath,
        ?string $filename = null,
        ?string $contentType = null,
        ?string $contentId = null
    ): self {
        $this->attachments[] = Attachment::fromPath($filePath, $filename, $contentType, $contentId)->toArray();
        return $this;
    }

    public function header(string $key, string $value): self
    {
        $this->headers[$key] = $value;
        return $this;
    }

    public function headers(array $headers): self
    {
        $this->headers = array_merge($this->headers, $headers);
        return $this;
    }

    public function tag(string $tag): self
    {
        if (!in_array($tag, $this->tags, true)) {
            $this->tags[] = $tag;
        }
        return $this;
    }

    public function tags(array $tags): self
    {
        $this->tags = array_unique(array_merge($this->tags, $tags));
        return $this;
    }

    public function metadata(string|array $keyOrMap, ?string $value = null): self
    {
        if (is_array($keyOrMap)) {
            $this->metadata = array_merge($this->metadata, $keyOrMap);
        } elseif ($value !== null) {
            $this->metadata[$keyOrMap] = $value;
        }
        return $this;
    }

    public function idempotencyKey(string $key): self
    {
        $this->idempotencyKey = $key;
        return $this;
    }

    public function getIdempotencyKey(): ?string
    {
        return $this->idempotencyKey;
    }

    public function toArray(): array
    {
        if (empty($this->recipients)) {
            throw new InvalidArgumentException('Batch email requires at least one recipient.');
        }

        if (count($this->recipients) > 500) {
            throw new InvalidArgumentException('Batch email exceeds maximum limit of 500 recipients per call.');
        }

        $formattedRecipients = [];
        foreach ($this->recipients as $recipient) {
            $formattedRecipients[] = $recipient instanceof BatchRecipient ? $recipient->toArray() : $recipient;
        }

        $payload = [
            'from' => $this->from,
            'subject' => $this->subject,
            'stream' => $this->stream,
            'recipients' => $formattedRecipients,
        ];

        if ($this->html !== null) {
            $payload['html'] = $this->html;
        }

        if ($this->text !== null) {
            $payload['text'] = $this->text;
        }

        if ($this->templateId !== null) {
            $payload['template_id'] = $this->templateId;
        }

        if (!empty($this->templateVariables)) {
            $payload['template_variables'] = $this->templateVariables;
        }

        if (!empty($this->attachments)) {
            $payload['attachments'] = $this->attachments;
        }

        if (!empty($this->headers)) {
            $payload['headers'] = $this->headers;
        }

        if (!empty($this->tags)) {
            $payload['tags'] = $this->tags;
        }

        if (!empty($this->metadata)) {
            $payload['metadata'] = $this->metadata;
        }

        return $payload;
    }
}
