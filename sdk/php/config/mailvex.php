<?php

return [
    /*
    |--------------------------------------------------------------------------
    | Mailvex API Key
    |--------------------------------------------------------------------------
    |
    | Your Mailvex API key. Can be a live key (mvx_live_...) or a test key
    | (mvx_test_...) for dry runs and sandbox testing.
    |
    */
    'api_key' => env('MAILVEX_API_KEY', ''),

    /*
    |--------------------------------------------------------------------------
    | Mailvex Base API URL
    |--------------------------------------------------------------------------
    |
    | The base URL of your Mailvex deployment or local development edge.
    | In production, set this explicitly (e.g. https://api.mailvex.yourdomain.com).
    | Defaults to http://localhost:8080 in non-production environments.
    |
    */
    'base_url' => env('MAILVEX_API_URL', env('APP_ENV') === 'production' ? '' : 'http://localhost:8080'),

    /*
    |--------------------------------------------------------------------------
    | Default Tenant ID
    |--------------------------------------------------------------------------
    |
    | Optional tenant ULID for multi-tenant setups. Passed as X-Tenant-ID header.
    |
    */
    'tenant_id' => env('MAILVEX_TENANT_ID', null),

    /*
    |--------------------------------------------------------------------------
    | Webhook Signing Secret
    |--------------------------------------------------------------------------
    |
    | The secret key used to verify HMAC-SHA256 signatures of incoming webhooks.
    | Begins with whsec_...
    |
    */
    'webhook_secret' => env('MAILVEX_WEBHOOK_SECRET', null),

    /*
    |--------------------------------------------------------------------------
    | Webhook Timestamp Tolerance (Seconds)
    |--------------------------------------------------------------------------
    |
    | Anti-replay window in seconds. Webhook timestamps differing by more than
    | this tolerance will be rejected (default 300 seconds = 5 minutes).
    |
    */
    'webhook_tolerance' => (int) env('MAILVEX_WEBHOOK_TOLERANCE', 300),

    /*
    |--------------------------------------------------------------------------
    | HTTP Client Timeout & Retries
    |--------------------------------------------------------------------------
    |
    | Request timeout in seconds and maximum automatic retries on network
    | failures or 429 / 5xx responses.
    |
    */
    'timeout' => (int) env('MAILVEX_TIMEOUT', 30),
    'max_retries' => (int) env('MAILVEX_MAX_RETRIES', 3),
];
