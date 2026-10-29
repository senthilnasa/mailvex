package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/senthilnasa/mailvex/api/middleware"
	"github.com/senthilnasa/mailvex/core/auth"
	"github.com/senthilnasa/mailvex/core/domain"
)

// ListMessagesHandler lists messages with cursor-based pagination and filters
func (a *API) ListMessagesHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	tenantID, _ := r.Context().Value(middleware.TenantIDKey).(string)

	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	cursor := q.Get("cursor")
	status := q.Get("status")
	recipient := q.Get("recipient")
	from := q.Get("from")
	stream := q.Get("stream")
	tag := q.Get("tag")

	var sincePtr *time.Time
	if sinceStr := q.Get("since"); sinceStr != "" {
		if t, err := time.Parse(time.RFC3339, sinceStr); err == nil {
			sincePtr = &t
		}
	}

	var untilPtr *time.Time
	if untilStr := q.Get("until"); untilStr != "" {
		if t, err := time.Parse(time.RFC3339, untilStr); err == nil {
			untilPtr = &t
		}
	}

	filter := domain.MessageFilter{
		Limit:     limit,
		Cursor:    cursor,
		Status:    status,
		Recipient: recipient,
		From:      from,
		Stream:    stream,
		Tag:       tag,
		Since:     sincePtr,
		Until:     untilPtr,
	}

	msgs, nextCursor, hasMore, err := a.repo.ListMessagesPaginated(r.Context(), tenantID, filter)
	if err != nil {
		http.Error(w, `{"error":{"code":"internal_error","message":"`+err.Error()+`"}}`, http.StatusInternalServerError)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"data":        msgs,
		"next_cursor": nextCursor,
		"has_more":    hasMore,
	})
}

// GetMessageHandler retrieves a single message by ID
func (a *API) GetMessageHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	tenantID, _ := r.Context().Value(middleware.TenantIDKey).(string)
	id := r.PathValue("id")

	msg, err := a.repo.GetMessageByID(r.Context(), tenantID, id)
	if err != nil {
		http.Error(w, `{"error":{"code":"message_not_found","message":"Message not found"}}`, http.StatusNotFound)
		return
	}

	rcpts, _ := a.repo.GetMessageRecipients(r.Context(), tenantID, id)

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"message":    msg,
		"recipients": rcpts,
	})
}

// GetMessageRecipientsHandler retrieves recipients for a message
func (a *API) GetMessageRecipientsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	tenantID, _ := r.Context().Value(middleware.TenantIDKey).(string)
	id := r.PathValue("id")

	rcpts, err := a.repo.GetMessageRecipients(r.Context(), tenantID, id)
	if err != nil {
		http.Error(w, `{"error":{"code":"internal_error","message":"`+err.Error()+`"}}`, http.StatusInternalServerError)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"data": rcpts,
	})
}

// GetMessageEventsHandler retrieves chronological event log for a message
func (a *API) GetMessageEventsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	tenantID, _ := r.Context().Value(middleware.TenantIDKey).(string)
	id := r.PathValue("id")

	evts, err := a.repo.GetMessageEvents(r.Context(), tenantID, id)
	if err != nil {
		http.Error(w, `{"error":{"code":"internal_error","message":"`+err.Error()+`"}}`, http.StatusInternalServerError)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"data": evts,
	})
}

// GetMessageBodyHandler retrieves stored body (HTML and Text)
func (a *API) GetMessageBodyHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	tenantID, _ := r.Context().Value(middleware.TenantIDKey).(string)
	id := r.PathValue("id")

	msg, err := a.repo.GetMessageByID(r.Context(), tenantID, id)
	if err != nil {
		http.Error(w, `{"error":{"code":"message_not_found","message":"Message not found"}}`, http.StatusNotFound)
		return
	}

	if msg.BodyRef == nil {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id":   msg.ID,
			"body": "",
		})
		return
	}

	rawBody, err := a.storage.Get(r.Context(), *msg.BodyRef)
	if err != nil {
		http.Error(w, `{"error":{"code":"body_not_found","message":"Message body not found or retention expired"}}`, http.StatusNotFound)
		return
	}

	bodyStr := string(rawBody)
	isSupport, _ := r.Context().Value(middleware.IsSupportRoleKey).(bool)
	bodyStr = auth.MaskMessageBody(bodyStr, isSupport)

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"id":     msg.ID,
		"body":   bodyStr,
		"masked": isSupport,
	})
}

// RetryMessageHandler reschedules delivery for failed/deferred recipients
func (a *API) RetryMessageHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	tenantID, _ := r.Context().Value(middleware.TenantIDKey).(string)
	id := r.PathValue("id")

	if err := a.repo.UpdateMessageStatus(r.Context(), tenantID, id, "queued"); err != nil {
		http.Error(w, `{"error":{"code":"update_failed","message":"Failed to retry message"}}`, http.StatusInternalServerError)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"id":     id,
		"status": "queued",
	})
}

// CancelMessageHandler cancels a pending or queued message
func (a *API) CancelMessageHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	tenantID, _ := r.Context().Value(middleware.TenantIDKey).(string)
	id := r.PathValue("id")

	if err := a.repo.UpdateMessageStatus(r.Context(), tenantID, id, "cancelled"); err != nil {
		http.Error(w, `{"error":{"code":"cancel_failed","message":"Failed to cancel message"}}`, http.StatusInternalServerError)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"id":     id,
		"status": "cancelled",
	})
}

// ListDevMessagesHandler lists messages intercepted in the development mailbox
func (a *API) ListDevMessagesHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	tenantID, _ := r.Context().Value(middleware.TenantIDKey).(string)

	msgs, err := a.repo.ListDevelopmentMessages(r.Context(), tenantID)
	if err != nil {
		http.Error(w, `{"error":{"code":"internal_error","message":"`+err.Error()+`"}}`, http.StatusInternalServerError)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]any{
		"data": msgs,
	})
}
