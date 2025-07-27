package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

const (
	usageText = `Migration tool for xSync database

Usage:
  migrate [command]

Available Commands:
  up                   Run all available migrations
  down                 Revert all migrations  
  steps [N]            Migrate up/down by N steps (can be negative)
  goto [version]       Migrate to specific version
  force [version]      Force set version without running migrations
  version              Print current migration version

Database Configuration:
  Set database connection via environment variables or flags:

  PostgreSQL:
    DB_TYPE=postgres DB_HOST=localhost DB_PORT=5432 DB_USER=user DB_PASSWORD=pass DB_NAME=xsync
    or use -postgres flag with DSN: "postgres://user:pass@localhost:5432/xsync?sslmode=disable"

  SQLite:
    DB_TYPE=sqlite DB_PATH=/path/to/db.sqlite
    or use -sqlite flag with path: "/path/to/database.db"

Examples:
  migrate -postgres="postgres://user:pass@localhost:5432/xsync?sslmode=disable" up
  migrate -sqlite="/path/to/xsync.db" down
  migrate -postgres="postgres://localhost:5432/xsync" steps 2
  migrate -sqlite="./data/xsync.db" goto 1
`
)

var (
	postgresURL = flag.String("postgres", "", "PostgreSQL connection URL")
	sqlitePath  = flag.String("sqlite", "", "SQLite database file path")
	help        = flag.Bool("help", false, "Show help message")
	h           = flag.Bool("h", false, "Show help message")
)

func main() {
	flag.Parse()

	if *help || *h {
		fmt.Print(usageText)
		return
	}

	args := flag.Args()
	if len(args) == 0 {
		fmt.Print(usageText)
		os.Exit(1)
	}

	command := args[0]

	// Determine database type and connection
	var dbURL string
	var migrationsPath string
	var db *sql.DB
	var driver database.Driver
	var err error

	if *postgresURL != "" {
		// PostgreSQL
		dbURL = *postgresURL
		migrationsPath = "file://postgres"
		db, err = sql.Open("postgres", dbURL)
		if err != nil {
			log.Fatalf("Failed to connect to PostgreSQL: %v", err)
		}
		defer db.Close()

		driver, err = postgres.WithInstance(db, &postgres.Config{})
		if err != nil {
			log.Fatalf("Failed to create PostgreSQL driver: %v", err)
		}

	} else if *sqlitePath != "" {
		// SQLite
		dbURL = *sqlitePath
		migrationsPath = "file://sqlite"
		db, err = sql.Open("sqlite3", dbURL)
		if err != nil {
			log.Fatalf("Failed to connect to SQLite: %v", err)
		}
		defer db.Close()

		driver, err = sqlite3.WithInstance(db, &sqlite3.Config{})
		if err != nil {
			log.Fatalf("Failed to create SQLite driver: %v", err)
		}

	} else {
		// Try environment variables
		dbType := os.Getenv("DB_TYPE")

		switch dbType {
		case "postgres":
			host := getEnvOrDefault("DB_HOST", "localhost")
			port := getEnvOrDefault("DB_PORT", "5432")
			user := getEnvOrDefault("DB_USER", "postgres")
			password := os.Getenv("DB_PASSWORD")
			dbname := getEnvOrDefault("DB_NAME", "xsync")
			sslmode := getEnvOrDefault("DB_SSLMODE", "disable")

			dbURL = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
				user, password, host, port, dbname, sslmode)
			migrationsPath = "file://postgres"

			db, err = sql.Open("postgres", dbURL)
			if err != nil {
				log.Fatalf("Failed to connect to PostgreSQL: %v", err)
			}
			defer db.Close()

			driver, err = postgres.WithInstance(db, &postgres.Config{})
			if err != nil {
				log.Fatalf("Failed to create PostgreSQL driver: %v", err)
			}

		case "sqlite":
			dbPath := getEnvOrDefault("DB_PATH", "./data/xsync.db")
			dbURL = dbPath
			migrationsPath = "file://sqlite"

			db, err = sql.Open("sqlite3", dbURL)
			if err != nil {
				log.Fatalf("Failed to connect to SQLite: %v", err)
			}
			defer db.Close()

			driver, err = sqlite3.WithInstance(db, &sqlite3.Config{})
			if err != nil {
				log.Fatalf("Failed to create SQLite driver: %v", err)
			}

		default:
			log.Fatal("No database configuration found. Use flags or set environment variables.")
		}
	}

	// Create migrate instance
	m, err := migrate.NewWithDatabaseInstance(migrationsPath, "", driver)
	if err != nil {
		log.Fatalf("Failed to create migrate instance: %v", err)
	}

	// Execute command
	switch command {
	case "up":
		if err := m.Up(); err != nil && err != migrate.ErrNoChange {
			log.Fatalf("Failed to run migrations up: %v", err)
		}
		fmt.Println("Migrations applied successfully")

	case "down":
		if err := m.Down(); err != nil && err != migrate.ErrNoChange {
			log.Fatalf("Failed to run migrations down: %v", err)
		}
		fmt.Println("Migrations reverted successfully")

	case "steps":
		if len(args) < 2 {
			log.Fatal("steps command requires a number argument")
		}
		var steps int
		if _, err := fmt.Sscanf(args[1], "%d", &steps); err != nil {
			log.Fatalf("Invalid steps number: %v", err)
		}
		if err := m.Steps(steps); err != nil && err != migrate.ErrNoChange {
			log.Fatalf("Failed to run migration steps: %v", err)
		}
		fmt.Printf("Applied %d migration steps\n", steps)

	case "goto":
		if len(args) < 2 {
			log.Fatal("goto command requires a version argument")
		}
		var version uint
		if _, err := fmt.Sscanf(args[1], "%d", &version); err != nil {
			log.Fatalf("Invalid version number: %v", err)
		}
		if err := m.Migrate(version); err != nil && err != migrate.ErrNoChange {
			log.Fatalf("Failed to migrate to version %d: %v", version, err)
		}
		fmt.Printf("Migrated to version %d\n", version)

	case "force":
		if len(args) < 2 {
			log.Fatal("force command requires a version argument")
		}
		var version int
		if _, err := fmt.Sscanf(args[1], "%d", &version); err != nil {
			log.Fatalf("Invalid version number: %v", err)
		}
		if err := m.Force(version); err != nil {
			log.Fatalf("Failed to force version %d: %v", version, err)
		}
		fmt.Printf("Forced version to %d\n", version)

	case "version":
		version, dirty, err := m.Version()
		if err != nil {
			log.Fatalf("Failed to get version: %v", err)
		}
		status := "clean"
		if dirty {
			status = "dirty"
		}
		fmt.Printf("Current version: %d (%s)\n", version, status)

	default:
		fmt.Printf("Unknown command: %s\n\n", command)
		fmt.Print(usageText)
		os.Exit(1)
	}
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
