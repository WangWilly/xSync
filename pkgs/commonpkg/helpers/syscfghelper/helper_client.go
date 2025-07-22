package syscfghelper

import (
	"context"
	"os"

	"github.com/WangWilly/xSync/pkgs/clipkg/config"
	"github.com/WangWilly/xSync/pkgs/commonpkg/clients/twitterclient"
	"github.com/gookit/color"

	log "github.com/sirupsen/logrus"
)

func (h *helper) GetMainClient(ctx context.Context) (*twitterclient.Client, error) {
	logger := log.WithField("caller", "syscfghelper.GetMainClient")

	client := twitterclient.New(
		h.sysConfig.Cookie.AuthToken,
		h.sysConfig.Cookie.Ct0,
	)
	screenName, err := client.GetScreenName(ctx)
	if err != nil {
		logger.Errorln("failed to get screen name:", err)
		return nil, err
	}

	////////////////////////////////////////////////////////////////////////////

	clientLogFile, err := os.OpenFile(
		h.workerClientLogFilePath,
		os.O_TRUNC|os.O_WRONLY|os.O_CREATE,
		0644,
	)
	if err != nil {
		logger.Errorln("failed to create log file:", err)
		return nil, err
	}
	twitterclient.SetTwitterClientLogger(client, clientLogFile)
	h.clientLogFiles = append(h.clientLogFiles, clientLogFile)

	////////////////////////////////////////////////////////////////////////////

	logger.Infoln("signed in as:", color.FgLightBlue.Render(screenName))
	return client, nil
}

func (h *helper) GetOtherClients(ctx context.Context) ([]*twitterclient.Client, error) {
	logger := log.WithField("caller", "syscfghelper.GetOtherClients")

	cookies, err := config.ReadAdditionalCookies(h.additionalCookiesPath)
	if err != nil {
		logger.Warnln("failed to load additional cookies:", err)
		return nil, err
	}

	clients := batchLogin(ctx, cookies)
	clientLogFile, err := os.OpenFile(
		h.workerClientLogFilePath,
		os.O_TRUNC|os.O_WRONLY|os.O_CREATE,
		0644,
	)
	if err != nil {
		logger.Errorln("failed to create log file:", err)
		return nil, err
	}
	for _, client := range clients {
		twitterclient.SetTwitterClientLogger(client, clientLogFile)
	}

	return clients, nil
}

func batchLogin(ctx context.Context, cookies []*config.Cookie) []*twitterclient.Client {
	if len(cookies) == 0 {
		return nil
	}

	res := make([]*twitterclient.Client, 0, len(cookies))
	for i, cookie := range cookies {
		if cookie.AuthToken == "" || cookie.Ct0 == "" {
			log.Warnf("skipping invalid cookie at index %d: %v", i, cookie)
			continue
		}

		client := twitterclient.New(cookie.AuthToken, cookie.Ct0)
		screenName, err := client.GetScreenName(ctx)
		if err != nil {
			log.Warnf("failed to get screen name for client %d: %v", i, err)
			continue
		}

		log.Infof("logged in as %s with additional cookies at index %d", screenName, i)
		res = append(res, client)
	}

	log.Infoln("loaded additional accounts:", len(res))
	return res
}
