package twitterclient

import (
	"net/url"
	"strings"

	"github.com/go-resty/resty/v2"
	"github.com/tidwall/gjson"
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

////////////////////////////////////////////////////////////////////////////////

const (
	ErrTimeout         = 29
	ErrDependency      = 0
	ErrExceedPostLimit = 88
	ErrOverCapacity    = 130
	ErrAccountLocked   = 326
)

////////////////////////////////////////////////////////////////////////////////

func CheckApiResp(body []byte) error {
	errors := gjson.GetBytes(body, "errors")
	if !errors.Exists() {
		return nil
	}

	codej := errors.Get("0.code")
	code := -1
	if codej.Exists() {
		code = int(codej.Int())
	}
	return NewTwitterApiError(code, string(body))
}

type TwitterApiError struct {
	Code int
	raw  string
}

func (err *TwitterApiError) Error() string {
	return err.raw
}

func NewTwitterApiError(code int, raw string) *TwitterApiError {
	return &TwitterApiError{Code: code, raw: raw}
}
