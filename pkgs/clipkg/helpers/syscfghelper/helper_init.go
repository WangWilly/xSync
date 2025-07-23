package syscfghelper

import (
	"errors"
	"os"
)

////////////////////////////////////////////////////////////////////////////////

const (
	SYS_STATE_DIR           = ".x_sync"
	SYS_LOG_FILE            = "x_sync.log"
	WORKER_CLIENT_LOG_FILE  = "worker_client.log"
	SYS_CONF_FILE           = "conf.yaml"
	ADDITIONAL_COOKIES_FILE = "additional_cookies.yaml"
)

////////////////////////////////////////////////////////////////////////////////

func fileExists(filename string) (bool, error) {
	info, err := os.Stat(filename)
	if err == nil {
		return !info.IsDir(), nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return false, err
}
