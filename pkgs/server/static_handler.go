package server

import (
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// handleStatic serves static files (CSS, JS, images, etc.)
func (s *Server) handleStatic(w http.ResponseWriter, r *http.Request) {
	// Serve static files (CSS, JS, etc.)
	staticPath := r.URL.Path[len("/static/"):]
	fullPath := filepath.Join("./cmd/server/static", staticPath)

	// Check if file exists
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	// Set proper content type for SVG files
	if filepath.Ext(staticPath) == ".svg" {
		w.Header().Set("Content-Type", "image/svg+xml")
	}

	http.ServeFile(w, r, fullPath)
}

// handleFiles serves user media files
func (s *Server) handleFiles(w http.ResponseWriter, r *http.Request) {
	// Extract the file path from the URL
	filePath := r.URL.Path[len("/files/"):]
	if filePath == "" {
		http.Error(w, "File path required", http.StatusBadRequest)
		return
	}

	// URL decode the file path
	decodedPath, err := url.QueryUnescape(filePath)
	if err != nil {
		http.Error(w, "Invalid file path", http.StatusBadRequest)
		return
	}

	// Construct the full file path
	fullPath := filepath.Join("./conf/users", decodedPath)

	// Check if file exists
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	// Set proper content type for media files
	ext := strings.ToLower(filepath.Ext(decodedPath))
	switch ext {
	case ".mp4":
		w.Header().Set("Content-Type", "video/mp4")
	case ".jpg", ".jpeg":
		w.Header().Set("Content-Type", "image/jpeg")
	case ".png":
		w.Header().Set("Content-Type", "image/png")
	case ".gif":
		w.Header().Set("Content-Type", "image/gif")
	}

	// Serve the file
	http.ServeFile(w, r, fullPath)
}
