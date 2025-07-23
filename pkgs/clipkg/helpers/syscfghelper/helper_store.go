package syscfghelper

import (
	"os"
	"path/filepath"

	"github.com/WangWilly/xSync/pkgs/clipkg/config"
)

////////////////////////////////////////////////////////////////////////////////

const (
	SQLITE_DB_FILE     = "/data/xSync.db"
	ERROR_BK_JSON_FILE = "/data/errors.json"
	USERS_ASSETS_DIR   = "/users"
)

////////////////////////////////////////////////////////////////////////////////

// Deprecated: Use GetDatabaseConfig() instead
func (h *helper) GetSqliteDBPath() (string, error) {
	p := filepath.Join(h.sysConfig.RootPath, SQLITE_DB_FILE)
	err := os.MkdirAll(filepath.Dir(p), 0755)
	if err != nil && !os.IsExist(err) {
		return "", err
	}

	return p, nil
}

func (h *helper) GetErrorBkJsonPath() (string, error) {
	p := filepath.Join(h.sysConfig.RootPath, ERROR_BK_JSON_FILE)
	err := os.MkdirAll(filepath.Dir(p), 0755)
	if err != nil && !os.IsExist(err) {
		return "", err
	}

	return p, nil
}

func (h *helper) GetUsersAssetsPath() (string, error) {
	p := filepath.Join(h.sysConfig.RootPath, USERS_ASSETS_DIR)
	err := os.MkdirAll(p, 0755)
	if err != nil && !os.IsExist(err) {
		return "", err
	}

	return p, nil
}

func (h *helper) GetDatabaseConfig() config.DatabaseConfig {
	dbConfig := h.sysConfig.Database

	// If database configuration is not specified, default to SQLite
	if dbConfig.Type == "" {
		dbConfig.Type = "sqlite"
		if dbConfig.Path == "" {
			dbConfig.Path = filepath.Join(h.sysConfig.RootPath, SQLITE_DB_FILE)
		}
	}

	return dbConfig
}
