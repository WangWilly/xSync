package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/WangWilly/xSync/pkgs/clipkg/config"
	"github.com/WangWilly/xSync/pkgs/commonpkg/clients/chromatokenclient"
	"github.com/WangWilly/xSync/pkgs/commonpkg/database"
	"github.com/WangWilly/xSync/pkgs/commonpkg/repos/tokenrepo"
	"github.com/WangWilly/xSync/pkgs/commonpkg/repos/tweetrepo"
	"github.com/WangWilly/xSync/pkgs/ragpkg/analyzer"
)

func main() {
	ctx := context.Background()

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

	// If no configuration loaded, check environment variables or use defaults
	if dbConfig.Type == "" {
		dbType := os.Getenv("DB_TYPE")
		if dbType == "" {
			dbType = "postgres" // RAG analyzer defaults to PostgreSQL
		}
		dbConfig.Type = dbType

		switch dbType {
		case "postgres", "postgresql":
			dbConfig.Host = getEnvOrDefault("DB_HOST", "localhost")
			dbConfig.Port = getEnvOrDefault("DB_PORT", "5432")
			dbConfig.User = getEnvOrDefault("DB_USER", "xsync-2025")
			dbConfig.Password = getEnvOrDefault("DB_PASSWORD", "xsync-2025")
			dbConfig.DBName = getEnvOrDefault("DB_NAME", "xsync-2025")
		default:
			// SQLite fallback
			dbPath := getEnvOrDefault("DB_PATH", "./conf/data/xSync.db")
			dbConfig.Path = dbPath
		}
	}

	chromaURL := getEnvOrDefault("CHROMA_URL", "http://localhost:8000")

	log.Println("Starting xSync RAG Analyzer...")

	// Initialize database connection
	log.Printf("Connecting to %s database...", dbConfig.Type)
	db, err := database.ConnectWithConfig(dbConfig)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Initialize ChromaDB client
	log.Println("Connecting to ChromaDB...")
	chromaTokenClient, err := chromatokenclient.New(chromaURL)
	if err != nil {
		log.Fatalf("Failed to connect to ChromaDB: %v", err)
	}
	defer chromaTokenClient.Close()

	// Initialize repositories
	tweetRepo := tweetrepo.New()
	tokenRepo := tokenrepo.New()

	// Initialize RAG analyzer service
	ragAnalyzer := analyzer.NewRAGAnalyzer(
		db,
		chromaTokenClient,
		tweetRepo,
		tokenRepo,
	)

	// Set up graceful shutdown
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Handle shutdown signals
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-signalChan
		log.Println("Received shutdown signal, stopping...")
		cancel()
	}()

	// Start the RAG analysis process
	log.Println("Starting RAG analysis process...")
	err = ragAnalyzer.StartContinuousAnalysis(ctx)
	if err != nil {
		log.Fatalf("Failed to start RAG analysis: %v", err)
	}

	log.Println("RAG analysis completed successfully!")
	log.Println("Application stopped.")
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
