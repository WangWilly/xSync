package packedtweetdto

import (
	"github.com/WangWilly/xSync/internal/downloading/dtos/smartpathdto"
	"github.com/WangWilly/xSync/internal/twitter"
)

// InEntity represents a tweet associated with a user entity
type InEntity struct {
	Tweet  *twitter.Tweet
	Entity *smartpathdto.UserSmartPath
}

func (pt InEntity) GetTweet() *twitter.Tweet {
	return pt.Tweet
}

func (pt InEntity) GetPath() string {
	path, err := pt.Entity.Path()
	if err != nil {
		return ""
	}
	return path
}
