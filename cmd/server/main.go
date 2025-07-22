package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/WangWilly/xSync/pkgs/clipkg/config"
	"github.com/WangWilly/xSync/pkgs/serverpkg/server"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Try to load configuration from file first
	var dbConfig config.DatabaseConfig
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		// Default configuration path
		homeDir, err := os.UserHomeDir()
		if err == nil {
			configPath = filepath.Join(homeDir, ".x_sync", "conf.yaml")
		}
	}

	if configPath != "" {
		if conf, err := config.ParseConfigFromFile(configPath); err == nil {
			dbConfig = conf.Database
			log.Printf("Loaded database configuration from: %s", configPath)
		}
	}

	// If no configuration loaded, check environment variables for database setup
	if dbConfig.Type == "" {
		dbType := os.Getenv("DB_TYPE")
		if dbType == "" {
			dbType = "sqlite"
		}
		dbConfig.Type = dbType

		switch dbType {
		case "postgres", "postgresql":
			dbConfig.Host = getEnvOrDefault("DB_HOST", "localhost")
			dbConfig.Port = getEnvOrDefault("DB_PORT", "5432")
			dbConfig.User = getEnvOrDefault("DB_USER", "xsync")
			dbConfig.Password = getEnvOrDefault("DB_PASSWORD", "")
			dbConfig.DBName = getEnvOrDefault("DB_NAME", "xsync")
		default:
			// SQLite
			dbPath := getEnvOrDefault("DB_PATH", "./conf/data/xSync.db")
			dbConfig.Path = dbPath
		}
	}

	// If SQLite and no path set, use default
	if (dbConfig.Type == "sqlite" || dbConfig.Type == "sqlite3" || dbConfig.Type == "") && dbConfig.Path == "" {
		dbConfig.Path = "./conf/data/xSync.db"
		dbConfig.Type = "sqlite"
	}

	srv, err := server.NewServerWithConfig(dbConfig, port)
	if err != nil {
		log.Fatal("Failed to create server:", err)
	}
	defer srv.Close()

	log.Printf("Starting server on port %s", port)
	log.Printf("Database type: %s", dbConfig.Type)
	if dbConfig.Type == "sqlite" || dbConfig.Type == "sqlite3" {
		log.Printf("Database path: %s", dbConfig.Path)
	} else {
		log.Printf("Database host: %s:%s", dbConfig.Host, dbConfig.Port)
	}
	log.Printf("Open http://localhost:%s to view the dashboard", port)

	if err := srv.Start(); err != nil {
		log.Fatal("Server failed to start:", err)
	}
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
