package main

import (
	"log"
	"os"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/spounge-ai/polykey/internal/infra/config"
)

func main() {
	cfg, err := config.Load(os.Getenv("POLYKEY_CONFIG_PATH"))
	if err != nil {
		log.Fatalf("FATAL: could not load config: %v", err)
	}

	log.Println("INFO: running database migrations...")

	m, err := migrate.New("file://migrations", cfg.NeonDB.URL)
	if err != nil {
		log.Fatalf("FATAL: failed to create migration instance: %v", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		log.Fatalf("FATAL: migration failed: %v", err)
	}

	log.Println("SUCCESS: migrations completed.")
}
