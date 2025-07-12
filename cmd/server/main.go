package main

import (
	"log"
	"os"

	"github.com/WangWilly/xSync/pkgs/server"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	dbPath := "./conf/data/xSync.db"

	srv, err := server.NewServer(dbPath, port)
	if err != nil {
		log.Fatal("Failed to create server:", err)
	}
	defer srv.Close()

	log.Printf("Starting server on port %s", port)
	log.Printf("Database path: %s", dbPath)
	log.Printf("Open http://localhost:%s to view the dashboard", port)

	if err := srv.Start(); err != nil {
		log.Fatal("Server failed to start:", err)
	}
}
