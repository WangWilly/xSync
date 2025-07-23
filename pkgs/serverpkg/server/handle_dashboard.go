package server

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/WangWilly/xSync/pkgs/serverpkg/serverdto"
)

// handleDashboard serves the main dashboard page
func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	data, err := s.getDashboardData(ctx)
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
	ctx := r.Context()

	data, err := s.getDashboardData(ctx)
	if err != nil {
		http.Error(w, "Failed to get stats: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

// getDashboardData compiles all dashboard statistics and user information
func (s *Server) getDashboardData(ctx context.Context) (*serverdto.DashboardData, error) {
	users, err := s.userRepo.ListAll(ctx, s.db)
	if err != nil {
		return nil, err
	}

	totalTweets, err := s.tweetRepo.CountAll(ctx, s.db)
	if err != nil {
		return nil, err
	}

	totalMedias, err := s.mediaRepo.CountAll(ctx, s.db)
	if err != nil {
		return nil, err
	}

	var userStats []*serverdto.UserStats
	for _, user := range users {
		userEntity, err := s.userEntityRepo.GetByTwitterId(ctx, s.db, user.Id)
		if err != nil {
			continue
		}

		stats := &serverdto.UserStats{
			User:   user,
			Entity: userEntity,
		}

		userMedias, err := s.mediaRepo.CountByUserId(ctx, s.db, user.Id)
		if err != nil {
			continue
		}
		stats.TotalMedias = int(userMedias)

		if userEntity.LatestReleaseTime.Valid && userEntity.LatestReleaseTime.Time.After(stats.LatestActivity) {
			stats.LatestActivity = userEntity.LatestReleaseTime.Time
		}

		userStats = append(userStats, stats)
	}

	return &serverdto.DashboardData{
		Users:       userStats,
		TotalUsers:  len(users),
		TotalTweets: int(totalTweets),
		TotalMedias: int(totalMedias),
		LastUpdated: time.Now(),
	}, nil
}
