<?php

declare(strict_types=1);

namespace Mailvex\Laravel\Http\Middleware;

use Closure;
use Illuminate\Http\Request;
use Mailvex\Exceptions\WebhookVerificationException;
use Mailvex\Webhooks\Webhook;
use Symfony\Component\HttpFoundation\Response;

class VerifyWebhookSignature
{
    /**
     * Handle an incoming request.
     */
    public function handle(Request $request, Closure $next): Response
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
            Webhook::verifySignature(
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

        return $next($request);
    }
}
