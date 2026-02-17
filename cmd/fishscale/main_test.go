package main

import (
	"context"
	"net/http"
	"testing"
	"time"
)

func TestGracefulShutdown_DevMode(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("FISHSCALE_DEV_MODE", "true")
	t.Setenv("FISHSCALE_DB_PATH", dir+"/test.db")
	t.Setenv("FISHSCALE_PHOTO_DIR", dir+"/photos")

	// Start server in background with a cancel context
	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)

	go func() {
		errCh <- runDevServer(ctx, ":0") // :0 = random port
	}()

	// Give server time to start
	time.Sleep(200 * time.Millisecond)

	// Cancel context (simulate SIGTERM)
	cancel()

	select {
	case err := <-errCh:
		if err != nil && err != http.ErrServerClosed {
			t.Fatalf("unexpected error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("server did not shut down within 5 seconds")
	}
}
