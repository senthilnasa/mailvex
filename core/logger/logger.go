package logger

import (
	"context"
	"log/slog"
	"os"
	"strings"
)

type RedactingHandler struct {
	slog.Handler
	redactEmail bool
}

func (h *RedactingHandler) Handle(ctx context.Context, r slog.Record) error {
	var newAttrs []slog.Attr
	r.Attrs(func(a slog.Attr) bool {
		newAttrs = append(newAttrs, h.redactAttr(a))
		return true
	})

	newRecord := slog.NewRecord(r.Time, r.Level, r.Message, r.PC)
	newRecord.AddAttrs(newAttrs...)
	return h.Handler.Handle(ctx, newRecord)
}

func (h *RedactingHandler) redactAttr(a slog.Attr) slog.Attr {
	key := strings.ToLower(a.Key)
	if strings.Contains(key, "secret") || strings.Contains(key, "password") || strings.Contains(key, "token") || strings.Contains(key, "key") {
		return slog.String(a.Key, "[REDACTED]")
	}

	if h.redactEmail && (strings.Contains(key, "email") || strings.Contains(key, "recipient") || strings.Contains(key, "address")) {
		if val, ok := a.Value.Any().(string); ok && strings.Contains(val, "@") {
			parts := strings.Split(val, "@")
			if len(parts) == 2 && len(parts[0]) > 2 {
				redacted := parts[0][:2] + "***@" + parts[1]
				return slog.String(a.Key, redacted)
			}
			return slog.String(a.Key, "***@redacted.com")
		}
	}
	return a
}

// New initializes a structured JSON logger
func New(levelStr string, redactEmail bool) *slog.Logger {
	var level slog.Level
	switch strings.ToLower(levelStr) {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	jsonHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	})

	handler := &RedactingHandler{
		Handler:     jsonHandler,
		redactEmail: redactEmail,
	}

	return slog.New(handler)
}
