package integration_test

import (
	"context"
	"fmt"
	"log"
	"os"
	"testing"
	"path/filepath" // Added import

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
)

var dbpool *pgxpool.Pool

// findModuleRoot finds the directory containing go.mod by traversing up from the current directory.
func findModuleRoot() (string, error) {
	currentDir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		goModPath := filepath.Join(currentDir, "go.mod")
		if _, err := os.Stat(goModPath); err == nil {
			return currentDir, nil // Found go.mod
		}

		parentDir := filepath.Dir(currentDir)
		if parentDir == currentDir { // Reached root directory
			return "", fmt.Errorf("go.mod not found in any parent directory")
		}
		currentDir = parentDir
	}
}

func TestMain(m *testing.M) {
	pool, err := dockertest.NewPool("")
	if err != nil {
		log.Fatalf("Could not construct pool: %s", err)
	}

	err = pool.Client.Ping()
	if err != nil {
		log.Fatalf("Could not connect to Docker: %s", err)
	}

	// pull postgres docker image for version 13
	resource, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "postgres",
		Tag:        "13",
		Env: []string{
			"POSTGRES_PASSWORD=secret",
			"POSTGRES_USER=user",
			"POSTGRES_DB=polykey",
			"listen_addresses = '*'",
		},
	}, func(config *docker.HostConfig) {
		config.AutoRemove = true
		config.RestartPolicy = docker.RestartPolicy{Name: "no"}
	})
	if err != nil {
		log.Fatalf("Could not start resource: %s", err)
	}

	hostAndPort := resource.GetHostPort("5432/tcp")
	databaseUrl := fmt.Sprintf("postgres://user:secret@%s/polykey?sslmode=disable", hostAndPort)

	log.Println("Connecting to database on url: ", databaseUrl)

		if err := resource.Expire(120); err != nil {
		log.Fatalf("Could not set resource expiration: %s", err)
	}

	// exponential backoff-retry, because the application in the container might not be ready to accept connections yet
	if err := pool.Retry(func() error {
		var err error
		dbpool, err = pgxpool.New(context.Background(), databaseUrl)
		if err != nil {
			return err
		}
		return dbpool.Ping(context.Background())
	}); err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}

	// run migrations
	moduleRoot, err := findModuleRoot() // Use the new helper
	if err != nil {
		log.Fatalf("Could not find module root: %s", err)
	}
	migrationsPath := filepath.Join(moduleRoot, "migrations") // Construct path relative to module root

	mig, err := migrate.New("file://" + migrationsPath, databaseUrl)
	if err != nil {
		log.Fatalf("Could not create migrate instance: %s", err)
	}
	if err := mig.Up(); err != nil && err != migrate.ErrNoChange {
		log.Fatalf("Could not run migrations: %s", err)
	}

	code := m.Run()

	// You can't defer this because os.Exit doesn't care for defer
	if err := pool.Purge(resource); err != nil {
		log.Fatalf("Could not purge resource: %s", err)
	}

	os.Exit(code)
}

func truncate(t *testing.T) {
	_, err := dbpool.Exec(context.Background(), "TRUNCATE keys, audit_events RESTART IDENTITY")
	if err != nil {
		t.Fatalf("failed to truncate database: %v", err)
	}
}