package dldto

import (
	"github.com/WangWilly/xSync/pkgs/downloading/dtos/smartpathdto"
	"github.com/WangWilly/xSync/pkgs/twitter"
)

// TweetDlMeta represents a tweet with its associated download path
type TweetDlMeta interface {
	GetTweet() *twitter.Tweet
	GetPath() string
	GetUserSmartPath() smartpathdto.SmartPath
}
