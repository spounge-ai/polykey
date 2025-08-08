package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
	"github.com/spounge-ai/polykey/internal/infra/config"
)

func main() {
	cfg, err := config.Load(os.Getenv("POLYKEY_CONFIG_PATH"))
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	m, err := migrate.New(
		"file://migrations",
		cfg.NeonDB.URL)
	if err != nil {
		log.Fatalf("failed to create migration instance: %v", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		log.Fatalf("migration failed: %v", err)
	}

	fmt.Println("Migrations completed successfully.")

	// Verification step
	fmt.Println("\nVerifying tables in 'public' schema...")
	db, err := sql.Open("postgres", cfg.NeonDB.URL)
	if err != nil {
		log.Fatalf("failed to connect to database for verification: %v", err)
	}
	defer db.Close()


rows, err := db.Query(`SELECT table_name FROM information_schema.tables WHERE table_schema = 'public'`)
	if err != nil {
		log.Fatalf("failed to query tables: %v", err)
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			log.Fatalf("failed to scan table name: %v", err)
		}
	tables = append(tables, tableName)
	}

	if len(tables) > 0 {
		fmt.Println("Tables found:")
		for _, table := range tables {
			fmt.Printf("- %s\n", table)
		}
	} else {
		fmt.Println("No tables found in 'public' schema.")
	}
}
