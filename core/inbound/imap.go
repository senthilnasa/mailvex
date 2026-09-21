package inbound

import (
	"context"
	"io"
	"time"
)

// IMAPMailboxConfig defines connection parameters for an IMAP mailbox listener/poller
type IMAPMailboxConfig struct {
	ID             string        `json:"id"`
	TenantID       string        `json:"tenant_id"`
	Host           string        `json:"host"`
	Port           int           `json:"port"` // usually 993 for IMAPS
	UseTLS         bool          `json:"use_tls"`
	Username       string        `json:"username"`
	Password       string        `json:"password"`
	MailboxFolder  string        `json:"mailbox_folder"` // default "INBOX"
	PollInterval   time.Duration `json:"poll_interval"`
	IdleEnabled    bool          `json:"idle_enabled"` // RFC 2177 IMAP IDLE
	DeleteOnIngest bool          `json:"delete_on_ingest"`
}

// IMAPMessageSummary encapsulates an email fetched from an IMAP server
type IMAPMessageSummary struct {
	UID         uint32            `json:"uid"`
	MessageID   string            `json:"message_id"`
	From        string            `json:"from"`
	To          []string          `json:"to"`
	Subject     string            `json:"subject"`
	Date        time.Time         `json:"date"`
	Flags       []string          `json:"flags"`
	Size        int64             `json:"size"`
	RawMIMERead io.ReadCloser     `json:"-"`
}

// IMAPSyncStats captures performance counters for an IMAP synchronization run
type IMAPSyncStats struct {
	FetchedCount   int           `json:"fetched_count"`
	IngestedCount  int           `json:"ingested_count"`
	SkippedCount   int           `json:"skipped_count"`
	ErrorCount     int           `json:"error_count"`
	Duration       time.Duration `json:"duration"`
}

// IMAPMailboxHook is the pluggable extension interface for IMAP polling and ingestion (§35)
type IMAPMailboxHook interface {
	// Connect initiates a TLS session to the remote IMAP server
	Connect(ctx context.Context, cfg *IMAPMailboxConfig) error

	// SyncFolder polls or awaits IDLE notifications and yields new incoming messages
	SyncFolder(ctx context.Context, folder string, lastUID uint32) ([]*IMAPMessageSummary, error)

	// FetchMessageRaw downloads the complete raw RFC 5322 MIME stream for a given UID
	FetchMessageRaw(ctx context.Context, uid uint32) (io.ReadCloser, error)

	// MarkIngested marks the message as processed (e.g. adding \Seen flag or moving to archive)
	MarkIngested(ctx context.Context, uid uint32) error

	// Disconnect terminates the IMAP session cleanly
	Disconnect(ctx context.Context) error
}
