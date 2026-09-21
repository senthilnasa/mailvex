package inbound_test

import (
	"context"
	"testing"

	"github.com/mailvex/mailvex/core/domain"
	"github.com/mailvex/mailvex/core/inbound"
	"github.com/mailvex/mailvex/core/repository"
)

func TestInboundRouteMatchingAndProcessing(t *testing.T) {
	repo := repository.NewMemoryRepository()
	engine := inbound.NewInboundEngine(repo)
	ctx := context.Background()
	tenantID := "tenant_inbound_test"

	targetWebhook := "https://api.example.com/inbound-webhook"

	// 1. Seed routes:
	// Route 1: exact support@domain.com (priority 5) -> webhook
	routeExact := &domain.InboundRoute{
		ID:           "route_exact",
		TenantID:     tenantID,
		Priority:     5,
		MatchType:    "exact",
		Pattern:      "support@domain.com",
		Action:       "webhook",
		ActionTarget: &targetWebhook,
		IsActive:     true,
	}
	// Route 2: wildcard tickets-*@domain.com (priority 10) -> webhook
	routeWildcard := &domain.InboundRoute{
		ID:           "route_wildcard",
		TenantID:     tenantID,
		Priority:     10,
		MatchType:    "wildcard",
		Pattern:      "tickets-*@domain.com",
		Action:       "webhook",
		ActionTarget: &targetWebhook,
		IsActive:     true,
	}
	// Route 3: catch_all @domain.com (priority 20) -> store_only
	routeCatchAll := &domain.InboundRoute{
		ID:        "route_catchall",
		TenantID:  tenantID,
		Priority:  20,
		MatchType: "catch_all",
		Pattern:   "@domain.com",
		Action:    "store_only",
		IsActive:  true,
	}

	_ = engine.CreateRoute(ctx, routeExact)
	_ = engine.CreateRoute(ctx, routeWildcard)
	_ = engine.CreateRoute(ctx, routeCatchAll)

	// 2. Test exact match
	match1, err := engine.TestRouteMatch(ctx, tenantID, "support@domain.com")
	if err != nil {
		t.Fatalf("expected exact match, got error: %v", err)
	}
	if match1.ID != "route_exact" {
		t.Errorf("expected route_exact, got %s", match1.ID)
	}

	// 3. Test wildcard match
	match2, err := engine.TestRouteMatch(ctx, tenantID, "tickets-456@domain.com")
	if err != nil {
		t.Fatalf("expected wildcard match, got error: %v", err)
	}
	if match2.ID != "route_wildcard" {
		t.Errorf("expected route_wildcard, got %s", match2.ID)
	}

	// 4. Test catch_all match
	match3, err := engine.TestRouteMatch(ctx, tenantID, "random-user@domain.com")
	if err != nil {
		t.Fatalf("expected catch_all match, got error: %v", err)
	}
	if match3.ID != "route_catchall" {
		t.Errorf("expected route_catchall, got %s", match3.ID)
	}

	// 5. Test no route for different domain
	_, err = engine.TestRouteMatch(ctx, tenantID, "user@otherdomain.com")
	if err == nil {
		t.Fatalf("expected error for unrouted domain, got match")
	}

	// 6. Test full ProcessInboundMessage
	rawMIME := []byte("From: sender@external.com\r\nTo: support@domain.com\r\nSubject: Need help\r\n\r\nHelp body")
	msg, matchedRoute, err := engine.ProcessInboundMessage(
		ctx,
		tenantID,
		"sender@external.com",
		"support@domain.com",
		"Need help",
		rawMIME,
		"pass", "pass", "pass",
		0.05,
	)
	if err != nil {
		t.Fatalf("failed to process inbound message: %v", err)
	}
	if matchedRoute.ID != "route_exact" {
		t.Errorf("expected matched route_exact, got %s", matchedRoute.ID)
	}
	if msg.Status != "processed" {
		t.Errorf("expected status processed, got %s", msg.Status)
	}
	if msg.SizeBytes != len(rawMIME) {
		t.Errorf("expected size %d, got %d", len(rawMIME), msg.SizeBytes)
	}

	// Check stored inbound messages
	list, err := engine.ListMessages(ctx, tenantID, 10, 0)
	if err != nil {
		t.Fatalf("failed to list inbound messages: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 inbound message, got %d", len(list))
	}
}

func TestDisabledInboundRoute(t *testing.T) {
	repo := repository.NewMemoryRepository()
	engine := inbound.NewInboundEngine(repo)
	ctx := context.Background()
	tenantID := "tenant_disabled_test"

	route := &domain.InboundRoute{
		ID:        "route_off",
		TenantID:  tenantID,
		Priority:  1,
		MatchType: "exact",
		Pattern:   "inactive@domain.com",
		Action:    "webhook",
		IsActive:  false, // disabled
	}
	_ = engine.CreateRoute(ctx, route)

	// Inactive routes are skipped during route matching (§21)
	_, _, err := engine.ProcessInboundMessage(
		ctx,
		tenantID,
		"sender@external.com",
		"inactive@domain.com",
		"Hello",
		[]byte("MIME"),
		"none", "none", "none",
		0,
	)
	if err == nil {
		t.Fatalf("expected error when route is disabled, got nil")
	}
}
