package main

import (
	"log"
	"net/http"

	"github.com/allen/fishscale/internal/config"
	"github.com/allen/fishscale/internal/database"
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

	router := server.NewRouter(cfg, db, store)

	if cfg.DevMode {
		log.Println("DEV MODE: listening on http://localhost:8080")
		log.Fatal(http.ListenAndServe(":8080", router))
	} else {
		// tsnet startup will be added in Task 12
		log.Println("fishscale starting on :8080 (tsnet not yet configured)")
		log.Fatal(http.ListenAndServe(":8080", router))
	}
}
