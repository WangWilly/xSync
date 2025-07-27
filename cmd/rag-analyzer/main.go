package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/WangWilly/xSync/migration/automigrate"
	"github.com/WangWilly/xSync/pkgs/clipkg/helpers/syscfghelper"
	"github.com/WangWilly/xSync/pkgs/commonpkg/clients/chromatokenclient"
	"github.com/WangWilly/xSync/pkgs/commonpkg/database"
	"github.com/WangWilly/xSync/pkgs/commonpkg/repos/tokenrepo"
	"github.com/WangWilly/xSync/pkgs/commonpkg/repos/tweetrepo"
	"github.com/WangWilly/xSync/pkgs/ragpkg/analyzer"
)

func main() {
	ctx := context.Background()
	log.Println("Starting xSync RAG Analyzer...")

	////////////////////////////////////////////////////////////////////////////

	db, err := database.ConnectWithConfig(
		syscfghelper.GetDefaultDbConfig(),
	)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()
	log.Println("Automatically migrating database...")
	if err := automigrate.AutoMigrateUp(
		automigrate.AutoMigrateConfig{SqlxDB: db},
	); err != nil {
		log.Fatalf("Failed to create database tables: %v", err)
	}

	////////////////////////////////////////////////////////////////////////////

	// TODO:
	chromaURL := getEnvOrDefault("CHROMA_URL", "http://localhost:8000")

	// Initialize ChromaDB client
	log.Println("Connecting to ChromaDB...")
	chromaTokenClient, err := chromatokenclient.New(chromaURL)
	if err != nil {
		log.Fatalf("Failed to connect to ChromaDB: %v", err)
	}
	defer chromaTokenClient.Close()

	////////////////////////////////////////////////////////////////////////////

	tweetRepo := tweetrepo.New()
	tokenRepo := tokenrepo.New()
	ragAnalyzer := analyzer.NewRAGAnalyzer(
		db,
		chromaTokenClient,
		tweetRepo,
		tokenRepo,
	)

	////////////////////////////////////////////////////////////////////////////

	// Set up graceful shutdown
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	////////////////////////////////////////////////////////////////////////////

	// Handle shutdown signals
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-signalChan
		log.Println("Received shutdown signal, stopping...")
		cancel()
	}()

	////////////////////////////////////////////////////////////////////////////

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
