package twitterclient

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
)

////////////////////////////////////////////////////////////////////////////////
// List Interfaces and Base Types
////////////////////////////////////////////////////////////////////////////////

// ListBase defines the interface for list-like objects that can provide members
type ListBase interface {
	GetMembers(ctx context.Context, client *Client) ([]*User, error)
	GetId() int64
	Title() string
}

////////////////////////////////////////////////////////////////////////////////
// List Structure and Core Operations
////////////////////////////////////////////////////////////////////////////////

// List represents a Twitter list with its metadata and creator information
type List struct {
	Id          uint64 // Unique identifier for the list
	MemberCount int    // Number of members in the list
	Name        string // Display name of the list
	Creator     *User  // User who created the list
}

// GetId returns the list ID as int64
func (list *List) GetId() int64 {
	return int64(list.Id)
}

// Title returns a formatted title for the list
func (list *List) Title() string {
	return fmt.Sprintf("%s(%d)", list.Name, list.Id)
}

// GetMembers retrieves all members of the list using the client
func (list *List) GetMembers(ctx context.Context, client *Client) ([]*User, error) {
	return client.GetAllListMembers(ctx, list.Id)
}

////////////////////////////////////////////////////////////////////////////////
// User Following Structure and Operations
////////////////////////////////////////////////////////////////////////////////

// UserFollowing represents a user's following list as a ListBase implementation
type UserFollowing struct {
	creator *User
}

// NewUserFollowing creates a new UserFollowing instance
func NewUserFollowing(creator *User) *UserFollowing {
	return &UserFollowing{
		creator: creator,
	}
}

// GetMembers retrieves all users that the creator is following
func (fo *UserFollowing) GetMembers(ctx context.Context, client *Client) ([]*User, error) {
	return client.GetAllFollowingMembers(ctx, fo.creator.TwitterId)
}

// GetId returns a negative ID to distinguish from regular lists
func (fo *UserFollowing) GetId() int64 {
	return -int64(fo.creator.TwitterId)
}

// Title returns a formatted title for the following list
func (fo *UserFollowing) Title() string {
	name := fmt.Sprintf("%s's Following", fo.creator.ScreenName)
	return name
}

////////////////////////////////////////////////////////////////////////////////
// JSON Parsing and Data Conversion
////////////////////////////////////////////////////////////////////////////////

// parseList parses a Twitter list from JSON response data
func (c *Client) parseList(list *gjson.Result) (*List, error) {
	if !list.Exists() {
		return nil, fmt.Errorf("the list doesn't exist")
	}
	user_results := list.Get("user_results")
	creator, err := c.parseUserJson(&user_results)
	if err != nil {
		return nil, err
	}
	id_str := list.Get("id_str")
	member_count := list.Get("member_count")
	name := list.Get("name")

	result := List{}
	result.Creator = creator
	result.Id = id_str.Uint()
	result.MemberCount = int(member_count.Int())
	result.Name = name.String()
	return &result, nil
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
