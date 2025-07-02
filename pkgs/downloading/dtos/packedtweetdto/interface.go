package packedtweetdto

import "github.com/WangWilly/xSync/pkgs/twitter"

// PackedTweet represents a tweet with its associated download path
type PackedTweet interface {
	GetTweet() *twitter.Tweet
	GetPath() string
}
