<?php

declare(strict_types=1);

namespace Mailvex\Laravel\Transport;

use Mailvex\Client;
use Symfony\Component\Mailer\Envelope;
use Symfony\Component\Mailer\SentMessage;
use Symfony\Component\Mailer\Transport\AbstractTransport;
use Symfony\Component\Mime\Address;
use Symfony\Component\Mime\Email;
use Symfony\Component\Mime\MessageConverter;
use Symfony\Component\Mime\Part\DataPart;

class MailvexTransport extends AbstractTransport
{
    protected Client $client;

    public function __construct(Client $client)
    {
        parent::__construct();
        $this->client = $client;
    }

    public function __toString(): string
    {
        return 'mailvex';
    }

    protected function doSend(SentMessage $message): void
    {
        $rawMessage = $message->getOriginalMessage();
        $email = MessageConverter::toEmail($rawMessage);

        $payload = $this->buildPayload($email, $message->getEnvelope());

        $response = $this->client->send($payload);

        if (isset($response['id'])) {
            $rawMessage->getHeaders()->addHeader('X-Mailvex-Message-ID', $response['id']);
        }
    }

    protected function buildPayload(Email $email, Envelope $envelope): array
    {
        $from = $this->formatAddresses($email->getFrom());
        $fromAddress = !empty($from) ? $from[0] : '';

        $to = $this->formatAddresses($email->getTo());
        if (empty($to)) {
            $to = array_map(fn (Address $addr) => $addr->getAddress(), $envelope->getRecipients());
        }

        $cc = $this->formatAddresses($email->getCc());
        $bcc = $this->formatAddresses($email->getBcc());
        $replyTo = $this->formatAddresses($email->getReplyTo());

        $payload = [
            'from' => $fromAddress,
            'to' => $to,
            'subject' => $email->getSubject() ?? '',
            'stream' => 'transactional',
        ];

        if (!empty($cc)) {
            $payload['cc'] = $cc;
        }

        if (!empty($bcc)) {
            $payload['bcc'] = $bcc;
        }

        if (!empty($replyTo)) {
            $payload['reply_to'] = $replyTo[0];
        }

        if ($email->getTextBody() !== null) {
            $payload['text'] = $email->getTextBody();
        }

        if ($email->getHtmlBody() !== null) {
            $payload['html'] = $email->getHtmlBody();
        }

        // Attachments
        $attachments = [];
        foreach ($email->getAttachments() as $attachment) {
            if ($attachment instanceof DataPart) {
                $item = [
                    'filename' => $attachment->getFilename() ?? 'attachment',
                    'content_type' => $attachment->getMediaType() . '/' . $attachment->getMediaSubtype(),
                    'content' => base64_encode($attachment->getBody()),
                ];

                $cid = $attachment->getContentId();
                if ($cid) {
                    $item['content_id'] = trim($cid, '<>');
                }

                $attachments[] = $item;
            }
        }

        if (!empty($attachments)) {
            $payload['attachments'] = $attachments;
        }

        // Custom headers & Mailvex overrides
        $customHeaders = [];
        $tags = [];
        $metadata = [];

        foreach ($email->getHeaders()->all() as $header) {
            $name = $header->getName();
            $lower = strtolower($name);

            if ($lower === 'x-mailvex-stream') {
                $payload['stream'] = $header->getBodyAsString();
            } elseif ($lower === 'x-mailvex-template-id') {
                $payload['template_id'] = $header->getBodyAsString();
            } elseif ($lower === 'x-mailvex-template-vars') {
                $decoded = json_decode($header->getBodyAsString(), true);
                if (is_array($decoded)) {
                    $payload['template_variables'] = $decoded;
                }
            } elseif ($lower === 'x-mailvex-tags') {
                $split = explode(',', $header->getBodyAsString());
                $tags = array_merge($tags, array_map('trim', $split));
            } elseif (str_starts_with($lower, 'x-metadata-')) {
                $metaKey = substr($lower, 11);
                $metadata[$metaKey] = $header->getBodyAsString();
            } elseif (str_starts_with($lower, 'x-') || $lower === 'list-id' || $lower === 'precedence') {
                $customHeaders[$name] = $header->getBodyAsString();
            }
        }

        if (!empty($customHeaders)) {
            $payload['headers'] = $customHeaders;
        }

        if (!empty($tags)) {
            $payload['tags'] = array_values(array_unique($tags));
        }

        if (!empty($metadata)) {
            $payload['metadata'] = $metadata;
        }

        return $payload;
    }

    /**
     * @param Address[] $addresses
     * @return string[]
     */
    protected function formatAddresses(array $addresses): array
    {
        $formatted = [];
        foreach ($addresses as $addr) {
            $name = $addr->getName();
            if ($name !== '') {
                $formatted[] = "{$name} <{$addr->getAddress()}>";
            } else {
                $formatted[] = $addr->getAddress();
            }
        }
        return $formatted;
    }
}
