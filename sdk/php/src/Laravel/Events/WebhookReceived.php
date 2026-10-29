<?php

declare(strict_types=1);

namespace Mailvex\Laravel\Events;

use Mailvex\Webhooks\WebhookEvent;

class WebhookReceived
{
    public WebhookEvent $event;
    public array $payload;

    public function __construct(WebhookEvent $event, array $payload)
    {
        $this->event = $event;
        $this->payload = $payload;
    }
}
