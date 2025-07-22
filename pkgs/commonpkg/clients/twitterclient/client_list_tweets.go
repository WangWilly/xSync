package twitterclient

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/WangWilly/xSync/pkgs/commonpkg/utils"
	"github.com/tidwall/gjson"
)

////////////////////////////////////////////////////////////////////////////////

type Tweet struct {
	Id        uint64    // Unique identifier for the tweet
	Text      string    // Tweet content text
	CreatedAt time.Time // When the tweet was created
	Creator   *User     // User who created the tweet
	Urls      []string  // Media URLs associated with the tweet
}

////////////////////////////////////////////////////////////////////////////////

func (c *Client) ListAllTweetsByUser(ctx context.Context, user *User) ([]*Tweet, error) {
	return c.ListTweetsByUserAndTimeRange(ctx, user, utils.TimeRange{})
}

func (c *Client) ListTweetsByUserAndTimeRange(ctx context.Context, user *User, timeRange utils.TimeRange) ([]*Tweet, error) {
	if !user.IsUserVisible() {
		return nil, nil
	}

	results := make([]*Tweet, 0)
	cursor := ""

	for {
		currentTweets, next, err := c.ListTweets(
			ctx,
			user.TwitterId,
			DEFAULT_PAGE_SIZE_FOR_TWEETS,
			cursor,
		)
		if err != nil {
			return nil, err
		}

		if len(currentTweets) == 0 {
			break
		}

		cursor = next

		// 筛选推文，并判断是否获取下页
		cutMin, cutMax, currentTweets := filterTweetsByTimeRange(currentTweets, timeRange.Begin, timeRange.End)
		results = append(results, currentTweets...)

		if cutMin {
			break
		}
		if cutMax && len(currentTweets) != 0 {
			timeRange.End = time.Time{}
		}
	}
	return results, nil
}

func (c *Client) ListTweets(ctx context.Context, userId uint64, pageSize int, cursor string) ([]*Tweet, string, error) {
	// Build the request URL
	requestUrl := c.buildUserMediaUrl(userId, pageSize, cursor)

	// Make the API request
	resp, err := c.restyClient.R().SetContext(ctx).Get(requestUrl)
	if err != nil {
		return nil, "", err
	}

	// Parse the response directly
	return c.parseUserMediaResponse(resp.Body())
}

func (c *Client) buildUserMediaUrl(userId uint64, pageSize int, cursor string) string {
	baseUrl := API_HOST + GRAPHQL_USER_MEDIA

	// Build query parameters
	params := url.Values{}

	// Variables parameter
	variables := fmt.Sprintf(`{"userId":"%d","count":%d,"cursor":"%s","includePromotedContent":false,"withClientEventToken":false,"withBirdwatchNotes":false,"withVoice":true,"withV2Timeline":true}`, userId, pageSize, cursor)
	params.Set("variables", variables)

	// Features parameter
	features := `{"rweb_tipjar_consumption_enabled":true,"responsive_web_graphql_exclude_directive_enabled":true,"verified_phone_label_enabled":false,"creator_subscriptions_tweet_preview_api_enabled":true,"responsive_web_graphql_timeline_navigation_enabled":true,"responsive_web_graphql_skip_user_profile_image_extensions_enabled":false,"communities_web_enable_tweet_community_results_fetch":true,"c9s_tweet_anatomy_moderator_badge_enabled":true,"articles_preview_enabled":true,"tweetypie_unmention_optimization_enabled":true,"responsive_web_edit_tweet_api_enabled":true,"graphql_is_translatable_rweb_tweet_is_translatable_enabled":true,"view_counts_everywhere_api_enabled":true,"longform_notetweets_consumption_enabled":true,"responsive_web_twitter_article_tweet_consumption_enabled":true,"tweet_awards_web_tipping_enabled":false,"creator_subscriptions_quote_tweet_preview_enabled":false,"freedom_of_speech_not_reach_fetch_enabled":true,"standardized_nudges_misinfo":true,"tweet_with_visibility_results_prefer_gql_limited_actions_policy_enabled":true,"rweb_video_timestamps_enabled":true,"longform_notetweets_rich_text_read_enabled":true,"longform_notetweets_inline_media_enabled":true,"responsive_web_enhance_cards_enabled":false}`
	params.Set("features", features)

	// Field toggles parameter
	fieldToggles := `{"withArticlePlainText":false}`
	params.Set("fieldToggles", fieldToggles)

	// Construct final URL
	u, _ := url.Parse(baseUrl)
	u.RawQuery = params.Encode()
	return u.String()
}

// parseUserMediaResponse parses the user media API response
func (c *Client) parseUserMediaResponse(body []byte) ([]*Tweet, string, error) {
	// Check for protected user response
	if string(body) == "{\"data\":{\"user\":{}}}" {
		return nil, "", nil
	}

	// Parse instructions
	instructions := gjson.GetBytes(body, INST_PATH_USER_MEDIA)
	if !instructions.Exists() {
		return nil, "", fmt.Errorf("unable to get instructions from response")
	}

	// Extract entries and items
	entries := getEntries(instructions)
	moduleItems := getModuleItems(instructions)

	if !entries.Exists() && !moduleItems.Exists() {
		return nil, "", fmt.Errorf("invalid instructions: no entries or module items found")
	}

	// Process item contents
	itemContents := make([]gjson.Result, 0)
	if entries.IsArray() {
		for _, entry := range entries.Array() {
			if entry.Get("content.entryType").String() != "TimelineTimelineCursor" {
				itemContents = append(itemContents, getItemContentsFromEntry(entry)...)
			}
		}
	}
	if moduleItems.IsArray() {
		for _, moduleItem := range moduleItems.Array() {
			itemContents = append(itemContents, getItemContentFromModuleItem(moduleItem))
		}
	}

	// Convert to tweets and get next cursor
	tweets := itemContentsToTweets(itemContents)
	nextCursor := getNextCursor(entries)

	return tweets, nextCursor, nil
}

func filterTweetsByTimeRange(tweetsFromLatestToEarliest []*Tweet, timeBegin time.Time, timeEnd time.Time) (cutMin bool, cutMax bool, res []*Tweet) {
	n := len(tweetsFromLatestToEarliest)
	begin, end := 0, n

	// 从左到右查找第一个小于 min 的推文
	if !timeBegin.IsZero() {
		for i := range n {
			if tweetsFromLatestToEarliest[i].CreatedAt.After(timeBegin) {
				continue
			}
			end = i // 找到第一个不大于 min 的推文位置
			cutMin = true
			break
		}
	}

	// 从右到左查找最后一个大于 max 的推文
	if !timeEnd.IsZero() {
		for i := n - 1; i >= 0; i-- {
			if tweetsFromLatestToEarliest[i].CreatedAt.Before(timeEnd) {
				continue
			}
			begin = i + 1 // 找到第一个不小于 max 的推文位置
			cutMax = true
			break
		}
	}

	if begin >= end {
		// 如果最终的范围无效，返回空结果
		return cutMin, cutMax, nil
	}

	res = tweetsFromLatestToEarliest[begin:end]
	return
}
