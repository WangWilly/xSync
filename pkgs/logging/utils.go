package logging

import (
	"io"

	"github.com/WangWilly/xSync/pkgs/clients/twitterclient"
	"github.com/rifflock/lfshook"
	log "github.com/sirupsen/logrus"
)

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

////////////////////////////////////////////////////////////////////////////////

func SetTwitterClientLogger(client *twitterclient.Client, out io.Writer) {
	logger := log.New()
	logger.SetLevel(log.InfoLevel)
	logger.SetOutput(out)
	logger.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
		DisableQuote:  true,
	})
	client.SetLogger(logger)
}
