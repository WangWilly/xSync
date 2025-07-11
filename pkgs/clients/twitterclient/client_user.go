package twitterclient

import (
	"context"
	"fmt"
	"time"

	"github.com/WangWilly/xSync/pkgs/utils"
	log "github.com/sirupsen/logrus"
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

// Tweet represents a Twitter tweet with its metadata and content
type Tweet struct {
	Id        uint64    // Unique identifier for the tweet
	Text      string    // Tweet content text
	CreatedAt time.Time // When the tweet was created
	Creator   *User     // User who created the tweet
	Urls      []string  // Media URLs associated with the tweet
}

// UserFollowing represents a user's following list as a ListBase implementation
type UserFollowing struct {
	creator *User // The user whose following list this represents
}

////////////////////////////////////////////////////////////////////////////////
// User Operations
////////////////////////////////////////////////////////////////////////////////

// GetUserById retrieves a user by their unique ID
// getUser is a low-level function that makes the HTTP request to get user data
func (c *Client) getUser(ctx context.Context, url string) (*User, error) {
	resp, err := c.restyClient.R().SetContext(ctx).Get(url)
	if err != nil {
		return nil, err
	}
	return c.parseUserResp(resp.Body())
}

// GetAllMedias retrieves all media tweets for the user without a time range
func (c *Client) GetAllMedias(ctx context.Context, user *User) ([]*Tweet, error) {
	return c.GetMedias(ctx, user, utils.TimeRange{})
}

// GetMedias retrieves all media tweets for the user within an optional time range
func (c *Client) GetMedias(ctx context.Context, user *User, timeRange utils.TimeRange) ([]*Tweet, error) {
	if !c.isUserVisible(user) {
		return nil, nil
	}

	results := make([]*Tweet, 0)
	cursor := ""

	for {
		currentTweets, next, err := c.GetMediasOnePage(ctx, user.TwitterId, DEFAULT_PAGE_SIZE, cursor)
		if err != nil {
			return nil, err
		}

		if len(currentTweets) == 0 {
			break
		}

		cursor = next

		// 筛选推文，并判断是否获取下页
		cutMin, cutMax, currentTweets := c.filterTweetsByTimeRange(currentTweets, timeRange.Begin, timeRange.End)
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

// FollowUser sends a follow request to the specified user
////////////////////////////////////////////////////////////////////////////////
// User Helper Methods
////////////////////////////////////////////////////////////////////////////////

// Title returns a formatted string with the user's display name and screen name
func (u *User) Title() string {
	return fmt.Sprintf("%s(%s)", u.Name, u.ScreenName)
}

// Following returns a UserFollowing interface for this user
func (u *User) Following() UserFollowing {
	return UserFollowing{u}
}

// IsVisible checks if the user's content is visible (either following or public account)
func (c *Client) isUserVisible(user *User) bool {
	return user.Followstate == FS_FOLLOWING || !user.IsProtected
}

////////////////////////////////////////////////////////////////////////////////
// User Following Operations
////////////////////////////////////////////////////////////////////////////////

// GetMembers retrieves all users that the creator is following
// GetId returns a negative ID to distinguish from regular lists
func (fo UserFollowing) GetId() int64 {
	return -int64(fo.creator.TwitterId)
}

// Title returns a formatted title for the following list
func (fo UserFollowing) Title() string {
	name := fmt.Sprintf("%s's Following", fo.creator.ScreenName)
	return name
}

// GetMembers retrieves all users that the creator is following
func (c *Client) GetUserFollowingMembers(ctx context.Context, fo UserFollowing) ([]*User, error) {
	return c.GetAllFollowing(ctx, fo.creator.TwitterId)
}

////////////////////////////////////////////////////////////////////////////////
// JSON Parsing and Data Processing
////////////////////////////////////////////////////////////////////////////////

// parseUserResp parses the top-level JSON response to extract user data
func (c *Client) parseUserResp(resp []byte) (*User, error) {
	user := gjson.GetBytes(resp, "data.user")
	if !user.Exists() {
		return nil, fmt.Errorf("user does not exist")
	}
	return c.parseUserJson(&user)
}

// parseUserJson parses the user data from Twitter API JSON response
func (c *Client) parseUserJson(userJson *gjson.Result) (*User, error) {
	result := userJson.Get("result")
	if result.Get("__typename").String() == "UserUnavailable" {
		return nil, fmt.Errorf("user unavailable")
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
// Tweet Operations
////////////////////////////////////////////////////////////////////////////////

// itemContentsToTweets converts timeline item contents to Tweet objects
func (c *Client) itemContentsToTweets(itemContents []gjson.Result) []*Tweet {
	res := make([]*Tweet, 0, len(itemContents))
	for _, itemContent := range itemContents {
		tweetResults := c.getResults(itemContent, timelineTweet)
		if tw := c.parseTweetResults(&tweetResults); tw != nil {
			res = append(res, tw)
		}
	}
	return res
}

// parseTweetResults parses tweet data from Twitter API JSON response
func (c *Client) parseTweetResults(tweet_results *gjson.Result) *Tweet {
	var tweet Tweet
	var err error = nil

	result := tweet_results.Get("result")
	if !result.Exists() || result.Get("__typename").String() == "TweetTombstone" {
		return nil
	}
	if result.Get("__typename").String() == "TweetWithVisibilityResults" {
		result = result.Get("tweet")
	}
	legacy := result.Get("legacy")
	// TODO: 利用 rest_id 重新获取推文信息
	if !legacy.Exists() {
		return nil
	}
	user_results := result.Get("core.user_results")

	tweet.Id = result.Get("rest_id").Uint()
	tweet.Text = legacy.Get("full_text").String()
	tweet.Creator, _ = c.parseUserJson(&user_results)
	tweet.CreatedAt, err = time.Parse(time.RubyDate, legacy.Get("created_at").String())
	if err != nil {
		panic(fmt.Errorf("invalid time format %v", err))
	}
	media := legacy.Get("extended_entities.media")
	if media.Exists() {
		tweet.Urls = c.getUrlsFromMedia(&media)
	}
	return &tweet
}

// getUrlsFromMedia extracts media URLs from tweet media entities
func (c *Client) getUrlsFromMedia(media *gjson.Result) []string {
	results := []string{}
	for _, m := range media.Array() {
		typ := m.Get("type").String()
		switch typ {
		case "video", "animated_gif":
			results = append(results, m.Get("video_info.variants.@reverse.0.url").String())
		case "photo":
			results = append(results, m.Get("media_url_https").String())
		}
	}
	return results
}

// filterTweetsByTimeRange filters tweets by time range from a reverse-ordered slice
func (c *Client) filterTweetsByTimeRange(tweetsFromLatestToEarliest []*Tweet, timeBegin time.Time, timeEnd time.Time) (cutMin bool, cutMax bool, res []*Tweet) {
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

////////////////////////////////////////////////////////////////////////////////
// Member Retrieval Operations
////////////////////////////////////////////////////////////////////////////////

// getMembers retrieves members from a timeline API endpoint
func (c *Client) getMembers(ctx context.Context, api timelineApi, instsPath string) ([]*User, error) {
	api.SetCursor("")
	itemContents, err := c.getTimelineItemContentsTillEnd(ctx, api, instsPath)
	if err != nil {
		return nil, err
	}
	return c.itemContentsToUsers(itemContents), nil
}

// itemContentsToUsers converts timeline item contents to User objects
func (c *Client) itemContentsToUsers(itemContents []gjson.Result) []*User {
	users := make([]*User, 0, len(itemContents))
	for _, ic := range itemContents {
		user_results := c.getResults(ic, timelineUser)
		if user_results.String() == "{}" {
			continue
		}
		u, err := c.parseUserJson(&user_results)
		if err != nil {
			log.WithFields(log.Fields{
				"user_results": user_results.String(),
				"reason":       err,
			}).Debugf("failed to parse user_results")
			continue
		}
		users = append(users, u)
	}
	return users
}
