package twitterclient

import (
	"context"
	"fmt"
	"net/url"
)

// GetUserById retrieves a user by their unique ID
func (c *Client) GetUserById(ctx context.Context, id uint64) (*User, error) {
	// Build the request URL
	requestUrl := c.buildUserByIdUrl(id)

	// Make the API request
	resp, err := c.restyClient.R().SetContext(ctx).Get(requestUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to get user [%d]: %v", id, err)
	}

	// Parse the response
	return c.parseUserResp(resp.Body())
}

// buildUserByIdUrl constructs the URL for fetching user by ID
func (c *Client) buildUserByIdUrl(userId uint64) string {
	baseUrl := API_HOST + GRAPHQL_USER_BY_REST_ID

	// Build query parameters
	params := url.Values{}

	// Variables parameter
	variables := fmt.Sprintf(`{"userId":"%d","withSafetyModeUserFields":true}`, userId)
	params.Set("variables", variables)

	// Features parameter
	features := `{"hidden_profile_likes_enabled":true,"hidden_profile_subscriptions_enabled":true,"rweb_tipjar_consumption_enabled":true,"responsive_web_graphql_exclude_directive_enabled":true,"verified_phone_label_enabled":false,"highlights_tweets_tab_ui_enabled":true,"responsive_web_twitter_article_notes_tab_enabled":true,"subscriptions_feature_can_gift_premium":false,"creator_subscriptions_tweet_preview_api_enabled":true,"responsive_web_graphql_skip_user_profile_image_extensions_enabled":false,"responsive_web_graphql_timeline_navigation_enabled":true}`
	params.Set("features", features)

	// Construct final URL
	u, _ := url.Parse(baseUrl)
	u.RawQuery = params.Encode()
	return u.String()
}
