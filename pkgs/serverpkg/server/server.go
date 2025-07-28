package server

import (
	"fmt"
	"html/template"
	"net/http"

	"github.com/WangWilly/xSync/pkgs/commonpkg/repos/mediarepo"
	"github.com/WangWilly/xSync/pkgs/commonpkg/repos/tweetrepo"
	"github.com/WangWilly/xSync/pkgs/commonpkg/repos/userentityrepo"
	"github.com/WangWilly/xSync/pkgs/commonpkg/repos/userrepo"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

// Server represents the web server for displaying tweet data
type Server struct {
	db        *sqlx.DB
	templates *template.Template
	port      string

	userRepo       UserRepo
	userEntityRepo UserEntityRepo
	mediaRepo      MediaRepo
	tweetRepo      TweetRepo
}

// NewServerWithConfig creates a new server instance with database configuration
func NewServerWithConfig(db *sqlx.DB, port string) (*Server, error) {

	templates, err := template.New("").
		Funcs(createTemplateFunctions()).
		ParseGlob("./cmd/server/templates/*.html")
	if err != nil {
		return nil, fmt.Errorf("failed to parse templates: %w", err)
	}
	fmt.Println(templates.DefinedTemplates())

	return &Server{
		db:        db,
		templates: templates,
		port:      port,

		userRepo:       userrepo.New(),
		userEntityRepo: userentityrepo.New(),
		mediaRepo:      mediarepo.New(),
		tweetRepo:      tweetrepo.New(),
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
