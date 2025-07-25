package twitterclient

import (
	"fmt"

	"github.com/tidwall/gjson"
)

////////////////////////////////////////////////////////////////////////////////

const (
	TITLED_TYPE_TWITTER_USER      = "twitter_user"
	TITLED_TYPE_TWITTER_LIST      = "twitter_list"
	TITLED_TYPE_TWITTER_FOLLOWERS = "twitter_followers"
)

////////////////////////////////////////////////////////////////////////////////

type TitledUserList struct {
	Type string

	Id    uint64
	Title string
	Users []*User

	BelongsTo   *User
	TwitterName string
}

////////////////////////////////////////////////////////////////////////////////

func NewTitledUserListByUser(user *User) *TitledUserList {
	return &TitledUserList{
		Type:        TITLED_TYPE_TWITTER_USER,
		Id:          user.TwitterId,
		Title:       fmt.Sprintf("%s(%s)", user.Name, user.ScreenName),
		Users:       []*User{user},
		BelongsTo:   user,
		TwitterName: user.Name,
	}
}

////////////////////////////////////////////////////////////////////////////////

func NewTulByRawListByteAndMembers(gjson *gjson.Result, members []*User) (*TitledUserList, error) {
	user_results := gjson.Get("user_results")
	creator, err := parseUserJsonForNewTulByTwitterListId(&user_results)
	if err != nil {
		return nil, err
	}
	id_str := gjson.Get("id_str")
	name := gjson.Get("name")

	return &TitledUserList{
		Type:        TITLED_TYPE_TWITTER_LIST,
		Id:          id_str.Uint(),
		Title:       fmt.Sprintf("%s(%d)", name.String(), id_str.Uint()),
		Users:       members,
		BelongsTo:   creator,
		TwitterName: name.String(),
	}, nil
}

func parseUserJsonForNewTulByTwitterListId(userJson *gjson.Result) (*User, error) {
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

func NewTulByUserAndFollowers(user *User, followers []*User) *TitledUserList {
	return &TitledUserList{
		Type:        TITLED_TYPE_TWITTER_FOLLOWERS,
		Id:          user.TwitterId,
		Title:       fmt.Sprintf("%s(%s)", user.Name, user.ScreenName),
		Users:       followers,
		BelongsTo:   user,
		TwitterName: user.Name,
	}
}
