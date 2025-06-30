package logger

import (
	"io"

	"github.com/rifflock/lfshook"
	log "github.com/sirupsen/logrus"
)

////////////////////////////////////////////////////////////////////////////////
// Logging Configuration Functions
////////////////////////////////////////////////////////////////////////////////

// InitLogger initializes the application logger with the specified configuration
func InitLogger(dbg bool, logFile io.Writer) {
	log.SetFormatter(&log.TextFormatter{
		ForceColors:   true,
		FullTimestamp: true,
	})

	if dbg {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}

	log.AddHook(lfshook.NewHook(logFile, nil))
}
