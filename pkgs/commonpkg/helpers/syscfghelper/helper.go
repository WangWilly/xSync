package syscfghelper

import (
	"log"
	"os"
	"path/filepath"

	"github.com/WangWilly/xSync/pkgs/clipkg/config"
	"github.com/WangWilly/xSync/pkgs/commonpkg/logging"
	"github.com/WangWilly/xSync/pkgs/downloading"
)

type CliParams struct {
	IsDebug       bool
	ConfOverWrite bool
}

type helper struct {
	cliParams CliParams

	logFile                 *os.File
	clientLogFiles          []*os.File
	workerClientLogFilePath string

	sysConfig             *config.Config
	additionalCookiesPath string
}

func New(cliParams CliParams) *helper {
	h := &helper{
		cliParams: cliParams,
	}

	h.init()

	return h
}

func (h *helper) init() {
	sysStateDir := filepath.Join(getHomePath(), SYS_STATE_DIR)
	if err := os.MkdirAll(sysStateDir, 0755); err != nil {
		log.Fatalln("failed to make app dir", err)
	}

	////////////////////////////////////////////////////////////////////////////

	logPath := filepath.Join(sysStateDir, SYS_LOG_FILE)
	logFile, err := os.OpenFile(logPath, os.O_TRUNC|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		log.Fatalln("failed to create log file:", err)
	}
	logging.InitLogger(h.cliParams.IsDebug, logFile)
	h.logFile = logFile

	h.workerClientLogFilePath = filepath.Join(sysStateDir, WORKER_CLIENT_LOG_FILE)

	////////////////////////////////////////////////////////////////////////////

	confPath := filepath.Join(sysStateDir, SYS_CONF_FILE)
	if ok, err := fileExists(confPath); err != nil {
		log.Fatalln("failed to check config file existence:", err)
	} else if !ok || h.cliParams.ConfOverWrite {
		conf, err := config.PromptConfig(confPath)
		if err != nil {
			log.Fatalln("failed to prompt config:", err)
		}
		h.sysConfig = conf
	} else {
		conf, err := config.ParseConfigFromFile(confPath)
		if err != nil {
			log.Fatalln("failed to load config:", err)
		}
		h.sysConfig = conf
	}

	h.additionalCookiesPath = filepath.Join(sysStateDir, ADDITIONAL_COOKIES_FILE)

	////////////////////////////////////////////////////////////////////////////
}

////////////////////////////////////////////////////////////////////////////////

func (h *helper) GetDownloadingCfg() downloading.Config {
	return downloading.Config{
		MaxDownloadRoutine: h.sysConfig.MaxDownloadRoutine,
	}
}

////////////////////////////////////////////////////////////////////////////////

func (h *helper) Close() {
	if h.logFile != nil {
		h.logFile.Close()
	}
	for _, file := range h.clientLogFiles {
		if file != nil {
			file.Close()
		}
	}
}
