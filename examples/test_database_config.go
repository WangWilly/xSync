package main

import (
	"fmt"
	"log"
	"os"

	"github.com/WangWilly/xSync/pkgs/clipkg/config"
	"github.com/WangWilly/xSync/pkgs/commonpkg/database"
)

func main() {
	fmt.Println("Testing database configurations...")

	// Test SQLite configuration
	fmt.Println("\n1. Testing SQLite connection...")
	sqliteConfig := config.DatabaseConfig{
		Type: "sqlite",
		Path: "./test.db",
	}

	db, err := database.ConnectWithConfig(sqliteConfig)
	if err != nil {
		log.Printf("SQLite connection failed: %v", err)
	} else {
		fmt.Println("✓ SQLite connection successful")
		db.Close()
		// Clean up test database
		os.Remove("./test.db")
	}

	// Test PostgreSQL configuration (only if environment variables are set)
	fmt.Println("\n2. Testing PostgreSQL connection...")
	pgConfig := config.DatabaseConfig{
		Type:     "postgres",
		Host:     "localhost",
		Port:     "5432",
		User:     "xsync",
		Password: "xsync_password",
		DBName:   "xsync",
	}

	db, err = database.ConnectWithConfig(pgConfig)
	if err != nil {
		log.Printf("PostgreSQL connection failed: %v (this is expected if PostgreSQL is not running)", err)
	} else {
		fmt.Println("✓ PostgreSQL connection successful")
		db.Close()
	}

	fmt.Println("\nDatabase configuration test completed!")
}
