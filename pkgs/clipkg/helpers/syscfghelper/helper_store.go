package syscfghelper

import (
	"os"
	"path/filepath"

	"github.com/WangWilly/xSync/pkgs/commonpkg/database"
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
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil && !os.IsExist(err) {
		return "", err
	}
	return p, nil
}

func (h *helper) GetErrorBkJsonPath() (string, error) {
	p := filepath.Join(h.sysConfig.RootPath, ERROR_BK_JSON_FILE)
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil && !os.IsExist(err) {
		return "", err
	}
	return p, nil
}

func (h *helper) GetUsersAssetsPath() (string, error) {
	p := filepath.Join(h.sysConfig.RootPath, USERS_ASSETS_DIR)
	if err := os.MkdirAll(p, 0755); err != nil && !os.IsExist(err) {
		return "", err
	}
	return p, nil
}

func (h *helper) GetDatabaseConfig() database.DatabaseConfig {
	dbConfig := h.sysConfig.Database
	if dbConfig.Type == "" {
		panic("Database type is not set in the configuration")
	}
	return dbConfig
}
