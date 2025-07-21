package syscfghelper

import (
	"os"
	"path/filepath"
)

////////////////////////////////////////////////////////////////////////////////

const (
	SQLITE_DB_FILE     = "/data/xSync.db"
	ERROR_BK_JSON_FILE = "/data/errors.json"
	USERS_ASSETS_DIR   = "/users"
)

////////////////////////////////////////////////////////////////////////////////

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
