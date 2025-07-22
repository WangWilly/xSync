package server

import (
	"fmt"
	"html/template"
	"net/http"
	"path/filepath"

	"github.com/WangWilly/xSync/pkgs/clipkg/config"
	"github.com/WangWilly/xSync/pkgs/commonpkg/database"
	"github.com/WangWilly/xSync/pkgs/commonpkg/repos/mediarepo"
	"github.com/WangWilly/xSync/pkgs/commonpkg/repos/tweetrepo"
	"github.com/WangWilly/xSync/pkgs/commonpkg/repos/userrepo"
	"github.com/WangWilly/xSync/pkgs/downloading"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

// Server represents the web server for displaying tweet data
type Server struct {
	db        *sqlx.DB
	dumper    *downloading.TweetDumper
	templates *template.Template
	port      string

	userRepo  UserRepo
	mediaRepo MediaRepo
	tweetRepo TweetRepo
}

// NewServer creates a new server instance
func NewServer(dbPath, port string) (*Server, error) {
	// For backward compatibility, create a SQLite database config
	dbConfig := config.DatabaseConfig{
		Type: "sqlite",
		Path: dbPath,
	}
	return NewServerWithConfig(dbConfig, port)
}

// NewServerWithConfig creates a new server instance with database configuration
func NewServerWithConfig(dbConfig config.DatabaseConfig, port string) (*Server, error) {
	// Connect to database
	db, err := database.ConnectWithConfig(dbConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Load tweet dumper
	dumper := downloading.NewDumper(db)

	// For SQLite, try to load error file from same directory
	if dbConfig.Type == "sqlite" || dbConfig.Type == "sqlite3" {
		dumpPath := filepath.Join(filepath.Dir(dbConfig.Path), "error.json")
		if err := dumper.Load(dumpPath); err != nil {
			// Log warning but continue - this is not a fatal error
			fmt.Printf("Warning: Failed to load tweet dump file: %v\n", err)
		}
	}

	// Parse templates
	templates, err := template.New("").Funcs(createTemplateFunctions()).ParseGlob("./cmd/server/templates/*.html")
	if err != nil {
		return nil, fmt.Errorf("failed to parse templates: %w", err)
	}

	fmt.Println(templates.DefinedTemplates())

	return &Server{
		db:        db,
		dumper:    dumper,
		templates: templates,
		port:      port,

		userRepo:  userrepo.New(),
		mediaRepo: mediarepo.New(),
		tweetRepo: tweetrepo.New(),
	}, nil
}

// Close closes the server resources
func (s *Server) Close() {
	if s.db != nil {
		s.db.Close()
	}
}

// Start starts the HTTP server
func (s *Server) Start() error {
	s.setupRoutes()
	return http.ListenAndServe(":"+s.port, nil)
}

// setupRoutes configures all HTTP routes
func (s *Server) setupRoutes() {
	// Dashboard routes
	http.HandleFunc("/", s.handleDashboard)
	http.HandleFunc("/api/stats", s.handleAPIStats)

	// User routes
	http.HandleFunc("/user/", s.handleUser)

	// Tweet routes
	http.HandleFunc("/tweets/", s.handleTweets)
	http.HandleFunc("/api/tweets/", s.handleAPITweets)
	http.HandleFunc("/tweets-media/", s.handleTweetsWithMedia)

	// Media routes
	http.HandleFunc("/media/", s.handleMedia)
	http.HandleFunc("/api/media/", s.handleAPIMedia)

	// Static file routes
	http.HandleFunc("/static/", s.handleStatic)
	http.HandleFunc("/files/", s.handleFiles)
}
