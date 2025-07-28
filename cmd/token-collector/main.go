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
	"github.com/WangWilly/xSync/pkgs/commonpkg/clients/juptokenclient"
	"github.com/WangWilly/xSync/pkgs/commonpkg/database"
	"github.com/WangWilly/xSync/pkgs/commonpkg/repos/tokenembedding"
	"github.com/WangWilly/xSync/pkgs/commonpkg/repos/tokenrepo"
	"github.com/WangWilly/xSync/pkgs/commonpkg/services"
)

func main() {
	ctx := context.Background()
	log.Println("Starting xSync Token Collector...")

	////////////////////////////////////////////////////////////////////////////

	db, err := database.ConnectWithConfig(
		syscfghelper.GetDefaultDbConfig(),
	)
	if err != nil {
		log.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}
	defer db.Close()
	if err := automigrate.AutoMigrateUp(
		automigrate.AutoMigrateConfig{SqlxDB: db},
	); err != nil {
		log.Fatalf("Failed to create database tables: %v", err)
	}

	////////////////////////////////////////////////////////////////////////////

	chromaURL := "http://localhost:8000"
	chromaTokenClient, err := chromatokenclient.New(chromaURL)
	if err != nil {
		log.Fatalf("Failed to connect to ChromaDB: %v", err)
	}
	defer chromaTokenClient.Close()

	////////////////////////////////////////////////////////////////////////////

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

	////////////////////////////////////////////////////////////////////////////

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-signalChan
		log.Println("Received shutdown signal, stopping...")
		cancel()
	}()

	////////////////////////////////////////////////////////////////////////////

	log.Println("Starting token collection process...")
	err = tokenService.CollectAndStoreTokens(ctx)
	if err != nil {
		log.Fatalf("Failed to collect and store tokens: %v", err)
	}

	////////////////////////////////////////////////////////////////////////////

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
