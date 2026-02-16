package main

import (
	"context"
	"log"
	"net"
	"net/http"
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

	db, err := database.Open(cfg.DBPath)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	store := storage.NewLocalStore(cfg.PhotoDir)

	if cfg.DevMode {
		log.Println("DEV MODE: listening on http://localhost:8080")
		if err := runDevServer(ctx, ":8080"); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
		log.Println("server stopped gracefully")
	} else {
		ts := &tsnet.Server{
			Hostname: cfg.TSHostname,
			AuthKey:  cfg.TSAuthKey,
			Dir:      cfg.TSStateDir,
		}
		defer ts.Close()

		lc, err := ts.LocalClient()
		if err != nil {
			log.Fatalf("tsnet local client: %v", err)
		}

		authMW := middleware.TailscaleAuth(lc, db)
		router := server.NewRouter(cfg, db, store, authMW)

		ln, err := ts.ListenTLS("tcp", ":443")
		if err != nil {
			log.Fatalf("tsnet listen: %v", err)
		}
		defer ln.Close()

		srv := &http.Server{Handler: router}

		go func() {
			<-ctx.Done()
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			srv.Shutdown(shutdownCtx)
		}()

		log.Printf("fishscale available at https://%s.<tailnet>.ts.net", cfg.TSHostname)
		if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
		log.Println("server stopped gracefully")
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

	srv := &http.Server{Handler: router}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		srv.Shutdown(shutdownCtx)
	}()

	return srv.Serve(ln)
}
