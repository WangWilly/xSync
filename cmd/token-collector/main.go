package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/WangWilly/xSync/pkgs/commonpkg/clients/chromatokenclient"
	"github.com/WangWilly/xSync/pkgs/commonpkg/clients/juptokenclient"
	"github.com/WangWilly/xSync/pkgs/commonpkg/database"
	"github.com/WangWilly/xSync/pkgs/commonpkg/repos/tokenembedding"
	"github.com/WangWilly/xSync/pkgs/commonpkg/repos/tokenrepo"
	"github.com/WangWilly/xSync/pkgs/commonpkg/services"
)

func main() {
	ctx := context.Background()

	// Database configuration
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

	log.Println("Starting xSync Token Collector...")

	// Initialize PostgreSQL connection
	log.Println("Connecting to PostgreSQL...")
	db, err := database.ConnectPostgres(dbConfig.Host, dbConfig.Port, dbConfig.User, dbConfig.Password, dbConfig.DBName)
	if err != nil {
		log.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}
	defer db.Close()

	// Create database tables
	log.Println("Creating database tables...")
	err = database.CreateTokenTables(db)
	if err != nil {
		log.Fatalf("Failed to create database tables: %v", err)
	}

	// Initialize ChromaDB client
	log.Println("Connecting to ChromaDB...")
	chromaTokenClient, err := chromatokenclient.New(chromaURL)
	if err != nil {
		log.Fatalf("Failed to connect to ChromaDB: %v", err)
	}
	defer chromaTokenClient.Close()

	jupiterClient := juptokenclient.New()
	tokenRepo := tokenrepo.New()
	tokenEmbeddingRepo := tokenembedding.New()
	tokenService := services.NewTokenService(
		services.Config{},
		db,
		jupiterClient,
		chromaTokenClient,
		tokenRepo,
		tokenEmbeddingRepo,
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

	// Start the token collection process
	log.Println("Starting token collection process...")

	// Run the collection process
	err = tokenService.CollectAndStoreTokens(ctx)
	if err != nil {
		log.Fatalf("Failed to collect and store tokens: %v", err)
	}

	// Get and display statistics
	log.Println("Getting token statistics...")
	stats, err := tokenService.GetTokenStats(ctx)
	if err != nil {
		log.Printf("Failed to get token stats: %v", err)
	} else {
		log.Printf("Token Statistics: %+v", stats)
	}

	// Demonstrate search functionality
	log.Println("Demonstrating search functionality...")

	// Search using PostgreSQL
	log.Println("Searching for 'SOL' tokens in PostgreSQL...")
	pgResults, err := tokenService.SearchTokens(ctx, "SOL", false)
	if err != nil {
		log.Printf("Failed to search tokens in PostgreSQL: %v", err)
	} else {
		log.Printf("PostgreSQL search results: %+v", pgResults)
	}

	// Search using ChromaDB (semantic search)
	log.Println("Searching for 'SOL' tokens in ChromaDB...")
	chromaResults, err := tokenService.SearchTokens(ctx, "SOL", true)
	if err != nil {
		log.Printf("Failed to search tokens in ChromaDB: %v", err)
	} else {
		log.Printf("ChromaDB search results: %+v", chromaResults)
	}

	// Keep the application running until cancelled
	log.Println("Token collection completed successfully!")
	log.Println("Application is running. Press Ctrl+C to stop.")

	// Wait for cancellation
	<-ctx.Done()
	log.Println("Application stopped.")
}
