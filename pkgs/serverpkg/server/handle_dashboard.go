package server

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/WangWilly/xSync/pkgs/serverpkg/serverdto"
)

// handleDashboard serves the main dashboard page
func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	data, err := s.getDashboardData()
	if err != nil {
		http.Error(w, "Failed to get dashboard data: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	if err := s.templates.ExecuteTemplate(w, "dashboard.html", data); err != nil {
		http.Error(w, "Failed to render template: "+err.Error(), http.StatusInternalServerError)
	}
}

// handleAPIStats serves dashboard statistics as JSON
func (s *Server) handleAPIStats(w http.ResponseWriter, r *http.Request) {
	data, err := s.getDashboardData()
	if err != nil {
		http.Error(w, "Failed to get stats: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

// getDashboardData compiles all dashboard statistics and user information
func (s *Server) getDashboardData() (*serverdto.DashboardData, error) {
	users, err := s.getAllUsers()
	if err != nil {
		return nil, err
	}

	var userStats []*serverdto.UserStats
	totalTweets := s.dumper.Count()
	totalMedias := 0

	// Get database counts
	var dbTotalTweets int
	var dbTotalMedias int
	s.db.Get(&dbTotalTweets, "SELECT COUNT(*) FROM tweets")
	s.db.Get(&dbTotalMedias, "SELECT COUNT(*) FROM medias")

	// Use database counts if available, otherwise fallback to dumper
	if dbTotalTweets > 0 {
		totalTweets = dbTotalTweets
	}
	if dbTotalMedias > 0 {
		totalMedias = dbTotalMedias
	}

	for _, user := range users {
		entities, err := s.getUserEntities(user.Id)
		if err != nil {
			continue
		}

		stats := &serverdto.UserStats{
			User:     user,
			Entities: entities,
		}

		// Get user-specific counts from database
		var userTweets int
		var userMedias int
		s.db.Get(&userTweets, "SELECT COUNT(*) FROM tweets WHERE user_id = ?", user.Id)
		s.db.Get(&userMedias, "SELECT COUNT(*) FROM medias WHERE user_id = ?", user.Id)

		stats.TotalMedias = userMedias

		for _, entity := range entities {
			if entity.LatestReleaseTime.Valid && entity.LatestReleaseTime.Time.After(stats.LatestActivity) {
				stats.LatestActivity = entity.LatestReleaseTime.Time
			}
		}

		userStats = append(userStats, stats)
	}

	return &serverdto.DashboardData{
		Users:       userStats,
		TotalUsers:  len(users),
		TotalTweets: totalTweets,
		TotalMedias: totalMedias,
		LastUpdated: time.Now(),
	}, nil
}
