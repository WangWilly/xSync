package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/WangWilly/xSync/pkgs/commonpkg/clients/chromatokenclient"
	"github.com/WangWilly/xSync/pkgs/commonpkg/database"
	"github.com/WangWilly/xSync/pkgs/commonpkg/repos/tokenrepo"
	"github.com/WangWilly/xSync/pkgs/commonpkg/repos/tweetrepo"
	"github.com/WangWilly/xSync/pkgs/ragpkg/analyzer"
)

func main() {
	ctx := context.Background()

	// Database configuration - you may want to move this to config file
	dbConfig := struct {
		Host     string
		Port     string
		User     string
		Password string
		DBName   string
	}{
		Host:     "localhost",
		Port:     "5432",
		User:     "xsync-2025",
		Password: "xsync-2025",
		DBName:   "xsync-2025",
	}

	chromaURL := "http://localhost:8000"

	log.Println("Starting xSync RAG Analyzer...")

	// Initialize PostgreSQL connection
	log.Println("Connecting to PostgreSQL...")
	db, err := database.ConnectPostgres(dbConfig.Host, dbConfig.Port, dbConfig.User, dbConfig.Password, dbConfig.DBName)
	if err != nil {
		log.Fatalf("Failed to connect to PostgreSQL: %v", err)
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
