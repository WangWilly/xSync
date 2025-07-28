package main

import (
	"log"
	"os"

	"github.com/WangWilly/xSync/migration/automigrate"
	"github.com/WangWilly/xSync/pkgs/clipkg/helpers/syscfghelper"
	"github.com/WangWilly/xSync/pkgs/commonpkg/database"
	"github.com/WangWilly/xSync/pkgs/serverpkg/server"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	////////////////////////////////////////////////////////////////////////////

	db, err := database.ConnectWithConfig(
		syscfghelper.GetDefaultDbConfig(),
	)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	if err := automigrate.AutoMigrateUp(
		automigrate.AutoMigrateConfig{SqlxDB: db},
	); err != nil {
		log.Fatalf("Failed to create database tables: %v", err)
	}

	////////////////////////////////////////////////////////////////////////////

	srv, err := server.NewServerWithConfig(db, port)
	if err != nil {
		log.Fatal("Failed to create server:", err)
	}
	defer srv.Close()

	////////////////////////////////////////////////////////////////////////////

	log.Printf("Open http://localhost:%s to view the dashboard", port)
	if err := srv.Start(); err != nil {
		log.Fatal("Server failed to start:", err)
	}
}
