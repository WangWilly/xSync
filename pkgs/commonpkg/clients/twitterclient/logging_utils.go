package twitterclient

import (
	"io"

	log "github.com/sirupsen/logrus"
)

func SetTwitterClientLogger(client *Client, out io.Writer) {
	logger := log.New()
	logger.SetLevel(log.InfoLevel)
	logger.SetOutput(out)
	logger.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
		DisableQuote:  true,
	})
	client.SetLogger(logger)
}
