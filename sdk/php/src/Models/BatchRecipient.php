<?php

declare(strict_types=1);

namespace Mailvex\Models;

class BatchRecipient
{
    protected string $to;
    protected array $templateVariables = [];
    protected array $metadata = [];

    public function __construct(string $to, array $templateVariables = [], array $metadata = [])
    {
        $this->to = $to;
        $this->templateVariables = $templateVariables;
        $this->metadata = $metadata;
    }

    public static function make(string $to, array $templateVariables = [], array $metadata = []): self
    {
        return new self($to, $templateVariables, $metadata);
    }

    public function getTo(): string
    {
        return $this->to;
    }

    public function setTo(string $to): self
    {
        $this->to = $to;
        return $this;
    }

    public function getTemplateVariables(): array
    {
        return $this->templateVariables;
    }

    public function setTemplateVariables(array $variables): self
    {
        $this->templateVariables = $variables;
        return $this;
    }

    public function withVariable(string $key, mixed $value): self
    {
        $this->templateVariables[$key] = $value;
        return $this;
    }

    public function getMetadata(): array
    {
        return $this->metadata;
    }

    public function setMetadata(array $metadata): self
    {
        $this->metadata = $metadata;
        return $this;
    }

    public function withMetadata(string $key, string $value): self
    {
        $this->metadata[$key] = $value;
        return $this;
    }

    public function toArray(): array
    {
        $data = [
            'to' => $this->to,
        ];

        if (!empty($this->templateVariables)) {
            $data['template_variables'] = $this->templateVariables;
        }

        if (!empty($this->metadata)) {
            $data['metadata'] = $this->metadata;
        }

        return $data;
    }
}
