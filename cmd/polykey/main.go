package main

import (
	"log"
	"os"

	"github.com/spounge-ai/polykey/internal/config"
	"github.com/spounge-ai/polykey/internal/server"
)

func main() {
	cfg, err := config.Load(os.Getenv("POLYKEY_CONFIG_PATH"))
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	srv, _, err := server.New(cfg)
	if err != nil {
		log.Fatalf("failed to create server: %v", err)
	}

	// Use RunBlocking instead of Run with context.Background()
	if err := srv.RunBlocking(); err != nil {
		log.Fatalf("server error: %v", err)
	}
}