package server

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/WangWilly/xSync/pkgs/serverpkg/serverdto"
)

// handleUser serves user information as JSON
func (s *Server) handleUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID := r.URL.Path[len("/user/"):]
	if userID == "" {
		http.Error(w, "User ID required", http.StatusBadRequest)
		return
	}

	id, err := strconv.ParseUint(userID, 10, 64)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	user, err := s.userRepo.GetById(ctx, s.db, id)
	if err != nil {
		http.Error(w, "Failed to get user: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if user == nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	userEntity, err := s.userEntityRepo.GetByTwitterId(ctx, s.db, user.Id)
	if err != nil {
		http.Error(w, "Failed to get user entities: "+err.Error(), http.StatusInternalServerError)
		return
	}

	data := serverdto.UserStats{
		User:   user,
		Entity: userEntity,
	}

	data.TotalMedias += int(userEntity.MediaCount.Int32)
	if userEntity.LatestReleaseTime.Valid && userEntity.LatestReleaseTime.Time.After(data.LatestActivity) {
		data.LatestActivity = userEntity.LatestReleaseTime.Time
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}
