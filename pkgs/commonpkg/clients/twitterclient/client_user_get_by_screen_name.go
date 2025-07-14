package twitterclient

import (
	"context"
	"fmt"
	"net/url"
)

// GetUserByScreenName retrieves a user by their screen name (username)
func (c *Client) GetUserByScreenName(ctx context.Context, screenName string) (*User, error) {
	// Build the request URL
	requestUrl := c.buildUserByScreenNameUrl(screenName)

	// Make the API request
	resp, err := c.restyClient.R().SetContext(ctx).Get(requestUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to get user [%s]: %v", screenName, err)
	}

	// Parse the response
	return c.parseUserResp(resp.Body())
}

// buildUserByScreenNameUrl constructs the URL for fetching user by screen name
func (c *Client) buildUserByScreenNameUrl(screenName string) string {
	baseUrl := API_HOST + GRAPHQL_USER_BY_SCREEN_NAME

	// Build query parameters
	params := url.Values{}

	// Variables parameter
	variables := fmt.Sprintf(`{"screen_name":"%s","withSafetyModeUserFields":true}`, screenName)
	params.Set("variables", variables)

	// Features parameter
	features := `{"hidden_profile_subscriptions_enabled":true,"rweb_tipjar_consumption_enabled":true,"responsive_web_graphql_exclude_directive_enabled":true,"verified_phone_label_enabled":false,"subscriptions_verification_info_is_identity_verified_enabled":true,"subscriptions_verification_info_verified_since_enabled":true,"highlights_tweets_tab_ui_enabled":true,"responsive_web_twitter_article_notes_tab_enabled":true,"subscriptions_feature_can_gift_premium":false,"creator_subscriptions_tweet_preview_api_enabled":true,"responsive_web_graphql_skip_user_profile_image_extensions_enabled":false,"responsive_web_graphql_timeline_navigation_enabled":true}`
	params.Set("features", features)

	// Field toggles parameter
	fieldToggles := `{"withAuxiliaryUserLabels":false}`
	params.Set("fieldToggles", fieldToggles)

	// Construct final URL
	u, _ := url.Parse(baseUrl)
	u.RawQuery = params.Encode()
	return u.String()
}
