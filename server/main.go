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
	User           *database.User
	Entities       []*database.UserEntity
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

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		// Try to find the database in the default location
		homePath := os.Getenv("HOME")
		if homePath == "" {
			homePath = os.Getenv("APPDATA")
		}
		dbPath = filepath.Join(homePath, ".x_sync", "tmd.db")
	}

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
	db, err := connectDatabase(dbPath)
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
	}).ParseGlob("templates/*.html")

	if err != nil {
		// If templates don't exist, create inline templates
		templates = template.Must(template.New("dashboard").Funcs(template.FuncMap{
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
		}).Parse(dashboardTemplate))
	}

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
	http.HandleFunc("/api/stats", s.handleAPIStats)
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
	if err := s.templates.ExecuteTemplate(w, "dashboard", data); err != nil {
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

func (s *Server) handleAPIStats(w http.ResponseWriter, r *http.Request) {
	data, err := s.getDashboardData()
	if err != nil {
		http.Error(w, "Failed to get stats: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
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

	for _, user := range users {
		entities, err := s.getUserEntities(user.Id)
		if err != nil {
			continue
		}

		stats := &UserStats{
			User:     user,
			Entities: entities,
		}

		for _, entity := range entities {
			if entity.MediaCount.Valid {
				count := int(entity.MediaCount.Int32)
				stats.TotalMedias += count
				totalMedias += count
			}
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

func (s *Server) getAllUsers() ([]*database.User, error) {
	var users []*database.User
	err := s.db.Select(&users, "SELECT * FROM users ORDER BY screen_name")

	// TODO:
	fmt.Println("Total users found:", len(users))
	for _, user := range users {
		// Load user entities for each user
		fmt.Printf("User: %s (%d)\n", user.ScreenName, user.Id)
	}
	//
	return users, err
}

func (s *Server) getUserEntities(userID uint64) ([]*database.UserEntity, error) {
	var entities []*database.UserEntity
	err := s.db.Select(&entities, "SELECT * FROM user_entities WHERE user_id = ? ORDER BY name", userID)
	return entities, err
}

func connectDatabase(path string) (*sqlx.DB, error) {
	dsn := fmt.Sprintf("file:%s?_journal_mode=WAL&busy_timeout=2147483647", path)
	db, err := sqlx.Connect("sqlite3", dsn)
	if err != nil {
		return nil, err
	}
	return db, nil
}

const dashboardTemplate = `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>xSync Tweet Dashboard</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            margin: 0;
            padding: 20px;
            background-color: #f5f5f5;
        }
        .container {
            max-width: 1200px;
            margin: 0 auto;
            background: white;
            border-radius: 8px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
            padding: 20px;
        }
        h1 {
            color: #1da1f2;
            text-align: center;
            margin-bottom: 30px;
        }
        .stats-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
            gap: 20px;
            margin-bottom: 30px;
        }
        .stat-card {
            background: #f8f9fa;
            padding: 20px;
            border-radius: 8px;
            text-align: center;
            border: 1px solid #e9ecef;
        }
        .stat-value {
            font-size: 2em;
            font-weight: bold;
            color: #1da1f2;
        }
        .stat-label {
            color: #6c757d;
            margin-top: 5px;
        }
        .users-grid {
            display: grid;
            grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
            gap: 20px;
        }
        .user-card {
            background: #fff;
            border: 1px solid #e1e8ed;
            border-radius: 8px;
            padding: 20px;
            transition: box-shadow 0.2s;
        }
        .user-card:hover {
            box-shadow: 0 4px 8px rgba(0,0,0,0.1);
        }
        .user-header {
            display: flex;
            align-items: center;
            margin-bottom: 15px;
        }
        .user-name {
            font-weight: bold;
            color: #14171a;
        }
        .user-screen-name {
            color: #657786;
            margin-left: 5px;
        }
        .user-stats {
            display: flex;
            justify-content: space-between;
            margin-bottom: 10px;
        }
        .user-stat {
            text-align: center;
        }
        .user-stat-value {
            font-weight: bold;
            color: #1da1f2;
        }
        .user-stat-label {
            font-size: 0.9em;
            color: #657786;
        }
        .last-activity {
            font-size: 0.9em;
            color: #657786;
            margin-top: 10px;
        }
        .protected-badge {
            background: #ffad1f;
            color: white;
            padding: 2px 8px;
            border-radius: 12px;
            font-size: 0.8em;
            margin-left: 10px;
        }
        .refresh-btn {
            background: #1da1f2;
            color: white;
            border: none;
            padding: 10px 20px;
            border-radius: 20px;
            cursor: pointer;
            margin-bottom: 20px;
        }
        .refresh-btn:hover {
            background: #1991db;
        }
        .entity-list {
            margin-top: 10px;
        }
        .entity-item {
            font-size: 0.9em;
            color: #657786;
            margin: 2px 0;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>🐦 xSync Tweet Dashboard</h1>
        
        <button class="refresh-btn" onclick="location.reload()">🔄 Refresh Data</button>
        
        <div class="stats-grid">
            <div class="stat-card">
                <div class="stat-value">{{.TotalUsers}}</div>
                <div class="stat-label">Total Users</div>
            </div>
            <div class="stat-card">
                <div class="stat-value">{{.TotalTweets}}</div>
                <div class="stat-label">Total Tweets</div>
            </div>
            <div class="stat-card">
                <div class="stat-value">{{.TotalMedias}}</div>
                <div class="stat-label">Total Media Files</div>
            </div>
        </div>

        <div class="users-grid">
            {{range .Users}}
            <div class="user-card">
                <div class="user-header">
                    <div>
                        <span class="user-name">{{.User.Name}}</span>
                        <span class="user-screen-name">@{{.User.ScreenName}}</span>
                        {{if .User.IsProtected}}
                        <span class="protected-badge">🔒 Protected</span>
                        {{end}}
                    </div>
                </div>
                
                <div class="user-stats">
                    <div class="user-stat">
                        <div class="user-stat-value">{{.TotalMedias}}</div>
                        <div class="user-stat-label">Media Files</div>
                    </div>
                    <div class="user-stat">
                        <div class="user-stat-value">{{len .Entities}}</div>
                        <div class="user-stat-label">Entities</div>
                    </div>
                    <div class="user-stat">
                        <div class="user-stat-value">{{.User.FriendsCount}}</div>
                        <div class="user-stat-label">Following</div>
                    </div>
                </div>
                
                <div class="last-activity">
                    📅 Last Activity: {{formatTimeAgo .LatestActivity}}
                </div>
                
                <div class="entity-list">
                    {{range .Entities}}
                    <div class="entity-item">
                        📁 {{.Name}} ({{if .MediaCount.Valid}}{{.MediaCount.Int32}}{{else}}0{{end}} files)
                    </div>
                    {{end}}
                </div>
            </div>
            {{end}}
        </div>
    </div>

    <script>
        // Auto-refresh every 30 seconds
        setInterval(() => {
            location.reload();
        }, 30000);
    </script>
</body>
</html>
`
