// Package server provides a web server for displaying xSync tweet data and media.
//
// This package contains a modular HTTP server implementation that serves:
// - Dashboard views with user statistics
// - User information and media galleries
// - Tweet data with associated media files
// - Static file serving for CSS, JS, and media files
//
// The server is organized into focused handler files:
//   - dashboard_handler.go: Main dashboard and statistics
//   - user_handler.go: User-specific data
//   - tweets_handler.go: Tweet data and tweets with media
//   - media_handler.go: Media file information
//   - static_handler.go: Static and user file serving
//
// Usage:
//
//	srv, err := server.NewServer(dbPath, port)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer srv.Close()
//
//	if err := srv.Start(); err != nil {
//	    log.Fatal(err)
//	}
//
// The server integrates with:
//   - SQLite database for user and media data
//   - HTML templates for web page rendering
//   - Tweet dumper for tweet data access
//   - File system for media file serving
package server
