package commandline

import (
	"context"

	"github.com/WangWilly/xSync/pkgs/clients/twitterclient"
	"github.com/WangWilly/xSync/pkgs/config"
	log "github.com/sirupsen/logrus"
)

func BatchLogin(ctx context.Context, cookies []*config.Cookie) []*twitterclient.Client {
	if len(cookies) == 0 {
		return nil
	}

	res := make([]*twitterclient.Client, 0, len(cookies))
	for i, cookie := range cookies {
		if cookie.AuthToken == "" || cookie.Ct0 == "" {
			log.Warnf("skipping invalid cookie at index %d: %v", i, cookie)
			continue
		}

		client := twitterclient.New()
		client.SetTwitterIdenty(ctx, cookie.AuthToken, cookie.Ct0)
		client.SetRateLimit()

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
