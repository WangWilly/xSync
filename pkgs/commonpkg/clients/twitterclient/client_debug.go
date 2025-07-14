package twitterclient

import (
	"net/url"
	"strings"

	"github.com/go-resty/resty/v2"
)

func (c *Client) SetRequestCounting(cb func(path string)) {
	c.restyClient.OnBeforeRequest(func(client *resty.Client, req *resty.Request) error {
		url, err := url.Parse(req.URL)
		if err != nil {
			return err
		}

		if strings.HasSuffix(url.Host, X_IMG) {
			return nil
		}

		cb(url.Path)
		return nil
	})
}
