package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/WangWilly/xSync/pkgs/database"
	"github.com/WangWilly/xSync/pkgs/downloading"
	"github.com/WangWilly/xSync/pkgs/model"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

// Server represents the web server for displaying tweet data
type Server struct {
	db        *sqlx.DB
	dumper    *downloading.TweetDumper
	templates *template.Template
	port      string
}

// UserStats represents user statistics for display
type UserStats struct {
	User           *model.User
	Entities       []*model.UserEntity
	TotalMedias    int
	LatestActivity time.Time
}

// DashboardData represents data for the dashboard
type DashboardData struct {
	Users       []*UserStats
	TotalUsers  int
	TotalTweets int
	TotalMedias int
	LastUpdated time.Time
}

// TweetData represents tweet data for display
type TweetData struct {
	UserName string
	Tweets   []map[string]interface{}
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	dbPath := "./conf/data/xSync.db"

	server, err := NewServer(dbPath, port)
	if err != nil {
		log.Fatal("Failed to create server:", err)
	}
	defer server.Close()

	log.Printf("Starting server on port %s", port)
	log.Printf("Database path: %s", dbPath)
	log.Printf("Open http://localhost:%s to view the dashboard", port)

	if err := server.Start(); err != nil {
		log.Fatal("Server failed to start:", err)
	}
}

func NewServer(dbPath, port string) (*Server, error) {
	// Connect to database
	db, err := database.ConnectDatabase(dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Load tweet dumper
	dumper := downloading.NewDumper()
	dumpPath := filepath.Join(filepath.Dir(dbPath), "error.json")
	if err := dumper.Load(dumpPath); err != nil {
		log.Printf("Warning: Failed to load tweet dump file: %v", err)
	}
	// Parse templates
	templates, err := template.New("").Funcs(template.FuncMap{
		"formatTime": func(t time.Time) string {
			if t.IsZero() {
				return "Never"
			}
			return t.Format("2006-01-02 15:04:05")
		},
		"formatTimeAgo": func(t time.Time) string {
			if t.IsZero() {
				return "Never"
			}
			return time.Since(t).Round(time.Minute).String() + " ago"
		},
	}).ParseGlob("./cmd/server/templates/*.html")
	if err != nil {
		return nil, fmt.Errorf("failed to parse templates: %w", err)
	}

	fmt.Println(templates.DefinedTemplates())

	return &Server{
		db:        db,
		dumper:    dumper,
		templates: templates,
		port:      port,
	}, nil
}

func (s *Server) Close() {
	if s.db != nil {
		s.db.Close()
	}
}

func (s *Server) Start() error {
	http.HandleFunc("/", s.handleDashboard)
	http.HandleFunc("/user/", s.handleUser)
	http.HandleFunc("/tweets/", s.handleTweets)
	http.HandleFunc("/media/", s.handleMedia)
	http.HandleFunc("/api/stats", s.handleAPIStats)
	http.HandleFunc("/api/tweets/", s.handleAPITweets)
	http.HandleFunc("/api/media/", s.handleAPIMedia)
	http.HandleFunc("/static/", s.handleStatic)

	return http.ListenAndServe(":"+s.port, nil)
}

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

func (s *Server) handleUser(w http.ResponseWriter, r *http.Request) {
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

	user, err := database.GetUserById(s.db, id)
	if err != nil {
		http.Error(w, "Failed to get user: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if user == nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	entities, err := s.getUserEntities(id)
	if err != nil {
		http.Error(w, "Failed to get user entities: "+err.Error(), http.StatusInternalServerError)
		return
	}

	data := UserStats{
		User:     user,
		Entities: entities,
	}

	// Calculate stats
	for _, entity := range entities {
		if entity.MediaCount.Valid {
			data.TotalMedias += int(entity.MediaCount.Int32)
		}
		if entity.LatestReleaseTime.Valid && entity.LatestReleaseTime.Time.After(data.LatestActivity) {
			data.LatestActivity = entity.LatestReleaseTime.Time
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

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

	user, err := database.GetUserById(s.db, uint64(id))
	if err != nil {
		http.Error(w, "Failed to get user: "+err.Error(), http.StatusInternalServerError)
		return
	}

	userName := "Unknown"
	if user != nil {
		userName = user.ScreenName
	}

	data := TweetData{
		UserName: userName,
		Tweets:   tweetData,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

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
	medias, err := database.GetMediasByUserId(s.db, id)
	if err != nil {
		http.Error(w, "Failed to get media: "+err.Error(), http.StatusInternalServerError)
		return
	}

	user, err := database.GetUserById(s.db, id)
	if err != nil {
		http.Error(w, "Failed to get user: "+err.Error(), http.StatusInternalServerError)
		return
	}

	userName := "Unknown"
	if user != nil {
		userName = user.ScreenName
	}

	data := map[string]interface{}{
		"user_name": userName,
		"medias":    medias,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func (s *Server) handleAPIStats(w http.ResponseWriter, r *http.Request) {
	data, err := s.getDashboardData()
	if err != nil {
		http.Error(w, "Failed to get stats: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

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
	tweetsWithMedia, err := database.GetTweetsWithMedia(s.db, id)
	if err != nil {
		http.Error(w, "Failed to get tweets: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tweetsWithMedia)
}

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
	medias, err := database.GetMediasByUserId(s.db, id)
	if err != nil {
		http.Error(w, "Failed to get media: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(medias)
}

func (s *Server) handleStatic(w http.ResponseWriter, r *http.Request) {
	// Serve static files (CSS, JS, etc.)
	http.ServeFile(w, r, r.URL.Path[1:])
}

func (s *Server) getDashboardData() (*DashboardData, error) {
	users, err := s.getAllUsers()
	if err != nil {
		return nil, err
	}

	var userStats []*UserStats
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

		stats := &UserStats{
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

	return &DashboardData{
		Users:       userStats,
		TotalUsers:  len(users),
		TotalTweets: totalTweets,
		TotalMedias: totalMedias,
		LastUpdated: time.Now(),
	}, nil
}

func (s *Server) getAllUsers() ([]*model.User, error) {
	var users []*model.User
	err := s.db.Select(&users, "SELECT * FROM users ORDER BY screen_name")
	return users, err
}

func (s *Server) getUserEntities(userID uint64) ([]*model.UserEntity, error) {
	var entities []*model.UserEntity
	err := s.db.Select(&entities, "SELECT * FROM user_entities WHERE user_id = ? ORDER BY name", userID)
	return entities, err
}
