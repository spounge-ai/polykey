package main

import (
	"database/sql"
	"fmt"
	"io/ioutil"
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

	// Read and display migration file content
	fmt.Println("Migration file content:")
	content, err := ioutil.ReadFile("migrations/001_create_keys_table.up.sql")
	if err != nil {
		log.Fatalf("failed to read migration file: %v", err)
	}
	fmt.Printf("SQL:\n%s\n", string(content))

	// Connect to database first
	db, err := sql.Open("postgres", cfg.NeonDB.URL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("failed to ping database: %v", err)
	}

	// Try manual execution first
	fmt.Println("\nExecuting SQL manually...")
	_, err = db.Exec(string(content))
	if err != nil {
		log.Printf("Manual execution failed: %v", err)
	} else {
		fmt.Println("Manual execution succeeded")
	}

	// Check if table exists after manual execution
	var exists bool
	err = db.QueryRow(`
		SELECT EXISTS (
			SELECT FROM information_schema.tables 
			WHERE table_schema = 'public' 
			AND table_name = 'keys'
		)
	`).Scan(&exists)
	if err != nil {
		log.Printf("failed to check table existence: %v", err)
	} else {
		fmt.Printf("Keys table exists after manual execution: %t\n", exists)
	}

	// If manual worked, try migration
	if exists {
		fmt.Println("\nDropping table to test migration...")
		_, err = db.Exec("DROP TABLE keys")
		if err != nil {
			log.Printf("failed to drop table: %v", err)
		}
	}

	// Reset migration state
	fmt.Println("\nResetting migration state...")
	_, err = db.Exec("DELETE FROM schema_migrations WHERE version = 1")
	if err != nil {
		log.Printf("failed to reset migration state: %v", err)
	}

	// Now try migration
	fmt.Println("\nRunning migration...")
	m, err := migrate.New("file://migrations", cfg.NeonDB.URL)
	if err != nil {
		log.Fatalf("failed to create migration instance: %v", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		log.Printf("migration failed: %v", err)
	} else {
		fmt.Println("Migration completed")
	}

	// Final check
	err = db.QueryRow(`
		SELECT EXISTS (
			SELECT FROM information_schema.tables 
			WHERE table_schema = 'public' 
			AND table_name = 'keys'
		)
	`).Scan(&exists)
	if err != nil {
		log.Printf("failed final check: %v", err)
	} else {
		fmt.Printf("Keys table exists after migration: %t\n", exists)
	}
}