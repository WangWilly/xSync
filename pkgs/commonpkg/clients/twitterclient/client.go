package twitterclient

import (
	"sync"

	"github.com/go-resty/resty/v2"
	log "github.com/sirupsen/logrus"
)

////////////////////////////////////////////////////////////////////////////////

// urls
const (
	X_HOME = "https://x.com/home"
	X_IMG  = "twimg.com"
)

// header keys
const (
	HEADER_USER_AGENT = "User-Agent"
	HEADER_CSRF_TOKEN = "X-Csrf-Token"
)

// cookie names
const (
	COOKIE_AUTH_TOKEN = "auth_token"
	COOKIE_CT0        = "ct0"
)

// agent strings
const (
	USER_AGENT_1 = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"
)

////////////////////////////////////////////////////////////////////////////////

type Client struct {
	restyClient *resty.Client
	screenName  string
	rateLimiter *rateLimiter
	error       error
	mutex       sync.RWMutex
}

func New() *Client {
	return &Client{
		restyClient: resty.New(),
	}
}

////////////////////////////////////////////////////////////////////////////////

func (c *Client) SetLogger(logger *log.Logger) {
	c.restyClient.SetLogger(logger)
}

////////////////////////////////////////////////////////////////////////////////
// Client State Management

// GetError returns any error associated with the client
func (c *Client) GetError() error {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.error
}

// SetError sets an error for the client, marking it as unavailable
func (c *Client) SetError(err error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.error = err
	if err != nil {
		log.WithField("client", c.screenName).Debugln("client is no longer available:", err)
	}
}

// IsAvailable checks if the client is available for use
func (c *Client) IsAvailable() bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.error == nil
}
