<?php

declare(strict_types=1);

namespace Mailvex\Laravel;

use Illuminate\Support\Facades\Route;
use Illuminate\Support\ServiceProvider;
use Mailvex\Client;
use Mailvex\Config;
use Mailvex\Laravel\Http\Controllers\WebhookController;
use Mailvex\Laravel\Transport\MailvexTransport;

class MailvexServiceProvider extends ServiceProvider
{
    /**
     * Register any application services.
     */
    public function register(): void
    {
        $this->mergeConfigFrom(__DIR__ . '/../../config/mailvex.php', 'mailvex');

        $this->app->singleton(Client::class, function ($app) {
            $config = $app['config']['mailvex'] ?? [];

            $mailvexConfig = new Config(
                apiKey: (string) ($config['api_key'] ?? ''),
                baseUrl: (string) ($config['base_url'] ?? Config::DEFAULT_BASE_URL),
                tenantId: $config['tenant_id'] ?? null,
                webhookSecret: $config['webhook_secret'] ?? null,
                timeout: (int) ($config['timeout'] ?? 30),
                maxRetries: (int) ($config['max_retries'] ?? 3)
            );

            return new Client($mailvexConfig);
        });

        $this->app->alias(Client::class, 'mailvex');
    }

    /**
     * Bootstrap any application services.
     */
    public function boot(): void
    {
        if ($this->app->runningInConsole()) {
            $this->publishes([
                __DIR__ . '/../../config/mailvex.php' => $this->app->configPath('mailvex.php'),
            ], 'mailvex-config');
        }

        // Register custom mail transport driver 'mailvex'
        if ($this->app->bound('mail.manager')) {
            $this->app->make('mail.manager')->extend('mailvex', function () {
                return new MailvexTransport($this->app->make(Client::class));
            });
        }

        // Register Route macro for easy webhook route declaration
        if (class_exists(Route::class)) {
            Route::macro('mailvexWebhooks', function (string $url = 'mailvex/webhook') {
                return Route::post($url, WebhookController::class)->name('mailvex.webhook');
            });
        }
    }
}
