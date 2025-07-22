package twitterclient

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
)

////////////////////////////////////////////////////////////////////////////////

const (
	timelineTweet = iota // Timeline item type for tweets
	timelineUser         // Timeline item type for users
)

////////////////////////////////////////////////////////////////////////////////

// itemContentsToUsers converts timeline item contents to User objects
func itemContentsToUsers(itemContents []gjson.Result) []*User {
	logger := log.WithField("caller", "Client.itemContentsToUsers")

	users := make([]*User, 0, len(itemContents))
	for _, ic := range itemContents {
		user_results := getResults(ic, timelineUser)
		if user_results.String() == "{}" {
			continue
		}

		u, err := parseUserJson(&user_results)
		if err != nil {
			logger.
				WithFields(log.Fields{
					"user_results": user_results.String(),
					"reason":       err,
				}).
				Debugf("failed to parse user_results")
			continue
		}
		users = append(users, u)
	}
	return users
}

////////////////////////////////////////////////////////////////////////////////

// getResults extracts results based on item type (tweet or user)
func getResults(itemContent gjson.Result, itemType int) gjson.Result {
	switch itemType {
	case timelineTweet:
		return itemContent.Get("tweet_results")
	case timelineUser:
		return itemContent.Get("user_results")
	}

	panic(fmt.Sprintf("invalid itemContent: %s", itemContent.String()))
}

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
