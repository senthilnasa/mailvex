package main

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"testing"
	"time"

	"github.com/senthilnasa/mailvex/core/repository"
	"github.com/senthilnasa/mailvex/core/storage"
)

func TestAllInOneSupervisorLifecycle(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	repo := repository.NewMemoryRepository()
	store := storage.NewMemoryStorage()

	supervisor := NewAllInOneSupervisor(repo, store, logger, "127.0.0.1:12525")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Use random dynamic port for test API server
	apiAddr := "127.0.0.1:18080"
	if err := supervisor.Start(ctx, apiAddr); err != nil {
		t.Fatalf("failed to start supervisor: %v", err)
	}

	// Wait for server to bind
	time.Sleep(100 * time.Millisecond)

	// Verify HTTP /health probe
	resp, err := http.Get("http://" + apiAddr + "/health")
	if err != nil {
		t.Fatalf("failed to reach /health on supervisor: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	// Stop supervisor cleanly
	stopCtx, stopCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer stopCancel()

	if err := supervisor.Stop(stopCtx); err != nil {
		t.Errorf("supervisor stop returned error: %v", err)
	}
}
