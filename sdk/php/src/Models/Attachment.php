<?php

declare(strict_types=1);

namespace Mailvex\Models;

use InvalidArgumentException;

class Attachment
{
    protected string $filename;
    protected string $contentType;
    protected string $content; // Base64 encoded
    protected ?string $contentId = null;

    public function __construct(
        string $filename,
        string $contentType,
        string $contentBase64,
        ?string $contentId = null
    ) {
        $this->filename = $filename;
        $this->contentType = $contentType;
        $this->content = $contentBase64;
        $this->contentId = $contentId;
    }

    /**
     * Create an attachment from a local file path.
     */
    public static function fromPath(
        string $filePath,
        ?string $filename = null,
        ?string $contentType = null,
        ?string $contentId = null
    ): self {
        if (!file_exists($filePath) || !is_readable($filePath)) {
            throw new InvalidArgumentException("Attachment file not found or not readable: {$filePath}");
        }

        $rawContent = file_get_contents($filePath);
        if ($rawContent === false) {
            throw new InvalidArgumentException("Unable to read attachment file: {$filePath}");
        }

        $resolvedFilename = $filename ?? basename($filePath);
        $resolvedContentType = $contentType;

        if ($resolvedContentType === null) {
            if (function_exists('mime_content_type')) {
                $detected = @mime_content_type($filePath);
                if ($detected) {
                    $resolvedContentType = $detected;
                }
            }
            if (!$resolvedContentType) {
                $resolvedContentType = 'application/octet-stream';
            }
        }

        return new self(
            $resolvedFilename,
            $resolvedContentType,
            base64_encode($rawContent),
            $contentId
        );
    }

    /**
     * Create an attachment from raw binary or string data.
     */
    public static function fromData(
        string $data,
        string $filename,
        string $contentType = 'application/octet-stream',
        ?string $contentId = null
    ): self {
        return new self(
            $filename,
            $contentType,
            base64_encode($data),
            $contentId
        );
    }

    /**
     * Create an attachment from pre-encoded Base64 data.
     */
    public static function fromBase64(
        string $base64Content,
        string $filename,
        string $contentType = 'application/octet-stream',
        ?string $contentId = null
    ): self {
        return new self(
            $filename,
            $contentType,
            $base64Content,
            $contentId
        );
    }

    public function getFilename(): string
    {
        return $this->filename;
    }

    public function getContentType(): string
    {
        return $this->contentType;
    }

    public function getContent(): string
    {
        return $this->content;
    }

    public function getContentId(): ?string
    {
        return $this->contentId;
    }

    public function toArray(): array
    {
        $arr = [
            'filename' => $this->filename,
            'content_type' => $this->contentType,
            'content' => $this->content,
        ];

        if ($this->contentId !== null) {
            $arr['content_id'] = $this->contentId;
        }

        return $arr;
    }
}
