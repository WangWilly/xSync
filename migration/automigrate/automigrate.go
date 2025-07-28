package automigrate

import (
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
	log "github.com/sirupsen/logrus"
)

////////////////////////////////////////////////////////////////////////////////

// AutoMigrateConfig holds configuration for auto-migration
type AutoMigrateConfig struct {
	SqlxDB *sqlx.DB
}

////////////////////////////////////////////////////////////////////////////////

// AutoMigrateUp automatically runs all pending migrations up
// This is intended for development use to ensure database schema is up to date
func AutoMigrateUp(config AutoMigrateConfig) error {
	logger := log.WithFields(log.Fields{
		"function": "AutoMigrateUp",
	})
	if config.SqlxDB == nil {
		return fmt.Errorf("no database connection provided")
	}
	logger.Info("Starting auto-migration...")

	db := config.SqlxDB.DB
	databaseType := driverNameToDatabaseType(driverNameFromDB(db))

	migrationsPath, err := getMigrationPath(databaseType)
	if err != nil {
		return fmt.Errorf("failed to get migration path: %w", err)
	}
	driver, err := getDriver(db)
	if err != nil {
		return fmt.Errorf("failed to create database driver: %w", err)
	}
	m, err := migrate.NewWithDatabaseInstance(migrationsPath, "", driver)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}
	// Note: Don't close the migrate instance when using WithInstance as it will close the underlying DB

	// Get current version
	currentVersion, dirty, err := m.Version()
	if err != nil && err != migrate.ErrNilVersion {
		return fmt.Errorf("failed to get current migration version: %w", err)
	}

	if dirty {
		logger.Warn("Database is in dirty state, attempting to continue...")
	}

	err = m.Up()
	if err != nil {
		if err == migrate.ErrNoChange {
			logger.WithField("version", currentVersion).Info("Database is already up to date")
			return nil
		}
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	// Get new version
	newVersion, _, err := m.Version()
	if err != nil {
		return fmt.Errorf("failed to get new migration version: %w", err)
	}

	logger.WithFields(log.Fields{
		"from_version": currentVersion,
		"to_version":   newVersion,
	}).Info("Auto-migration completed successfully")

	return nil
}

////////////////////////////////////////////////////////////////////////////////

// GetMigrationVersion returns the current migration version
func GetMigrationVersion(config AutoMigrateConfig) (uint, bool, error) {
	logger := log.WithFields(log.Fields{
		"function": "GetMigrationVersion",
	})
	if config.SqlxDB == nil {
		return 0, false, fmt.Errorf("no database connection provided")
	}
	logger.Info("Retrieving current migration version...")

	db := config.SqlxDB.DB
	databaseType := driverNameToDatabaseType(driverNameFromDB(db))

	migrationsPath, err := getMigrationPath(databaseType)
	if err != nil {
		return 0, false, fmt.Errorf("failed to get migration path: %w", err)
	}
	driver, err := getDriver(db)
	if err != nil {
		return 0, false, fmt.Errorf("failed to create database driver: %w", err)
	}
	m, err := migrate.NewWithDatabaseInstance(migrationsPath, "", driver)
	if err != nil {
		return 0, false, fmt.Errorf("failed to create migrate instance: %w", err)
	}
	// Note: Don't close the migrate instance when using WithInstance as it will close the underlying DB

	return m.Version()
}
