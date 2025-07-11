package twitterclient

import (
	"context"
	"fmt"
	"net/url"

	"github.com/tidwall/gjson"
)

// GetFollowing retrieves all users that the specified user is following
func (c *Client) GetFollowing(ctx context.Context, userId uint64, pageSize int, cursor string) ([]*User, string, error) {
	// Build the request URL
	requestUrl := c.buildFollowingUrl(userId, pageSize, cursor)

	// Make the API request
	resp, err := c.restyClient.R().SetContext(ctx).Get(requestUrl)
	if err != nil {
		return nil, "", err
	}

	// Parse the response
	return c.parseFollowingResponse(resp.Body())
}

// GetAllFollowing retrieves all users that the specified user is following (all pages)
func (c *Client) GetAllFollowing(ctx context.Context, userId uint64) ([]*User, error) {
	allUsers := make([]*User, 0)
	cursor := ""

	for {
		users, nextCursor, err := c.GetFollowing(ctx, userId, DEFAULT_MEMBERS_PAGE_SIZE, cursor)
		if err != nil {
			return nil, err
		}

		if len(users) == 0 {
			break
		}

		allUsers = append(allUsers, users...)
		cursor = nextCursor

		if cursor == "" {
			break
		}
	}

	return allUsers, nil
}

// parseFollowingResponse parses the following API response
func (c *Client) parseFollowingResponse(body []byte) ([]*User, string, error) {
	// Check for protected user response
	if string(body) == "{\"data\":{\"user\":{}}}" {
		return nil, "", nil
	}

	// Parse instructions
	instructions := gjson.GetBytes(body, INST_PATH_USER_TIMELINE)
	if !instructions.Exists() {
		return nil, "", fmt.Errorf("unable to get instructions from response")
	}

	// Extract entries and items
	entries := c.getEntries(instructions)
	moduleItems := c.getModuleItems(instructions)

	if !entries.Exists() && !moduleItems.Exists() {
		return nil, "", fmt.Errorf("invalid instructions: no entries or module items found")
	}

	// Process item contents
	itemContents := make([]gjson.Result, 0)
	if entries.IsArray() {
		for _, entry := range entries.Array() {
			if entry.Get("content.entryType").String() != "TimelineTimelineCursor" {
				itemContents = append(itemContents, c.getItemContentsFromEntry(entry)...)
			}
		}
	}
	if moduleItems.IsArray() {
		for _, moduleItem := range moduleItems.Array() {
			itemContents = append(itemContents, c.getItemContentFromModuleItem(moduleItem))
		}
	}

	// Convert to users and get next cursor
	users := c.itemContentsToUsers(itemContents)
	nextCursor := c.getNextCursor(entries)

	return users, nextCursor, nil
}

// buildFollowingUrl constructs the URL for fetching user following
func (c *Client) buildFollowingUrl(userId uint64, pageSize int, cursor string) string {
	baseUrl := API_HOST + GRAPHQL_FOLLOWING

	// Build query parameters
	params := url.Values{}

	// Variables parameter
	variables := fmt.Sprintf(`{"userId":"%d","count":%d,"includePromotedContent":false, "cursor":"%s"}`, userId, pageSize, cursor)
	params.Set("variables", variables)

	// Features parameter
	features := `{"rweb_tipjar_consumption_enabled":true,"responsive_web_graphql_exclude_directive_enabled":true,"verified_phone_label_enabled":false,"creator_subscriptions_tweet_preview_api_enabled":true,"responsive_web_graphql_timeline_navigation_enabled":true,"responsive_web_graphql_skip_user_profile_image_extensions_enabled":false,"communities_web_enable_tweet_community_results_fetch":true,"c9s_tweet_anatomy_moderator_badge_enabled":true,"articles_preview_enabled":true,"tweetypie_unmention_optimization_enabled":true,"responsive_web_edit_tweet_api_enabled":true,"graphql_is_translatable_rweb_tweet_is_translatable_enabled":true,"view_counts_everywhere_api_enabled":true,"longform_notetweets_consumption_enabled":true,"responsive_web_twitter_article_tweet_consumption_enabled":true,"tweet_awards_web_tipping_enabled":false,"creator_subscriptions_quote_tweet_preview_enabled":false,"freedom_of_speech_not_reach_fetch_enabled":true,"standardized_nudges_misinfo":true,"tweet_with_visibility_results_prefer_gql_limited_actions_policy_enabled":true,"rweb_video_timestamps_enabled":true,"longform_notetweets_rich_text_read_enabled":true,"longform_notetweets_inline_media_enabled":true,"responsive_web_enhance_cards_enabled":false}`
	params.Set("features", features)

	// Construct final URL
	u, _ := url.Parse(baseUrl)
	u.RawQuery = params.Encode()
	return u.String()
}
