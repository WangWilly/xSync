package twitterclient

import (
	"net/url"

	"github.com/go-resty/resty/v2"
	log "github.com/sirupsen/logrus"
)

func (c *Client) setRateLimit() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.rateLimiter != nil {
		log.WithField("client", c.screenName).Debugln("rate limiter already initialized")
		return
	}
	c.rateLimiter = newRateLimiter(true)

	////////////////////////////////////////////////////////////////////////////

	c.restyClient.OnBeforeRequest(func(client *resty.Client, req *resty.Request) error {
		u, err := url.Parse(req.URL)
		if err != nil {
			return err
		}
		return c.rateLimiter.check(req.Context(), u)
	})

	c.restyClient.OnSuccess(func(client *resty.Client, resp *resty.Response) {
		c.rateLimiter.reset(resp.Request.RawRequest.URL, resp)
	})

	c.restyClient.OnError(func(req *resty.Request, err error) {
		if req == nil || req.RawRequest == nil {
			return
		}

		var resp *resty.Response
		if v, ok := err.(*resty.ResponseError); ok {
			resp = v.Response
		}
		c.rateLimiter.reset(req.RawRequest.URL, resp)
	})

	c.restyClient.AddRetryHook(func(resp *resty.Response, err error) {
		if resp == nil || resp.Request == nil || resp.Request.RawRequest == nil {
			return
		}
		c.rateLimiter.reset(resp.Request.RawRequest.URL, resp)
	})
}

func (c *Client) WouldBlock(path string) bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.rateLimiter.wouldBlock(path)
}
