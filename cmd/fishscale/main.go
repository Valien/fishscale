package main

import (
	"log"
	"net/http"

	"tailscale.com/tsnet"

	"github.com/allen/fishscale/internal/config"
	"github.com/allen/fishscale/internal/database"
	"github.com/allen/fishscale/internal/middleware"
	"github.com/allen/fishscale/internal/server"
	"github.com/allen/fishscale/internal/storage"
)

func main() {
	cfg := config.Load()

	db, err := database.Open(cfg.DBPath)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	store := storage.NewLocalStore(cfg.PhotoDir)

	if cfg.DevMode {
		router := server.NewRouter(cfg, db, store, nil)
		log.Println("DEV MODE: listening on http://localhost:8080")
		log.Fatal(http.ListenAndServe(":8080", router))
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

		log.Printf("fishscale available at https://%s.<tailnet>.ts.net", cfg.TSHostname)
		log.Fatal(http.Serve(ln, router))
	}
}
