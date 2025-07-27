package database

import (
	"fmt"

	"github.com/WangWilly/xSync/pkgs/commonpkg/utils"
	"github.com/jmoiron/sqlx"

	log "github.com/sirupsen/logrus"
)

////////////////////////////////////////////////////////////////////////////////

const (
	DATABASE_TYPE_SQLITE   = "sqlite"
	DATABASE_TYPE_POSTGRES = "postgres"
)

type DatabaseConfig struct {
	Type string `yaml:"type"` // "sqlite" or "postgres"

	Host     string `yaml:"host"`     // For PostgreSQL
	Port     string `yaml:"port"`     // For PostgreSQL
	User     string `yaml:"user"`     // For PostgreSQL
	Password string `yaml:"password"` // For PostgreSQL
	DBName   string `yaml:"dbname"`   // For PostgreSQL

	Path string `yaml:"path"` // For SQLite
}

////////////////////////////////////////////////////////////////////////////////

func ConnectWithConfig(dbConfig DatabaseConfig) (*sqlx.DB, error) {
	logger := log.WithFields(log.Fields{
		"caller": "ConnectWithConfig",
		"type":   dbConfig.Type,
	})

	switch dbConfig.Type {
	case DATABASE_TYPE_POSTGRES:
		logger.WithFields(log.Fields{
			"host":   dbConfig.Host,
			"port":   dbConfig.Port,
			"dbname": dbConfig.DBName,
		}).Info("Connecting to PostgreSQL database")

		db, err := connectPostgres(
			dbConfig.Host,
			dbConfig.Port,
			dbConfig.User,
			dbConfig.Password,
			dbConfig.DBName,
		)
		if err != nil {
			return nil, err
		}
		return db, nil

	case DATABASE_TYPE_SQLITE:
		if dbConfig.Path == "" {
			return nil, fmt.Errorf("SQLite database path is required")
		}
		logger.
			WithField("path", dbConfig.Path).
			Info("Connecting to SQLite database")
		return connectSqlite(dbConfig.Path)

	default:
		return nil, fmt.Errorf("unsupported database type: %s", dbConfig.Type)
	}
}

////////////////////////////////////////////////////////////////////////////////

func connectSqlite(path string) (*sqlx.DB, error) {
	logger := log.WithFields(log.Fields{
		"caller": "ConnectDatabase",
		"path":   path,
	})

	ok, err := utils.PathExists(path)
	if err != nil {
		return nil, err
	}
	if !ok {
		logger.Debugln("created new db file")
	}

	db, err := sqlx.Connect(
		"sqlite3",
		fmt.Sprintf(
			"file:%s?_journal_mode=WAL&busy_timeout=2147483647",
			path,
		),
	)
	if err != nil {
		return nil, err
	}

	return db, nil
}

////////////////////////////////////////////////////////////////////////////////

func connectPostgres(host, port, user, password, dbname string) (*sqlx.DB, error) {
	logger := log.WithFields(log.Fields{
		"caller": "ConnectPostgres",
		"host":   host,
		"port":   port,
		"dbname": dbname,
	})

	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host,
		port,
		user,
		password,
		dbname,
	)

	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		logger.WithError(err).Error("Failed to connect to PostgreSQL")
		return nil, fmt.Errorf("failed to connect to postgres: %w", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		logger.WithError(err).Error("Failed to ping PostgreSQL")
		db.Close()
		return nil, fmt.Errorf("failed to ping postgres: %w", err)
	}

	return db, nil
}
