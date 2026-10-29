<?php

declare(strict_types=1);

namespace Mailvex\Laravel\Http\Controllers;

use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;
use Illuminate\Routing\Controller;
use Mailvex\Exceptions\WebhookVerificationException;
use Mailvex\Laravel\Events\WebhookReceived;
use Mailvex\Webhooks\Webhook;

class WebhookController extends Controller
{
    /**
     * Handle the incoming Mailvex webhook.
     */
    public function __invoke(Request $request): JsonResponse
    {
        $signature = $request->header('X-Mailvex-Signature');
        if (!$signature) {
            return response()->json([
                'error' => [
                    'code' => 'missing_signature',
                    'message' => 'Missing X-Mailvex-Signature header.',
                ],
            ], 400);
        }

        $secret = config('mailvex.webhook_secret');
        if (!$secret) {
            return response()->json([
                'error' => [
                    'code' => 'missing_secret_config',
                    'message' => 'Mailvex webhook secret is not configured in mailvex.webhook_secret.',
                ],
            ], 500);
        }

        $tolerance = (int) config('mailvex.webhook_tolerance', 300);

        try {
            $event = Webhook::constructEvent(
                $request->getContent(),
                $signature,
                $secret,
                $tolerance
            );
        } catch (WebhookVerificationException $e) {
            return response()->json([
                'error' => [
                    'code' => 'invalid_signature',
                    'message' => $e->getMessage(),
                ],
            ], 401);
        }

        // Dispatch the Laravel event
        event(new WebhookReceived($event, $event->getData()));

        return response()->json(['status' => 'success']);
    }
}
