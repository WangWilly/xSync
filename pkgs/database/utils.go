package database

import (
	"fmt"

	"github.com/WangWilly/xSync/pkgs/model"
	"github.com/WangWilly/xSync/pkgs/utils"
	"github.com/jmoiron/sqlx"

	log "github.com/sirupsen/logrus"
)

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
