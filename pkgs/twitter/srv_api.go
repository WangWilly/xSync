package twitter

import (
	"fmt"
	"net/url"
)

////////////////////////////////////////////////////////////////////////////////
// Constants and Global Configuration
////////////////////////////////////////////////////////////////////////////////

const HOST = "https://x.com"
const AvgTweetsPerPage = 70

////////////////////////////////////////////////////////////////////////////////
// Core API Interfaces
////////////////////////////////////////////////////////////////////////////////

// api defines the basic interface for all Twitter API endpoints
type api interface {
	Path() string
	QueryParam() url.Values
}

// timelineApi extends the basic api interface with cursor functionality for paginated endpoints
type timelineApi interface {
	SetCursor(cursor string)
	api
}

////////////////////////////////////////////////////////////////////////////////
// URL Construction Utilities
////////////////////////////////////////////////////////////////////////////////

// makeUrl constructs a complete URL from an API endpoint
func makeUrl(api api) string {
	u, _ := url.Parse(HOST) // 这里绝对不会出错
	u = u.JoinPath(api.Path())
	u.RawQuery = api.QueryParam().Encode()
	return u.String()
}

////////////////////////////////////////////////////////////////////////////////
// User API Structures
////////////////////////////////////////////////////////////////////////////////

// userByRestId represents the API endpoint for fetching user information by REST ID
type userByRestId struct {
	restId uint64
}

func (*userByRestId) Path() string {
	return "/i/api/graphql/CO4_gU4G_MRREoqfiTh6Hg/UserByRestId"
}

func (a *userByRestId) QueryParam() url.Values {
	v := url.Values{}
	variables := `{"userId":"%d","withSafetyModeUserFields":true}`
	v.Set("variables", fmt.Sprintf(variables, a.restId))
	features := `{"hidden_profile_likes_enabled":true,"hidden_profile_subscriptions_enabled":true,"rweb_tipjar_consumption_enabled":true,"responsive_web_graphql_exclude_directive_enabled":true,"verified_phone_label_enabled":false,"highlights_tweets_tab_ui_enabled":true,"responsive_web_twitter_article_notes_tab_enabled":true,"subscriptions_feature_can_gift_premium":false,"creator_subscriptions_tweet_preview_api_enabled":true,"responsive_web_graphql_skip_user_profile_image_extensions_enabled":false,"responsive_web_graphql_timeline_navigation_enabled":true}`
	v.Set("features", features)
	return v
}

// userByScreenName represents the API endpoint for fetching user information by screen name
type userByScreenName struct {
	screenName string
}

func (*userByScreenName) Path() string {
	return "/i/api/graphql/xmU6X_CKVnQ5lSrCbAmJsg/UserByScreenName"
}

func (a *userByScreenName) QueryParam() url.Values {
	v := url.Values{}

	variables := `{"screen_name":"%s","withSafetyModeUserFields":true}`
	features := `{"hidden_profile_subscriptions_enabled":true,"rweb_tipjar_consumption_enabled":true,"responsive_web_graphql_exclude_directive_enabled":true,"verified_phone_label_enabled":false,"subscriptions_verification_info_is_identity_verified_enabled":true,"subscriptions_verification_info_verified_since_enabled":true,"highlights_tweets_tab_ui_enabled":true,"responsive_web_twitter_article_notes_tab_enabled":true,"subscriptions_feature_can_gift_premium":false,"creator_subscriptions_tweet_preview_api_enabled":true,"responsive_web_graphql_skip_user_profile_image_extensions_enabled":false,"responsive_web_graphql_timeline_navigation_enabled":true}`
	fieldToggles := `{"withAuxiliaryUserLabels":false}`

	v.Set("variables", fmt.Sprintf(variables, a.screenName))
	v.Set("features", features)
	v.Set("fieldToggles", fieldToggles)
	return v
}

// userMediaQuery represents the API endpoint for fetching user media
type userMediaQuery struct {
	userId   uint64
	pageSize int
	cursor   string
}

func (*userMediaQuery) Path() string {
	return "/i/api/graphql/MOLbHrtk8Ovu7DUNOLcXiA/UserMedia"
}

func (a *userMediaQuery) QueryParam() url.Values {
	v := url.Values{}

	variables := `{"userId":"%d","count":%d,"cursor":"%s","includePromotedContent":false,"withClientEventToken":false,"withBirdwatchNotes":false,"withVoice":true,"withV2Timeline":true}`
	features := `{"rweb_tipjar_consumption_enabled":true,"responsive_web_graphql_exclude_directive_enabled":true,"verified_phone_label_enabled":false,"creator_subscriptions_tweet_preview_api_enabled":true,"responsive_web_graphql_timeline_navigation_enabled":true,"responsive_web_graphql_skip_user_profile_image_extensions_enabled":false,"communities_web_enable_tweet_community_results_fetch":true,"c9s_tweet_anatomy_moderator_badge_enabled":true,"articles_preview_enabled":true,"tweetypie_unmention_optimization_enabled":true,"responsive_web_edit_tweet_api_enabled":true,"graphql_is_translatable_rweb_tweet_is_translatable_enabled":true,"view_counts_everywhere_api_enabled":true,"longform_notetweets_consumption_enabled":true,"responsive_web_twitter_article_tweet_consumption_enabled":true,"tweet_awards_web_tipping_enabled":false,"creator_subscriptions_quote_tweet_preview_enabled":false,"freedom_of_speech_not_reach_fetch_enabled":true,"standardized_nudges_misinfo":true,"tweet_with_visibility_results_prefer_gql_limited_actions_policy_enabled":true,"rweb_video_timestamps_enabled":true,"longform_notetweets_rich_text_read_enabled":true,"longform_notetweets_inline_media_enabled":true,"responsive_web_enhance_cards_enabled":false}`
	fieldToggles := `{"withArticlePlainText":false}`

	v.Set("variables", fmt.Sprintf(variables, a.userId, a.pageSize, a.cursor))
	v.Set("features", features)
	v.Set("fieldToggles", fieldToggles)
	return v
}

func (a *userMediaQuery) SetCursor(cursor string) {
	a.cursor = cursor
}

func DefaultUserMediaQuery(twitterId uint64) *userMediaQuery {
	return &userMediaQuery{
		userId:   twitterId,
		pageSize: 100,
		cursor:   "",
	}
}

////////////////////////////////////////////////////////////////////////////////
// List API Structures
////////////////////////////////////////////////////////////////////////////////

// listByRestId represents the API endpoint for fetching list information by REST ID
type listByRestId struct {
	id uint64
}

func (*listByRestId) Path() string {
	return "/i/api/graphql/ZMQOSpxDo0cP5Cdt8MgEVA/ListByRestId"
}

func (a *listByRestId) QueryParam() url.Values {
	v := url.Values{}

	variables := `{"listId":"%d"}`
	features := `{"rweb_tipjar_consumption_enabled":true,"responsive_web_graphql_exclude_directive_enabled":true,"verified_phone_label_enabled":false,"responsive_web_graphql_skip_user_profile_image_extensions_enabled":false,"responsive_web_graphql_timeline_navigation_enabled":true}`

	v.Set("variables", fmt.Sprintf(variables, a.id))
	v.Set("features", features)
	return v
}

// listMembers represents the API endpoint for fetching list members
type listMembers struct {
	id     uint64
	count  int
	cursor string
}

func (*listMembers) Path() string {
	return "/i/api/graphql/3dQPyRyAj6Lslp4e0ClXzg/ListMembers"
}

func (a *listMembers) QueryParam() url.Values {
	v := url.Values{}
	variables := `{"listId":"%d","count":%d,"withSafetyModeUserFields":true, "cursor":"%s"}`
	features := `{"rweb_tipjar_consumption_enabled":true,"responsive_web_graphql_exclude_directive_enabled":true,"verified_phone_label_enabled":false,"creator_subscriptions_tweet_preview_api_enabled":true,"responsive_web_graphql_timeline_navigation_enabled":true,"responsive_web_graphql_skip_user_profile_image_extensions_enabled":false,"communities_web_enable_tweet_community_results_fetch":true,"c9s_tweet_anatomy_moderator_badge_enabled":true,"articles_preview_enabled":true,"tweetypie_unmention_optimization_enabled":true,"responsive_web_edit_tweet_api_enabled":true,"graphql_is_translatable_rweb_tweet_is_translatable_enabled":true,"view_counts_everywhere_api_enabled":true,"longform_notetweets_consumption_enabled":true,"responsive_web_twitter_article_tweet_consumption_enabled":true,"tweet_awards_web_tipping_enabled":false,"creator_subscriptions_quote_tweet_preview_enabled":false,"freedom_of_speech_not_reach_fetch_enabled":true,"standardized_nudges_misinfo":true,"tweet_with_visibility_results_prefer_gql_limited_actions_policy_enabled":true,"rweb_video_timestamps_enabled":true,"longform_notetweets_rich_text_read_enabled":true,"longform_notetweets_inline_media_enabled":true,"responsive_web_enhance_cards_enabled":false}`

	v.Set("variables", fmt.Sprintf(variables, a.id, a.count, a.cursor))
	v.Set("features", features)
	return v
}

func (a *listMembers) SetCursor(cursor string) {
	a.cursor = cursor
}

////////////////////////////////////////////////////////////////////////////////
// Following API Structures
////////////////////////////////////////////////////////////////////////////////

// following represents the API endpoint for fetching user's following list
type following struct {
	uid    uint64
	count  int
	cursor string
}

func (*following) Path() string {
	return "/i/api/graphql/7FEKOPNAvxWASt6v9gfCXw/Following"
}

func (a *following) QueryParam() url.Values {
	v := url.Values{}
	variables := `{"userId":"%d","count":%d,"includePromotedContent":false, "cursor":"%s"}`
	features := `{"rweb_tipjar_consumption_enabled":true,"responsive_web_graphql_exclude_directive_enabled":true,"verified_phone_label_enabled":false,"creator_subscriptions_tweet_preview_api_enabled":true,"responsive_web_graphql_timeline_navigation_enabled":true,"responsive_web_graphql_skip_user_profile_image_extensions_enabled":false,"communities_web_enable_tweet_community_results_fetch":true,"c9s_tweet_anatomy_moderator_badge_enabled":true,"articles_preview_enabled":true,"tweetypie_unmention_optimization_enabled":true,"responsive_web_edit_tweet_api_enabled":true,"graphql_is_translatable_rweb_tweet_is_translatable_enabled":true,"view_counts_everywhere_api_enabled":true,"longform_notetweets_consumption_enabled":true,"responsive_web_twitter_article_tweet_consumption_enabled":true,"tweet_awards_web_tipping_enabled":false,"creator_subscriptions_quote_tweet_preview_enabled":false,"freedom_of_speech_not_reach_fetch_enabled":true,"standardized_nudges_misinfo":true,"tweet_with_visibility_results_prefer_gql_limited_actions_policy_enabled":true,"rweb_video_timestamps_enabled":true,"longform_notetweets_rich_text_read_enabled":true,"longform_notetweets_inline_media_enabled":true,"responsive_web_enhance_cards_enabled":false}`

	v.Set("variables", fmt.Sprintf(variables, a.uid, a.count, a.cursor))
	v.Set("features", features)
	return v
}

func (a *following) SetCursor(cursor string) {
	a.cursor = cursor
}

////////////////////////////////////////////////////////////////////////////////
// Likes API Structures
////////////////////////////////////////////////////////////////////////////////

// likes represents the API endpoint for fetching user's liked tweets
type likes struct {
	userId uint64
	count  int
	cursor string
}

func (l *likes) Path() string {
	return "/i/api/graphql/aeJWz--kknVBOl7wQ7gh7Q/Likes"
}

func (l *likes) QueryParam() url.Values {
	v := url.Values{}
	variables := `{"userId":"%d","count":%d,"includePromotedContent":false,"withClientEventToken":false,"withBirdwatchNotes":false,"withVoice":true,"withV2Timeline":true, "cursor":"%s"}`
	features := `{"rweb_tipjar_consumption_enabled":true,"responsive_web_graphql_exclude_directive_enabled":true,"verified_phone_label_enabled":false,"creator_subscriptions_tweet_preview_api_enabled":true,"responsive_web_graphql_timeline_navigation_enabled":true,"responsive_web_graphql_skip_user_profile_image_extensions_enabled":false,"communities_web_enable_tweet_community_results_fetch":true,"c9s_tweet_anatomy_moderator_badge_enabled":true,"articles_preview_enabled":true,"responsive_web_edit_tweet_api_enabled":true,"graphql_is_translatable_rweb_tweet_is_translatable_enabled":true,"view_counts_everywhere_api_enabled":true,"longform_notetweets_consumption_enabled":true,"responsive_web_twitter_article_tweet_consumption_enabled":true,"tweet_awards_web_tipping_enabled":false,"creator_subscriptions_quote_tweet_preview_enabled":false,"freedom_of_speech_not_reach_fetch_enabled":true,"standardized_nudges_misinfo":true,"tweet_with_visibility_results_prefer_gql_limited_actions_policy_enabled":true,"rweb_video_timestamps_enabled":true,"longform_notetweets_rich_text_read_enabled":true,"longform_notetweets_inline_media_enabled":true,"responsive_web_enhance_cards_enabled":false}`
	fieldToggles := `{"withArticlePlainText":false}`

	v.Set("variables", fmt.Sprintf(variables, l.userId, l.count, l.cursor))
	v.Set("features", features)
	v.Set("fieldToggles", fieldToggles)
	return v
}

func (l *likes) SetCursor(cursor string) {
	l.cursor = cursor
}
