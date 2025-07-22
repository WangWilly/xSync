package twitterclient

import (
	"context"
	"fmt"
	"net/url"

	"github.com/tidwall/gjson"
)

////////////////////////////////////////////////////////////////////////////////

const (
	LIST_META_VARIABLES_FORM = `{"listId":"%d"}`
	LIST_META_FEATURES       = `{"rweb_tipjar_consumption_enabled":true,"responsive_web_graphql_exclude_directive_enabled":true,"verified_phone_label_enabled":false,"responsive_web_graphql_skip_user_profile_image_extensions_enabled":false,"responsive_web_graphql_timeline_navigation_enabled":true}`
)

////////////////////////////////////////////////////////////////////////////////

func (c *Client) GetRawListByteById(ctx context.Context, listId uint64) (*gjson.Result, error) {
	return c.getRawListByteById(ctx, GRAPHQL_LIST_BY_REST_ID, listId)
}

func (c *Client) getRawListByteById(ctx context.Context, path string, listId uint64) (*gjson.Result, error) {
	u, _ := url.Parse(API_HOST)
	u = u.JoinPath(path)

	params := url.Values{}
	params.Set("variables", fmt.Sprintf(LIST_META_VARIABLES_FORM, listId))
	params.Set("features", LIST_META_FEATURES)

	u.RawQuery = params.Encode()
	requestUrl := u.String()

	resp, err := c.restyClient.R().SetContext(ctx).Get(requestUrl)
	if err != nil {
		return nil, err
	}

	res := gjson.GetBytes(resp.Body(), "data.list")
	if !res.Exists() {
		return nil, fmt.Errorf("the list with ID %d doesn't exist", listId)
	}
	return &res, nil
}
