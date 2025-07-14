package dldto

import (
	"github.com/WangWilly/xSync/pkgs/commonpkg/clients/twitterclient"
	"github.com/WangWilly/xSync/pkgs/downloading/dtos/smartpathdto"
)

// NewEntity represents a tweet associated with a user entity
type NewEntity struct {
	Tweet  *twitterclient.Tweet
	Entity *smartpathdto.UserSmartPath
}

func (pt NewEntity) GetTweet() *twitterclient.Tweet {
	return pt.Tweet
}

func (pt NewEntity) GetPath() string {
	path, err := pt.Entity.Path()
	if err != nil {
		return ""
	}
	return path
}

func (pt NewEntity) GetUserSmartPath() smartpathdto.SmartPath {
	return pt.Entity
}
