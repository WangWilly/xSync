package database

import (
	"fmt"

	"github.com/WangWilly/xSync/pkgs/clipkg/config"
	"github.com/WangWilly/xSync/pkgs/commonpkg/model"
	"github.com/WangWilly/xSync/pkgs/commonpkg/utils"
	"github.com/jmoiron/sqlx"

	log "github.com/sirupsen/logrus"
)

// ConnectWithConfig connects to database using configuration
func ConnectWithConfig(dbConfig config.DatabaseConfig) (*sqlx.DB, error) {
	logger := log.WithFields(log.Fields{
		"caller": "ConnectWithConfig",
		"type":   dbConfig.Type,
	})

	switch dbConfig.Type {
	case "postgres", "postgresql":
		logger.WithFields(log.Fields{
			"host":   dbConfig.Host,
			"port":   dbConfig.Port,
			"dbname": dbConfig.DBName,
		}).Info("Connecting to PostgreSQL database")

		db, err := ConnectPostgres(dbConfig.Host, dbConfig.Port, dbConfig.User, dbConfig.Password, dbConfig.DBName)
		if err != nil {
			return nil, err
		}

		// Create tables for PostgreSQL
		err = model.CreateTablesPostgres(db)
		if err != nil {
			logger.WithError(err).Error("Failed to create PostgreSQL tables")
			return nil, err
		}

		return db, nil

	case "sqlite", "sqlite3", "":
		// Default to SQLite if not specified or empty
		if dbConfig.Path == "" {
			return nil, fmt.Errorf("SQLite database path is required")
		}

		logger.WithField("path", dbConfig.Path).Info("Connecting to SQLite database")
		return ConnectDatabase(dbConfig.Path)

	default:
		return nil, fmt.Errorf("unsupported database type: %s", dbConfig.Type)
	}
}

func ConnectDatabase(path string) (*sqlx.DB, error) {
	logger := log.WithFields(log.Fields{
		"caller": "ConnectDatabase",
		"path":   path,
	})

	ex, err := utils.PathExists(path)
	if err != nil {
		return nil, err
	}

	dsn := fmt.Sprintf("file:%s?_journal_mode=WAL&busy_timeout=2147483647", path)
	db, err := sqlx.Connect("sqlite3", dsn)
	if err != nil {
		return nil, err
	}
	model.CreateTables(db)
	//db.SetMaxOpenConns(1)

	if !ex {
		logger.Debugln("created new db file")
	}
	return db, nil
}
