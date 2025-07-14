package dtos

import (
	"time"

	"github.com/WangWilly/xSync/pkgs/commonpkg/clients/twitterclient"
)

// TODO:
// Tweet represents a Twitter tweet with its metadata and content
type TweetDto struct {
	Id        uint64    // Unique identifier for the tweet
	Text      string    // Tweet content text
	CreatedAt time.Time // When the tweet was created
	Creator   *UserDto  // User who created the tweet
	Urls      []string  // Media URLs associated with the tweet
}

func NewTweetDtoFromClient(tweet *twitterclient.Tweet) *TweetDto {
	if tweet == nil {
		return nil
	}

	return &TweetDto{
		Id:        tweet.Id,
		Text:      tweet.Text,
		CreatedAt: tweet.CreatedAt,
		Creator:   NewClientUserDtoFromClient(tweet.Creator),
		Urls:      tweet.Urls,
	}
}

func (tweetDto *TweetDto) GetTwitterClientTweet() *twitterclient.Tweet {
	if tweetDto == nil {
		return nil
	}

	return &twitterclient.Tweet{
		Id:        tweetDto.Id,
		Text:      tweetDto.Text,
		CreatedAt: tweetDto.CreatedAt,
		Creator:   tweetDto.Creator.GetTwitterClientUser(),
		Urls:      tweetDto.Urls,
	}
}

////////////////////////////////////////////////////////////////////////////////

// User represents a Twitter user with their profile information and relationship status
type UserDto struct {
	TwitterId    uint64 // User's unique identifier
	Name         string // Display name
	ScreenName   string // Username (handle)
	IsProtected  bool   // Whether the account is protected/private
	FriendsCount int    // Number of accounts this user follows
	Followstate  int    // Current follow relationship status
	MediaCount   int    // Number of media posts by this user
	Muting       bool   // Whether this user is muted
	Blocking     bool   // Whether this user is blocked
}

func NewClientUserDtoFromClient(user *twitterclient.User) *UserDto {
	if user == nil {
		return nil
	}

	return &UserDto{
		TwitterId:    user.TwitterId,
		Name:         user.Name,
		ScreenName:   user.ScreenName,
		IsProtected:  user.IsProtected,
		FriendsCount: user.FriendsCount,
		Followstate:  int(user.Followstate),
		MediaCount:   user.MediaCount,
		Muting:       user.Muting,
		Blocking:     user.Blocking,
	}
}

func (userDto *UserDto) GetTwitterClientUser() *twitterclient.User {
	if userDto == nil {
		return nil
	}

	return &twitterclient.User{
		TwitterId:    userDto.TwitterId,
		Name:         userDto.Name,
		ScreenName:   userDto.ScreenName,
		IsProtected:  userDto.IsProtected,
		FriendsCount: userDto.FriendsCount,
		Followstate:  twitterclient.FollowState(userDto.Followstate),
		MediaCount:   userDto.MediaCount,
		Muting:       userDto.Muting,
		Blocking:     userDto.Blocking,
	}
}
