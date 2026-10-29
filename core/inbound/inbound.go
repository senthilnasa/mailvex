package inbound

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"

	"github.com/senthilnasa/mailvex/core/crypto"
	"github.com/senthilnasa/mailvex/core/domain"
	"github.com/senthilnasa/mailvex/core/repository"
)

var (
	ErrNoMatchingRoute = errors.New("no matching inbound route found for recipient")
	ErrRouteDisabled   = errors.New("matched inbound route is inactive")
)

type InboundEngine struct {
	repo repository.Repository
}

func NewInboundEngine(repo repository.Repository) *InboundEngine {
	return &InboundEngine{repo: repo}
}

// ProcessInboundMessage executes the inbound email processing pipeline (§21)
func (e *InboundEngine) ProcessInboundMessage(
	ctx context.Context,
	tenantID, fromAddr, toAddr, subject string,
	rawMIME []byte,
	spfStatus, dkimStatus, dmarcStatus string,
	spamScore float64,
) (*domain.InboundMessage, *domain.InboundRoute, error) {
	normTo := strings.ToLower(strings.TrimSpace(toAddr))
	normFrom := strings.ToLower(strings.TrimSpace(fromAddr))

	// Match route
	route, err := e.repo.MatchInboundRoute(ctx, tenantID, normTo)
	if err != nil {
		return nil, nil, ErrNoMatchingRoute
	}
	if !route.IsActive {
		return nil, nil, ErrRouteDisabled
	}

	rawRef := ""
	if len(rawMIME) > 0 {
		rawRef = fmt.Sprintf("inbound_raw_%s_%d", tenantID, time.Now().UnixNano())
	}

	status := "processed"
	if route.Action == "discard" {
		status = "discarded"
	}

	msg := &domain.InboundMessage{
		ID:          crypto.NewULID(crypto.PrefixInboundMessage),
		TenantID:    tenantID,
		RouteID:     &route.ID,
		FromAddress: normFrom,
		ToAddress:   normTo,
		Subject:     &subject,
		SizeBytes:   len(rawMIME),
		RawMIMERef:  &rawRef,
		SPFStatus:   spfStatus,
		DKIMStatus:  dkimStatus,
		DMARCStatus: dmarcStatus,
		SpamScore:   spamScore,
		Status:      status,
		CreatedAt:   time.Now().UTC(),
	}

	if err := e.repo.SaveInboundMessage(ctx, msg); err != nil {
		return nil, nil, err
	}

	return msg, route, nil
}

func (e *InboundEngine) CreateRoute(ctx context.Context, route *domain.InboundRoute) error {
	if route.TenantID == "" {
		return errors.New("tenant_id is required")
	}
	if route.Pattern == "" {
		return errors.New("route pattern is required")
	}
	if route.MatchType == "" {
		route.MatchType = "exact"
	}
	if route.Action == "" {
		route.Action = "webhook"
	}
	return e.repo.CreateInboundRoute(ctx, route)
}

func (e *InboundEngine) GetRoute(ctx context.Context, tenantID, id string) (*domain.InboundRoute, error) {
	return e.repo.GetInboundRoute(ctx, tenantID, id)
}

func (e *InboundEngine) ListRoutes(ctx context.Context, tenantID string) ([]*domain.InboundRoute, error) {
	return e.repo.ListInboundRoutes(ctx, tenantID)
}

func (e *InboundEngine) UpdateRoute(ctx context.Context, route *domain.InboundRoute) error {
	return e.repo.UpdateInboundRoute(ctx, route)
}

func (e *InboundEngine) DeleteRoute(ctx context.Context, tenantID, id string) error {
	return e.repo.DeleteInboundRoute(ctx, tenantID, id)
}

func (e *InboundEngine) TestRouteMatch(ctx context.Context, tenantID, testAddress string) (*domain.InboundRoute, error) {
	normAddr := strings.TrimSpace(strings.ToLower(testAddress))
	if _, err := mail.ParseAddress(normAddr); err != nil {
		return nil, errors.New("invalid email address")
	}
	return e.repo.MatchInboundRoute(ctx, tenantID, normAddr)
}

func (e *InboundEngine) GetMessage(ctx context.Context, tenantID, id string) (*domain.InboundMessage, error) {
	return e.repo.GetInboundMessage(ctx, tenantID, id)
}

func (e *InboundEngine) ListMessages(ctx context.Context, tenantID string, limit, offset int) ([]*domain.InboundMessage, error) {
	return e.repo.ListInboundMessages(ctx, tenantID, limit, offset)
}
