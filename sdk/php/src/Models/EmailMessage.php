<?php

declare(strict_types=1);

namespace Mailvex\Models;

class EmailMessage
{
    protected string $from = '';
    protected array $to = [];
    protected array $cc = [];
    protected array $bcc = [];
    protected ?string $replyTo = null;
    protected string $subject = '';
    protected ?string $html = null;
    protected ?string $text = null;
    protected string $stream = 'transactional';
    protected ?string $templateId = null;
    protected array $templateVariables = [];
    protected array $attachments = [];
    protected array $headers = [];
    protected array $tags = [];
    protected array $metadata = [];
    protected ?bool $trackOpens = null;
    protected ?bool $trackClicks = null;
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

    /**
     * Add recipient(s). Can be string or array of strings.
     * Supports multiple recipients in one call.
     */
    public function to(string|array $to): self
    {
        $recipients = is_array($to) ? $to : [$to];
        foreach ($recipients as $recipient) {
            if ($recipient && !in_array($recipient, $this->to, true)) {
                $this->to[] = $recipient;
            }
        }
        return $this;
    }

    public function cc(string|array $cc): self
    {
        $recipients = is_array($cc) ? $cc : [$cc];
        foreach ($recipients as $recipient) {
            if ($recipient && !in_array($recipient, $this->cc, true)) {
                $this->cc[] = $recipient;
            }
        }
        return $this;
    }

    public function bcc(string|array $bcc): self
    {
        $recipients = is_array($bcc) ? $bcc : [$bcc];
        foreach ($recipients as $recipient) {
            if ($recipient && !in_array($recipient, $this->bcc, true)) {
                $this->bcc[] = $recipient;
            }
        }
        return $this;
    }

    public function replyTo(string $replyTo): self
    {
        $this->replyTo = $replyTo;
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

    public function template(string $templateId, array $variables = []): self
    {
        $this->templateId = $templateId;
        $this->templateVariables = $variables;
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

    public function attachFromData(
        string $data,
        string $filename,
        string $contentType = 'application/octet-stream',
        ?string $contentId = null
    ): self {
        $this->attachments[] = Attachment::fromData($data, $filename, $contentType, $contentId)->toArray();
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

    public function trackOpens(bool $track = true): self
    {
        $this->trackOpens = $track;
        return $this;
    }

    public function trackClicks(bool $track = true): self
    {
        $this->trackClicks = $track;
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
        $payload = [
            'from' => $this->from,
            'to' => $this->to,
            'subject' => $this->subject,
            'stream' => $this->stream,
        ];

        if (!empty($this->cc)) {
            $payload['cc'] = $this->cc;
        }

        if (!empty($this->bcc)) {
            $payload['bcc'] = $this->bcc;
        }

        if ($this->replyTo !== null) {
            $payload['reply_to'] = $this->replyTo;
        }

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

        if ($this->trackOpens !== null) {
            $payload['track_opens'] = $this->trackOpens;
        }

        if ($this->trackClicks !== null) {
            $payload['track_clicks'] = $this->trackClicks;
        }

        return $payload;
    }
}
