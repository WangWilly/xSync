package twitterclient

import (
	"net/http"
	"time"

	"github.com/WangWilly/xSync/pkgs/commonpkg/utils"
	"github.com/go-resty/resty/v2"
)

////////////////////////////////////////////////////////////////////////////////

const (
	TWITTER_API_BEARER_TOKEN = "AAAAAAAAAAAAAAAAAAAAANRILgAAAAAAnNwIzUejRCOuH5E6I8xnZz4puTs%3D1Zv7ttfk8LF81IUq16cHjhLTvJu4FA33AGWWjCpTnA"
)

////////////////////////////////////////////////////////////////////////////////

func (c *Client) setTwitterIdenty(authToken string, ct0 string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.setClientAuth(authToken, ct0)
	c.configureErrorHandling()
	c.configureRetryLogic()
	c.configureTransport()
}

// setClientAuth configures authentication for the Twitter API client
func (c *Client) setClientAuth(authToken string, ct0 string) {
	c.restyClient.SetAuthToken(TWITTER_API_BEARER_TOKEN)
	c.restyClient.SetCookie(&http.Cookie{
		Name:  COOKIE_AUTH_TOKEN,
		Value: authToken,
	})
	c.restyClient.SetCookie(&http.Cookie{
		Name:  COOKIE_CT0,
		Value: ct0,
	})
	c.restyClient.SetHeader(HEADER_CSRF_TOKEN, ct0)
}

// configureErrorHandling sets up error handling for the client
func (c *Client) configureErrorHandling() {
	c.restyClient.OnAfterResponse(func(client *resty.Client, r *resty.Response) error {
		// Import the CheckApiResp function from the twitter package
		// This would need to be moved to a common package or imported
		return utils.CheckRespStatus(r)
	})
}

func (c *Client) configureRetryLogic() {
	c.restyClient.SetRetryCount(5)

	c.restyClient.AddRetryCondition(func(r *resty.Response, err error) bool {
		if err == ErrWouldBlock {
			return false
		}
		// For TCP Error - would need to import TwitterApiError from twitter package
		return err != nil && !isKnownError(err)
	})

	c.restyClient.AddRetryCondition(func(r *resty.Response, err error) bool {
		// For Http 429
		if httpErr, ok := err.(*utils.HttpStatusError); ok {
			return r.Request.RawRequest.Host == "x.com" && httpErr.Code == 429
		}
		return false
	})
}

// configureTransport sets up HTTP transport configuration
func (c *Client) configureTransport() {
	c.restyClient.SetTransport(&http.Transport{
		MaxIdleConns:          0,
		MaxIdleConnsPerHost:   100,
		IdleConnTimeout:       5 * time.Second,
		TLSHandshakeTimeout:   5 * time.Second,
		ResponseHeaderTimeout: 5 * time.Second,
		Proxy:                 http.ProxyFromEnvironment,
	})
}
