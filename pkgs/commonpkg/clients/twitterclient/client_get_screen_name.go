package twitterclient

import (
	"context"
	"regexp"

	"github.com/WangWilly/xSync/pkgs/commonpkg/utils"
)

////////////////////////////////////////////////////////////////////////////////

var SCREEN_NAME_PATTERN = regexp.MustCompile(`"screen_name":"(\S+?)"`)

////////////////////////////////////////////////////////////////////////////////

// GetScreenName returns the screen name associated with the client
func (c *Client) GetScreenName(ctx context.Context) (string, error) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	if c.screenName != "" {
		return c.screenName, nil
	}

	name, err := c.getScreenNameFromTwitter(ctx)
	if err != nil {
		return "", err
	}
	c.screenName = name
	return c.screenName, nil
}

func (c *Client) getScreenNameFromTwitter(ctx context.Context) (string, error) {
	// Clone client and remove Authorization header
	clonedClient := c.restyClient.Clone()
	clonedClient.SetAuthToken("")

	req := clonedClient.R().
		SetContext(ctx).
		SetHeaders(map[string]string{
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
	subs := SCREEN_NAME_PATTERN.FindStringSubmatch(string(home))
	if len(subs) == 0 {
		return ""
	}
	return subs[1]
}
