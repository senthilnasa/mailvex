package handlers

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/mailvex/mailvex/core/crypto"
	"github.com/mailvex/mailvex/core/domain"
	"github.com/mailvex/mailvex/core/repository"
)

type UnsubscribeHandler struct {
	repo   repository.Repository
	secret []byte
}

func NewUnsubscribeHandler(repo repository.Repository, secret []byte) *UnsubscribeHandler {
	if len(secret) == 0 {
		secret = []byte("mailvex-unsubscribe-token-secret")
	}
	return &UnsubscribeHandler{
		repo:   repo,
		secret: secret,
	}
}

// HandleOneClick handles RFC 8058 One-Click POST /u/{token} (§17.3)
func (h *UnsubscribeHandler) HandleOneClick(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")
	if token == "" {
		http.Error(w, "invalid token", http.StatusBadRequest)
		return
	}

	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		http.Error(w, "malformed token", http.StatusBadRequest)
		return
	}
	rcptID, sig := parts[0], parts[1]

	rcpt, err := h.repo.GetRecipient(r.Context(), rcptID)
	if err != nil {
		http.Error(w, "recipient not found", http.StatusNotFound)
		return
	}

	// Validate HMAC signature
	mac := hmac.New(sha256.New, h.secret)
	mac.Write([]byte(fmt.Sprintf("%s:%s:%s", rcpt.TenantID, rcpt.ID, rcpt.Address)))
	expectedSig := hex.EncodeToString(mac.Sum(nil))[:24]

	if !hmac.Equal([]byte(sig), []byte(expectedSig)) {
		http.Error(w, "unauthorized token", http.StatusForbidden)
		return
	}

	// Read message to get stream ID
	msg, err := h.repo.GetMessageByID(r.Context(), rcpt.TenantID, rcpt.MessageID)
	if err != nil {
		http.Error(w, "message not found", http.StatusNotFound)
		return
	}

	// Insert suppression row: type=unsubscribe (§18)
	source := "one_click"
	bodyBytes, _ := io.ReadAll(r.Body)
	if strings.Contains(string(bodyBytes), "List-Unsubscribe=One-Click") {
		source = "rfc8058_one_click"
	}

	reason := "Unsubscribed by recipient request"
	supp := &domain.Suppression{
		ID:            crypto.NewULID("supp_"),
		TenantID:      rcpt.TenantID,
		StreamID:      &msg.StreamID,
		Address:       rcpt.Address,
		AddressDomain: rcpt.AddressDomain,
		Type:          "unsubscribe",
		Reason:        &reason,
		Source:        source,
		CreatedAt:     time.Now().UTC(),
	}

	if err := h.repo.AddSuppression(r.Context(), supp); err != nil {
		http.Error(w, "failed to record unsubscribe", http.StatusInternalServerError)
		return
	}

	// Emit message.unsubscribed event (§13.2)
	_ = h.repo.AddMessageEvent(r.Context(), &domain.MessageEvent{
		ID:          crypto.NewULID(crypto.PrefixEvent),
		TenantID:    rcpt.TenantID,
		MessageID:   msg.ID,
		RecipientID: &rcpt.ID,
		Type:        "message.unsubscribed",
		Data:        fmt.Sprintf(`{"address":"%s","source":"%s"}`, rcpt.Address, source),
		OccurredAt:  time.Now().UTC(),
		CreatedAt:   time.Now().UTC(),
	})

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("Unsubscribed successfully\n"))
}

// HandleHostedPage renders a minimal, brandable confirmation page with one button GET /u/{token} (§17.3)
func (h *UnsubscribeHandler) HandleHostedPage(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")
	if token == "" {
		http.Error(w, "invalid token", http.StatusBadRequest)
		return
	}

	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		http.Error(w, "malformed token", http.StatusBadRequest)
		return
	}
	rcptID, sig := parts[0], parts[1]

	rcpt, err := h.repo.GetRecipient(r.Context(), rcptID)
	if err != nil {
		http.Error(w, "recipient not found", http.StatusNotFound)
		return
	}

	// Validate HMAC signature
	mac := hmac.New(sha256.New, h.secret)
	mac.Write([]byte(fmt.Sprintf("%s:%s:%s", rcpt.TenantID, rcpt.ID, rcpt.Address)))
	expectedSig := hex.EncodeToString(mac.Sum(nil))[:24]

	if !hmac.Equal([]byte(sig), []byte(expectedSig)) {
		http.Error(w, "unauthorized token", http.StatusForbidden)
		return
	}

	html := fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Unsubscribe Confirmation - Mailvex</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif; background: #0f172a; color: #f8fafc; display: flex; align-items: center; justify-content: center; min-height: 100vh; margin: 0; }
        .card { background: #1e293b; border: 1px solid #334155; border-radius: 12px; padding: 2.5rem; max-width: 440px; width: 100%%; text-align: center; box-shadow: 0 10px 25px -5px rgba(0,0,0,0.3); }
        h1 { font-size: 1.5rem; margin-top: 0; color: #f8fafc; }
        p { color: #94a3b8; font-size: 0.95rem; line-height: 1.5; margin-bottom: 2rem; }
        .address { font-weight: 600; color: #38bdf8; }
        button { background: #e11d48; color: white; border: none; padding: 0.75rem 1.5rem; font-size: 1rem; font-weight: 600; border-radius: 8px; cursor: pointer; width: 100%%; transition: background 0.2s; }
        button:hover { background: #be123c; }
    </style>
</head>
<body>
    <div class="card">
        <h1>Confirm Unsubscribe</h1>
        <p>Are you sure you want to stop receiving messages sent to <span class="address">%s</span>?</p>
        <form method="POST" action="/u/%s">
            <input type="hidden" name="List-Unsubscribe" value="One-Click">
            <button type="submit">Confirm Unsubscribe</button>
        </form>
    </div>
</body>
</html>`, rcpt.Address, token)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(html))
}
