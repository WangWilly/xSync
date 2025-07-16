package juptokenclient

import (
	"time"

	"github.com/go-resty/resty/v2"
)

////////////////////////////////////////////////////////////////////////////////

const BASE_URL = "https://tokens.jup.ag"

////////////////////////////////////////////////////////////////////////////////

type client struct {
	restyClient *resty.Client
}

func New() *client {
	restyClient := resty.New()
	restyClient.SetBaseURL(BASE_URL)
	restyClient.SetHeader("User-Agent", "xSync/1.0")
	restyClient.SetTimeout(60 * time.Second)
	restyClient.SetRetryCount(3)
	restyClient.SetRetryWaitTime(1 * time.Second)

	return &client{
		restyClient: restyClient,
	}
}
