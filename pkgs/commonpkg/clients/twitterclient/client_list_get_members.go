package twitterclient

import (
	"context"

	"github.com/WangWilly/xSync/pkgs/commonpkg/utils"
	log "github.com/sirupsen/logrus"
)

////////////////////////////////////////////////////////////////////////////////

const (
	LIST_MEMBERS_VARIABLES_FORM = `{"listId":"%d","count":%d,"withSafetyModeUserFields":true, "cursor":"%s"}`
	LIST_FEATURES               = `{"rweb_tipjar_consumption_enabled":true,"responsive_web_graphql_exclude_directive_enabled":true,"verified_phone_label_enabled":false,"creator_subscriptions_tweet_preview_api_enabled":true,"responsive_web_graphql_timeline_navigation_enabled":true,"responsive_web_graphql_skip_user_profile_image_extensions_enabled":false,"communities_web_enable_tweet_community_results_fetch":true,"c9s_tweet_anatomy_moderator_badge_enabled":true,"articles_preview_enabled":true,"tweetypie_unmention_optimization_enabled":true,"responsive_web_edit_tweet_api_enabled":true,"graphql_is_translatable_rweb_tweet_is_translatable_enabled":true,"view_counts_everywhere_api_enabled":true,"longform_notetweets_consumption_enabled":true,"responsive_web_twitter_article_tweet_consumption_enabled":true,"tweet_awards_web_tipping_enabled":false,"creator_subscriptions_quote_tweet_preview_enabled":false,"freedom_of_speech_not_reach_fetch_enabled":true,"standardized_nudges_misinfo":true,"tweet_with_visibility_results_prefer_gql_limited_actions_policy_enabled":true,"rweb_video_timestamps_enabled":true,"longform_notetweets_rich_text_read_enabled":true,"longform_notetweets_inline_media_enabled":true,"responsive_web_enhance_cards_enabled":false}`
)

////////////////////////////////////////////////////////////////////////////////

func (c *Client) GetAllListMembers(ctx context.Context, listId uint64) ([]*User, error) {
	logger := log.WithField("caller", "Client.GetAllListMembers")

	res, err := c.MustGetAllListMembers(ctx, listId)
	if err != nil {
		// 403: Dmcaed
		if utils.IsStatusCode(err, 404) || utils.IsStatusCode(err, 403) {
			logger.WithError(err).Errorf("failed to get members for list %d", listId)
			return nil, nil
		}
		logger.WithError(err).Errorf("failed to get members for list %d", listId)
		return nil, err
	}
	return res, nil
}

func (c *Client) MustGetAllListMembers(ctx context.Context, listId uint64) ([]*User, error) {
	api := ListParams{
		VariablesForm: LIST_MEMBERS_VARIABLES_FORM,
		Features:      LIST_FEATURES,

		Id:     listId,
		Count:  200,
		Cursor: "",
	}

	itemContents, err := c.getTimelineItemContentsTillEnd(ctx, GRAPHQL_LIST_MEMBERS, api, INST_PATH_LIST_MEMBERS)
	if err != nil {
		return nil, err
	}

	return c.itemContentsToUsers(itemContents), nil
}

func (c *Client) GetListMembers(ctx context.Context, listId uint64, pageSize int, cursor string) ([]*User, string, error) {
	listParams := ListParams{
		VariablesForm: LIST_MEMBERS_VARIABLES_FORM,
		Features:      LIST_FEATURES,

		Id:     listId,
		Count:  pageSize,
		Cursor: cursor,
	}

	itemContents, nextCursor, err := c.getTimelineItemContents(ctx, GRAPHQL_LIST_MEMBERS, listParams, INST_PATH_LIST_MEMBERS)
	if err != nil {
		return nil, "", err
	}

	users := c.itemContentsToUsers(itemContents)
	return users, nextCursor, nil
}
