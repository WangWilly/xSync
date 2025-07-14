package twitterclient

import (
	"context"
	"regexp"

	"github.com/WangWilly/xSync/pkgs/commonpkg/utils"
)

////////////////////////////////////////////////////////////////////////////////

var screenNamePattern = regexp.MustCompile(`"screen_name":"(\S+?)"`)

////////////////////////////////////////////////////////////////////////////////

// GetScreenName returns the screen name associated with the client
func (c *Client) GetScreenName(ctx context.Context) (string, error) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	if c.screenName == "" {
		name, err := c.GetScreenNameFromTwitter(ctx)
		if err != nil {
			return "", err
		}
		c.screenName = name
	}
	return c.screenName, nil
}

func (c *Client) GetScreenNameFromTwitter(ctx context.Context) (string, error) {
	// Clone client and remove Authorization header
	clonedClient := c.restyClient.Clone()
	clonedClient.SetAuthToken("")

	req := clonedClient.R().SetContext(ctx).SetHeaders(map[string]string{
		HEADER_USER_AGENT: USER_AGENT_1,
	})
	resp, err := req.Get(X_HOME)
	if err != nil {
		return "", err
	}
	if err := utils.CheckRespStatus(resp); err != nil {
		return "", err
	}

	return extractScreenNameFromHome(resp.Body()), nil
}

func extractScreenNameFromHome(home []byte) string {
	// extracts screen name from home page HTML
	subs := screenNamePattern.FindStringSubmatch(string(home))
	if len(subs) == 0 {
		return ""
	}
	return subs[1]
}
