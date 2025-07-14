package server

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/WangWilly/xSync/pkgs/serverpkg/serverdto"
)

// handleTweets serves tweet data as JSON
func (s *Server) handleTweets(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Path[len("/tweets/"):]
	if userID == "" {
		http.Error(w, "User ID required", http.StatusBadRequest)
		return
	}

	id, err := strconv.Atoi(userID)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	// Get tweets from dumper
	tweets := s.dumper.GetTweetsByEntityId(id)
	var tweetData []map[string]interface{}

	for _, tweet := range tweets {
		tweetData = append(tweetData, map[string]interface{}{
			"id":         tweet.Id,
			"text":       tweet.Text,
			"created_at": tweet.CreatedAt,
			"urls":       tweet.Urls,
			"creator":    tweet.Creator,
		})
	}

	user, err := s.userRepo.GetById(s.db, uint64(id))
	if err != nil {
		http.Error(w, "Failed to get user: "+err.Error(), http.StatusInternalServerError)
		return
	}

	userName := "Unknown"
	if user != nil {
		userName = user.ScreenName
	}

	data := serverdto.TweetData{
		UserName: userName,
		Tweets:   tweetData,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

// handleAPITweets serves tweets with media data as JSON
func (s *Server) handleAPITweets(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Path[len("/api/tweets/"):]
	if userID == "" {
		http.Error(w, "User ID required", http.StatusBadRequest)
		return
	}

	id, err := strconv.ParseUint(userID, 10, 64)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	// Get tweets with media from database
	tweetsWithMedia, err := s.tweetRepo.GetWithMedia(s.db, id)
	if err != nil {
		http.Error(w, "Failed to get tweets: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tweetsWithMedia)
}

// handleTweetsWithMedia serves the tweets with media template page
func (s *Server) handleTweetsWithMedia(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Path[len("/tweets-media/"):]
	if userID == "" {
		http.Error(w, "User ID required", http.StatusBadRequest)
		return
	}

	id, err := strconv.ParseUint(userID, 10, 64)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	// Get user info
	user, err := s.userRepo.GetById(s.db, id)
	if err != nil {
		http.Error(w, "Failed to get user: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if user == nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Get tweets with media
	tweetsWithMedia, err := s.tweetRepo.GetWithMedia(s.db, id)
	if err != nil {
		http.Error(w, "Failed to get tweets with media: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Group media by tweet and convert paths
	tweetMediaMap := make(map[int64][]string)
	for _, tweet := range tweetsWithMedia {
		if tweetID, ok := tweet["id"].(int64); ok {
			if mediaLocation, ok := tweet["media_location"].(string); ok && mediaLocation != "" {
				// Convert absolute path to relative path for serving
				relativePath := s.convertToRelativePath(mediaLocation)
				tweetMediaMap[tweetID] = append(tweetMediaMap[tweetID], relativePath)
			}
		}
	}

	var tweetsData []serverdto.TweetWithMedia
	processedTweets := make(map[int64]bool)

	for _, tweet := range tweetsWithMedia {
		if tweetID, ok := tweet["id"].(int64); ok {
			if !processedTweets[tweetID] {
				processedTweets[tweetID] = true

				tweetData := serverdto.TweetWithMedia{
					ID:         tweetID,
					Content:    tweet["content"].(string),
					TweetTime:  tweet["tweet_time"].(time.Time),
					MediaFiles: tweetMediaMap[tweetID],
					MediaCount: len(tweetMediaMap[tweetID]),
				}
				tweetsData = append(tweetsData, tweetData)
			}
		}
	}

	data := map[string]interface{}{
		"user":   user,
		"tweets": tweetsData,
	}

	w.Header().Set("Content-Type", "text/html")
	if err := s.templates.ExecuteTemplate(w, "tweets-media.html", data); err != nil {
		http.Error(w, "Failed to render template: "+err.Error(), http.StatusInternalServerError)
	}
}
