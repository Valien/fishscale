package main

import (
	"fmt"
	"log"
	"os"

	"github.com/allen/fishscale/internal/config"
)

func main() {
	cfg := config.Load()

	if cfg.LogLevel == "debug" {
		fmt.Fprintf(os.Stderr, "config: %+v\n", cfg)
	}

	log.Println("fishscale starting...")
	// Server setup will go here in later tasks.
}
