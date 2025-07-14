package commandline

import (
	"context"
	"strconv"
	"strings"

	"github.com/WangWilly/xSync/pkgs/commonpkg/clients/twitterclient"
)

////////////////////////////////////////////////////////////////////////////////
// Command Line Argument Structures
////////////////////////////////////////////////////////////////////////////////

// UserArgs represents user arguments for CLI (user IDs and screen names)
type UserArgs struct {
	id         []uint64
	screenName []string
}

// GetUser retrieves users from Twitter API based on user IDs and screen names
func (u *UserArgs) GetUser(ctx context.Context, client *twitterclient.Client) ([]*twitterclient.User, error) {
	users := []*twitterclient.User{}
	for _, id := range u.id {
		usr, err := client.GetUserById(ctx, id)
		if err != nil {
			return nil, err
		}
		users = append(users, usr)
	}

	for _, screenName := range u.screenName {
		usr, err := client.GetUserByScreenName(ctx, screenName)
		if err != nil {
			return nil, err
		}
		users = append(users, usr)
	}
	return users, nil
}

// Set implements flag.Value interface for UserArgs
func (u *UserArgs) Set(str string) error {
	if u.id == nil {
		u.id = make([]uint64, 0)
		u.screenName = make([]string, 0)
	}

	id, err := strconv.ParseUint(str, 10, 64)
	if err != nil {
		str, _ := strings.CutPrefix(str, "@")
		u.screenName = append(u.screenName, str)
	} else {
		u.id = append(u.id, id)
	}
	return nil
}

// String implements flag.Value interface for UserArgs
func (u *UserArgs) String() string {
	return "string"
}

////////////////////////////////////////////////////////////////////////////////
// Integer Arguments Base Structure
////////////////////////////////////////////////////////////////////////////////

// IntArgs represents integer arguments for CLI
type IntArgs struct {
	id []uint64
}

// Set implements flag.Value interface for IntArgs
func (l *IntArgs) Set(str string) error {
	if l.id == nil {
		l.id = make([]uint64, 0)
	}

	id, err := strconv.ParseUint(str, 10, 64)
	if err != nil {
		return err
	}
	l.id = append(l.id, id)
	return nil
}

// String implements flag.Value interface for IntArgs
func (a *IntArgs) String() string {
	return "string array"
}

////////////////////////////////////////////////////////////////////////////////
// List Arguments Structure
////////////////////////////////////////////////////////////////////////////////

// ListArgs represents list arguments for CLI
type ListArgs struct {
	IntArgs
}

// GetList retrieves Twitter lists based on list IDs
func (l ListArgs) GetList(ctx context.Context, client *twitterclient.Client) ([]*twitterclient.List, error) {
	lists := []*twitterclient.List{}
	for _, id := range l.id {
		list, err := client.GetList(ctx, id)
		if err != nil {
			return nil, err
		}
		lists = append(lists, list)
	}
	return lists, nil
}
