<?php

declare(strict_types=1);

namespace Mailvex\Webhooks;

use Mailvex\Exceptions\WebhookVerificationException;

class Webhook
{
    public const DEFAULT_TOLERANCE = 300; // 5 minutes

    /**
     * Verify the X-Mailvex-Signature header.
     *
     * @param string $payload Raw request body string.
     * @param string $signatureHeader Header value (e.g. "t=1730107200,v1=3c6f8b9e...")
     * @param string $secret Signing secret (prefixed with whsec_).
     * @param int $tolerance Maximum allowed clock drift in seconds (0 to disable).
     * @return bool
     * @throws WebhookVerificationException
     */
    public static function verifySignature(
        string $payload,
        string $signatureHeader,
        string $secret,
        int $tolerance = self::DEFAULT_TOLERANCE
    ): bool {
        if (empty($signatureHeader)) {
            throw new WebhookVerificationException('Missing X-Mailvex-Signature header.');
        }

        if (empty($secret)) {
            throw new WebhookVerificationException('Webhook signing secret is required.');
        }

        $items = explode(',', $signatureHeader);
        $timestamp = null;
        $signatures = [];

        foreach ($items as $item) {
            $parts = explode('=', trim($item), 2);
            if (count($parts) === 2) {
                $key = trim($parts[0]);
                $val = trim($parts[1]);
                if ($key === 't') {
                    $timestamp = (int)$val;
                } elseif ($key === 'v1') {
                    $signatures[] = $val;
                }
            }
        }

        if ($timestamp === null || empty($signatures)) {
            throw new WebhookVerificationException('Invalid X-Mailvex-Signature format. Expected "t=<timestamp>,v1=<signature>".');
        }

        // Check timestamp drift
        if ($tolerance > 0) {
            $current = time();
            if (abs($current - $timestamp) > $tolerance) {
                throw new WebhookVerificationException(
                    "Webhook timestamp [{$timestamp}] falls outside the tolerance zone [{$tolerance}s] relative to server time [{$current}]."
                );
            }
        }

        $signedPayload = "{$timestamp}.{$payload}";
        $expectedSignature = hash_hmac('sha256', $signedPayload, $secret);

        $valid = false;
        foreach ($signatures as $sig) {
            if (hash_equals($expectedSignature, $sig)) {
                $valid = true;
                break;
            }
        }

        if (!$valid) {
            throw new WebhookVerificationException('Signature mismatch: payload hash does not match provided v1 signature.');
        }

        return true;
    }

    /**
     * Verify the signature and construct a WebhookEvent object.
     *
     * @throws WebhookVerificationException
     */
    public static function constructEvent(
        string $payload,
        string $signatureHeader,
        string $secret,
        int $tolerance = self::DEFAULT_TOLERANCE
    ): WebhookEvent {
        self::verifySignature($payload, $signatureHeader, $secret, $tolerance);

        $decoded = json_decode($payload, true);
        if (!is_array($decoded)) {
            throw new WebhookVerificationException('Invalid webhook JSON payload.');
        }

        $type = $decoded['type'] ?? $decoded['event'] ?? 'unknown';
        $data = $decoded['data'] ?? $decoded;
        $id = $decoded['id'] ?? null;
        $timestamp = isset($decoded['timestamp']) ? (int)$decoded['timestamp'] : null;

        return new WebhookEvent($type, $data, $id, $timestamp, $payload);
    }
}
