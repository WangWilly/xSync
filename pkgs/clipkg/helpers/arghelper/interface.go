package arghelper

import (
	"context"

	"github.com/WangWilly/xSync/pkgs/commonpkg/clients/twitterclient"
	"github.com/tidwall/gjson"
)

type TwitterClient interface {
	GetUserById(ctx context.Context, userId uint64) (*twitterclient.User, error)
	GetUserByScreenName(ctx context.Context, screenName string) (*twitterclient.User, error)

	GetRawListByteById(ctx context.Context, listId uint64) (*gjson.Result, error)
	GetAllListMembers(ctx context.Context, listId uint64) ([]*twitterclient.User, error)

	GetAllFollowingMembers(ctx context.Context, userId uint64) ([]*twitterclient.User, error)
}
