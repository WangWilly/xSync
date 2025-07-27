package automigrate

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"reflect"
	"runtime"

	"github.com/golang-migrate/migrate/v4/database"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

////////////////////////////////////////////////////////////////////////////////

type DatabaseType string

const (
	PostgreSQL DatabaseType = "postgres"
	SQLite     DatabaseType = "sqlite"
)

////////////////////////////////////////////////////////////////////////////////

var sqlDriverNamesByType = map[reflect.Type]string{}

func driverNameFromDB(db *sql.DB) string {
	if len(sqlDriverNamesByType) == 0 {
		for _, driverName := range sql.Drivers() {
			tmpDB, _ := sql.Open(driverName, "")
			if tmpDB == nil {
				continue
			}
			driverType := reflect.TypeOf(tmpDB.Driver())
			sqlDriverNamesByType[driverType] = driverName
			tmpDB.Close()
		}
	}

	driverType := reflect.TypeOf(db.Driver())
	if name, ok := sqlDriverNamesByType[driverType]; ok {
		return name
	}
	return ""
}

func driverNameToDatabaseType(driverName string) DatabaseType {
	switch driverName {
	case "postgres":
		return PostgreSQL
	case "sqlite3":
		return SQLite
	default:
		return ""
	}
}

////////////////////////////////////////////////////////////////////////////////

func getMigrationDir() (string, error) {
	// Get the migration files path relative to this package
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("failed to get current file path")
	}

	// Go up one directory from automigrate to migration root
	return filepath.Dir(filepath.Dir(currentFile)), nil
}

func getMigrationPath(databaseType DatabaseType) (string, error) {
	migrationDir, err := getMigrationDir()
	if err != nil {
		return "", err
	}

	switch databaseType {
	case PostgreSQL:
		return fmt.Sprintf("file://%s/postgres", migrationDir), nil
	case SQLite:
		return fmt.Sprintf("file://%s/sqlite", migrationDir), nil
	default:
		return "", fmt.Errorf("unsupported database type: %s", databaseType)
	}
}

func getDriver(db *sql.DB) (database.Driver, error) {
	databaseType := driverNameToDatabaseType(driverNameFromDB(db))

	switch databaseType {
	case PostgreSQL:
		return postgres.WithInstance(db, &postgres.Config{})
	case SQLite:
		return sqlite3.WithInstance(db, &sqlite3.Config{})
	default:
		return nil, fmt.Errorf("unsupported database type: %s", databaseType)
	}
}
