package twitterclient

import (
	"context"
	"fmt"
	"net/url"

	"github.com/tidwall/gjson"
)

// GetList retrieves a Twitter list by its ID
func (c *Client) GetList(ctx context.Context, listId uint64) (*List, error) {
	// Build the request URL
	requestUrl := c.buildListByIdUrl(listId)

	// Make the API request
	resp, err := c.restyClient.R().SetContext(ctx).Get(requestUrl)
	if err != nil {
		return nil, err
	}

	// Parse the response
	list := gjson.GetBytes(resp.Body(), "data.list")
	return c.parseList(&list)
}

// buildListByIdUrl constructs the URL for fetching list by ID
func (c *Client) buildListByIdUrl(listId uint64) string {
	baseUrl := API_HOST + GRAPHQL_LIST_BY_REST_ID

	// Build query parameters
	params := url.Values{}

	// Variables parameter
	variables := fmt.Sprintf(`{"listId":"%d"}`, listId)
	params.Set("variables", variables)

	// Features parameter
	features := `{"rweb_tipjar_consumption_enabled":true,"responsive_web_graphql_exclude_directive_enabled":true,"verified_phone_label_enabled":false,"responsive_web_graphql_skip_user_profile_image_extensions_enabled":false,"responsive_web_graphql_timeline_navigation_enabled":true}`
	params.Set("features", features)

	// Construct final URL
	u, _ := url.Parse(baseUrl)
	u.RawQuery = params.Encode()
	return u.String()
}
