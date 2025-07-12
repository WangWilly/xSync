package server

import (
	"fmt"
	"html/template"
	"net/http"
	"path/filepath"

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

// NewServer creates a new server instance
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
		// Log warning but continue - this is not a fatal error
		fmt.Printf("Warning: Failed to load tweet dump file: %v\n", err)
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

// GetDB returns the database connection
func (s *Server) GetDB() *sqlx.DB {
	return s.db
}

// GetDumper returns the tweet dumper
func (s *Server) GetDumper() *downloading.TweetDumper {
	return s.dumper
}

// GetTemplates returns the templates
func (s *Server) GetTemplates() *template.Template {
	return s.templates
}
