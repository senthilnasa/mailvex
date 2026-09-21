package handlers

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/mailvex/mailvex/core/domain"
	"github.com/mailvex/mailvex/core/tracking"
)

// HandleOpenPixel serves the 1x1 transparent tracking pixel (GET /o/{token})
func (a *API) HandleOpenPixel(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")
	now := time.Now().UTC()

	if token != "" {
		tenantID, messageID, recipientID, err := a.tracking.VerifyOpenToken(token)
		if err == nil {
			ip := extractClientIP(r)
			ua := r.UserAgent()

			var deliveredTime time.Time
			if rcpt, err := a.repo.GetRecipient(r.Context(), recipientID); err == nil && rcpt != nil && rcpt.DeliveredAt != nil {
				deliveredTime = *rcpt.DeliveredAt
			}

			isBot, botReason := tracking.ClassifyHit(ua, ip, deliveredTime, now)
			truncIP := tracking.TruncateIP(ip)

			var bReasonPtr *string
			if botReason != "" {
				bReasonPtr = &botReason
			}

			_ = a.repo.RecordTrackingEvent(r.Context(), &domain.TrackingEvent{
				TenantID:    tenantID,
				MessageID:   messageID,
				RecipientID: recipientID,
				Type:        "open",
				IP:          truncIP,
				UserAgent:   &ua,
				IsBot:       isBot,
				BotReason:   bReasonPtr,
				OccurredAt:  now,
			})

			if !isBot {
				evtPayload := fmt.Sprintf(`{"ip":"%s","user_agent":"%s"}`, truncIP, ua)
				_ = a.repo.AddMessageEvent(r.Context(), &domain.MessageEvent{
					TenantID:    tenantID,
					MessageID:   messageID,
					RecipientID: &recipientID,
					Type:        "message.opened",
					Data:        evtPayload,
					OccurredAt:  now,
					CreatedAt:   now,
				})

				_ = a.repo.RecordAnalyticsEvent(r.Context(), tenantID, "message.opened", nil, nil, "unknown", "", 0, now)
			}
		}
	}

	w.Header().Set("Content-Type", "image/gif")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(tracking.TransparentGIF)
}

// HandleClickRedirect handles signed link redirection (GET /c/{token})
func (a *API) HandleClickRedirect(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")
	if token == "" {
		http.NotFound(w, r)
		return
	}

	tenantID, messageID, recipientID, linkID, err := a.tracking.VerifyClickToken(token)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	link, err := a.repo.GetTrackingLink(r.Context(), tenantID, linkID)
	if err != nil || link == nil {
		http.NotFound(w, r)
		return
	}

	now := time.Now().UTC()
	ip := extractClientIP(r)
	ua := r.UserAgent()

	var deliveredTime time.Time
	if rcpt, err := a.repo.GetRecipient(r.Context(), recipientID); err == nil && rcpt != nil && rcpt.DeliveredAt != nil {
		deliveredTime = *rcpt.DeliveredAt
	}

	isBot, botReason := tracking.ClassifyHit(ua, ip, deliveredTime, now)
	truncIP := tracking.TruncateIP(ip)

	var bReasonPtr *string
	if botReason != "" {
		bReasonPtr = &botReason
	}

	_ = a.repo.RecordTrackingEvent(r.Context(), &domain.TrackingEvent{
		TenantID:    tenantID,
		MessageID:   messageID,
		RecipientID: recipientID,
		Type:        "click",
		TargetURL:   &link.TargetURL,
		IP:          truncIP,
		UserAgent:   &ua,
		IsBot:       isBot,
		BotReason:   bReasonPtr,
		OccurredAt:  now,
	})

	if !isBot {
		evtPayload := fmt.Sprintf(`{"url":"%s","ip":"%s"}`, link.TargetURL, truncIP)
		_ = a.repo.AddMessageEvent(r.Context(), &domain.MessageEvent{
			TenantID:    tenantID,
			MessageID:   messageID,
			RecipientID: &recipientID,
			Type:        "message.clicked",
			Data:        evtPayload,
			OccurredAt:  now,
			CreatedAt:   now,
		})

		_ = a.repo.RecordAnalyticsEvent(r.Context(), tenantID, "message.clicked", nil, nil, "unknown", "", 0, now)
	}

	http.Redirect(w, r, link.TargetURL, http.StatusFound)
}

func extractClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}
	parts := strings.Split(r.RemoteAddr, ":")
	if len(parts) > 0 {
		return parts[0]
	}
	return "127.0.0.1"
}
