package main

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"tailscale.com/tsnet"

	"github.com/allen/fishscale/internal/config"
	"github.com/allen/fishscale/internal/database"
	"github.com/allen/fishscale/internal/middleware"
	"github.com/allen/fishscale/internal/server"
	"github.com/allen/fishscale/internal/storage"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		slog.Error("invalid configuration", "error", err)
		os.Exit(1)
	}

	logLevel := config.ParseLogLevel(cfg.LogLevel)
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: logLevel})))

	db, err := database.Open(cfg.DBPath)
	if err != nil {
		slog.Error("failed to open database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	store := storage.NewLocalStore(cfg.PhotoDir)

	if cfg.DevMode {
		slog.Info("starting dev server", "addr", ":8080")
		if err := runDevServer(ctx, ":8080"); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
		slog.Info("server stopped gracefully")
	} else {
		ts := &tsnet.Server{
			Hostname: cfg.TSHostname,
			AuthKey:  cfg.TSAuthKey,
			Dir:      cfg.TSStateDir,
		}
		defer ts.Close()

		lc, err := ts.LocalClient()
		if err != nil {
			slog.Error("tsnet local client failed", "error", err)
			os.Exit(1)
		}

		authMW := middleware.TailscaleAuth(lc, db)
		router := server.NewRouter(cfg, db, store, authMW)

		ln, err := ts.ListenTLS("tcp", ":443")
		if err != nil {
			slog.Error("tsnet listen failed", "error", err)
			os.Exit(1)
		}
		defer ln.Close()

		srv := &http.Server{
			Handler:           router,
			ReadHeaderTimeout: 10 * time.Second,
		}

		go func() {
			<-ctx.Done()
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			_ = srv.Shutdown(shutdownCtx) //nolint:contextcheck // fresh context for graceful shutdown after parent cancellation
		}()

		slog.Info("fishscale available", "hostname", cfg.TSHostname)
		if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
		slog.Info("server stopped gracefully")
	}
}

// runDevServer starts an HTTP server on addr that shuts down when ctx is canceled.
func runDevServer(ctx context.Context, addr string) error {
	cfg := config.Load()

	db, err := database.Open(cfg.DBPath)
	if err != nil {
		return err
	}
	defer db.Close()

	store := storage.NewLocalStore(cfg.PhotoDir)
	router := server.NewRouter(cfg, db, store, nil)

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	srv := &http.Server{
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx) //nolint:contextcheck // fresh context for graceful shutdown after parent cancellation
	}()

	return srv.Serve(ln)
}
