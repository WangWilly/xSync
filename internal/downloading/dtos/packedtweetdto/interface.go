package packedtweetdto

import "github.com/WangWilly/xSync/internal/twitter"

// PackedTweet represents a tweet with its associated download path
type PackedTweet interface {
	GetTweet() *twitter.Tweet
	GetPath() string
}
