package twitter

import (
	"context"
	"fmt"
	"time"

	"github.com/WangWilly/xSync/pkgs/utils"
	"github.com/go-resty/resty/v2"
	"github.com/tidwall/gjson"
)

////////////////////////////////////////////////////////////////////////////////
// Types and Constants
////////////////////////////////////////////////////////////////////////////////

// FollowState represents the follow relationship state between users
type FollowState int

const (
	FS_UNFOLLOW  FollowState = iota // Not following the user
	FS_FOLLOWING                    // Currently following the user
	FS_REQUESTED                    // Follow request sent but not yet approved
)

// User represents a Twitter user with their profile information and relationship status
type User struct {
	TwitterId    uint64      // User's unique identifier
	Name         string      // Display name
	ScreenName   string      // Username (handle)
	IsProtected  bool        // Whether the account is protected/private
	FriendsCount int         // Number of accounts this user follows
	Followstate  FollowState // Current follow relationship status
	MediaCount   int         // Number of media posts by this user
	Muting       bool        // Whether this user is muted
	Blocking     bool        // Whether this user is blocked
}

// NewUserById retrieves a user by their unique ID
func NewUserById(ctx context.Context, client *resty.Client, id uint64) (*User, error) {
	api := userByRestId{id}
	getUrl := makeUrl(&api)
	r, err := newUser(ctx, client, getUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to get user [%d]: %v", id, err)
	}
	return r, err
}

// NewUserByScreenName retrieves a user by their screen name (username)
func NewUserByScreenName(ctx context.Context, client *resty.Client, screenName string) (*User, error) {
	u := makeUrl(&userByScreenName{screenName: screenName})
	r, err := newUser(ctx, client, u)
	if err != nil {
		return nil, fmt.Errorf("failed to get user [%s]: %v", screenName, err)
	}
	return r, err
}

// newUser is a low-level function that makes the HTTP request to get user data
func newUser(ctx context.Context, client *resty.Client, url string) (*User, error) {
	resp, err := client.R().SetContext(ctx).Get(url)
	if err != nil {
		return nil, err
	}
	return parseUserResp(resp.Body())
}

func (u *User) GetAllMeidas(ctx context.Context, client *resty.Client) ([]*Tweet, error) {
	// Get all media tweets for the user without a time range
	return u.GetMeidas(ctx, client, utils.TimeRange{})
}

// GetMeidas retrieves all media tweets for the user within an optional time range
func (u *User) GetMeidas(ctx context.Context, client *resty.Client, timeRange utils.TimeRange) ([]*Tweet, error) {
	if !u.IsVisiable() {
		return nil, nil
	}

	api := DefaultUserMediaQuery(u.TwitterId)
	results := make([]*Tweet, 0)
	for {
		currentTweets, next, err := u.getMediasOnePage(ctx, api, client)
		if err != nil {
			return nil, err
		}

		if len(currentTweets) == 0 {
			break
		}

		api.SetCursor(next)

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

// Title returns a formatted string with the user's display name and screen name
func (u *User) Title() string {
	return fmt.Sprintf("%s(%s)", u.Name, u.ScreenName)
}

// Following returns a UserFollowing interface for this user
func (u *User) Following() UserFollowing {
	return UserFollowing{u}
}

// IsVisiable checks if the user's content is visible (either following or public account)
func (u *User) IsVisiable() bool {
	return u.Followstate == FS_FOLLOWING || !u.IsProtected
}

// getMediasOnePage retrieves one page of media tweets for the user
func (u *User) getMediasOnePage(ctx context.Context, api *userMediaQuery, client *resty.Client) ([]*Tweet, string, error) {
	if !u.IsVisiable() {
		return nil, "", nil
	}

	itemContents, next, err := getTimelineItemContents(ctx, api, client, "data.user.result.timeline_v2.timeline.instructions")
	return itemContentsToTweets(itemContents), next, err
}

////////////////////////////////////////////////////////////////////////////////
// JSON Parsing and Data Processing
////////////////////////////////////////////////////////////////////////////////

// parseUserResp parses the top-level JSON response to extract user data
func parseUserResp(resp []byte) (*User, error) {
	user := gjson.GetBytes(resp, "data.user")
	if !user.Exists() {
		return nil, fmt.Errorf("user does not exist")
	}
	return parseUserJson(&user)
}

// parseUserJson parses the user data from Twitter API JSON response
func parseUserJson(userJson *gjson.Result) (*User, error) {
	result := userJson.Get("result")
	if result.Get("__typename").String() == "UserUnavailable" {
		return nil, fmt.Errorf("user unavaiable")
	}
	legacy := result.Get("legacy")

	restId := result.Get("rest_id")
	friends_count := legacy.Get("friends_count")
	name := legacy.Get("name")
	screen_name := legacy.Get("screen_name")
	protected := legacy.Get("protected").Exists() && legacy.Get("protected").Bool()
	media_count := legacy.Get("media_count")
	muting := legacy.Get("muting")
	blocking := legacy.Get("blocking")

	usr := User{}
	if foll := legacy.Get("following"); foll.Exists() {
		if foll.Bool() {
			usr.Followstate = FS_FOLLOWING
		} else {
			usr.Followstate = FS_UNFOLLOW
		}
	} else if legacy.Get("follow_request_sent").Exists() {
		usr.Followstate = FS_REQUESTED
	} else {
		usr.Followstate = FS_UNFOLLOW
	}

	usr.FriendsCount = int(friends_count.Int())
	usr.TwitterId = restId.Uint()
	usr.IsProtected = protected
	usr.Name = name.String()
	usr.ScreenName = screen_name.String()
	usr.MediaCount = int(media_count.Int())
	usr.Muting = muting.Exists() && muting.Bool()
	usr.Blocking = blocking.Exists() && blocking.Bool()
	return &usr, nil
}

////////////////////////////////////////////////////////////////////////////////
// User Media Operations
////////////////////////////////////////////////////////////////////////////////

// itemContentsToTweets converts timeline item contents to Tweet objects
func itemContentsToTweets(itemContents []gjson.Result) []*Tweet {
	res := make([]*Tweet, 0, len(itemContents))
	for _, itemContent := range itemContents {
		tweetResults := getResults(itemContent, timelineTweet)
		if tw := parseTweetResults(&tweetResults); tw != nil {
			res = append(res, tw)
		}
	}
	return res
}

// filterTweetsByTimeRange filters tweets by time range from a reverse-ordered slice
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

// FollowUser sends a follow request to the specified user
func FollowUser(ctx context.Context, client *resty.Client, user *User) error {
	url := "https://x.com/i/api/1.1/friendships/create.json"
	_, err := client.R().SetFormData(map[string]string{
		"user_id": fmt.Sprintf("%d", user.TwitterId),
		// "skip_status":                       1,
		// "include_profile_interstitial_type": 1,
		// "include_blocking":                  1,
		// "include_blocked_by":                1,
		// "include_followed_by":               1,
		// "include_want_retweets":             1,
		// "include_mute_edge":                 1,
		// "include_can_dm":                    1,
		// "include_can_media_tag":             1,
		// "include_ext_is_blue_verified":      1,
		// "include_ext_verified_type":         1,
		// "include_ext_profile_image_shape":   1,
	}).SetContext(ctx).Post(url)
	return err
}
