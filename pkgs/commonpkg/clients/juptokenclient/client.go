package juptokenclient

import (
	"time"

	"github.com/go-resty/resty/v2"
)

////////////////////////////////////////////////////////////////////////////////

const BASE_URL = "https://tokens.jup.ag"

////////////////////////////////////////////////////////////////////////////////

type Client struct {
	restyClient *resty.Client
}

func New() *Client {
	client := resty.New()
	client.SetBaseURL(BASE_URL)
	client.SetHeader("User-Agent", "xSync/1.0")
	client.SetTimeout(60 * time.Second)
	client.SetRetryCount(3)
	client.SetRetryWaitTime(1 * time.Second)

	return &Client{
		restyClient: client,
	}
}
