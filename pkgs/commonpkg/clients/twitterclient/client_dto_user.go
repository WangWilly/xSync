package twitterclient

import (
	"fmt"
)

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

func (user *User) IsVisiable() bool {
	return user.Followstate == FS_FOLLOWING || !user.IsProtected
}

// Title returns a formatted string with the user's display name and screen name
func (user *User) Title() string {
	return fmt.Sprintf("%s(%s)", user.Name, user.ScreenName)
}

// IsVisible checks if the user's content is visible (either following or public account)
func (user *User) IsUserVisible() bool {
	return user.Followstate == FS_FOLLOWING || !user.IsProtected
}
