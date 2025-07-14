package server

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/WangWilly/xSync/pkgs/serverpkg/serverdto"
)

// handleMedia serves media data as JSON
func (s *Server) handleMedia(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Path[len("/media/"):]
	if userID == "" {
		http.Error(w, "User ID required", http.StatusBadRequest)
		return
	}

	id, err := strconv.ParseUint(userID, 10, 64)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	// Get media from database
	medias, err := s.mediaRepo.GetByUserId(s.db, id)
	if err != nil {
		http.Error(w, "Failed to get media: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Convert absolute paths to relative paths for serving
	for _, media := range medias {
		media.Location = s.convertToRelativePath(media.Location)
	}

	user, err := s.userRepo.GetById(s.db, id)
	if err != nil {
		http.Error(w, "Failed to get user: "+err.Error(), http.StatusInternalServerError)
		return
	}

	userName := "Unknown"
	if user != nil {
		userName = user.ScreenName
	}

	data := serverdto.MediaResponse{
		UserName: userName,
		Medias:   medias,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

// handleAPIMedia serves media data as JSON for API endpoints
func (s *Server) handleAPIMedia(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Path[len("/api/media/"):]
	if userID == "" {
		http.Error(w, "User ID required", http.StatusBadRequest)
		return
	}

	id, err := strconv.ParseUint(userID, 10, 64)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	// Get media from database
	medias, err := s.mediaRepo.GetByUserId(s.db, id)
	if err != nil {
		http.Error(w, "Failed to get media: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Convert absolute paths to relative paths for serving
	for _, media := range medias {
		media.Location = s.convertToRelativePath(media.Location)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(medias)
}
