package packedtweetdto

import "github.com/WangWilly/xSync/pkgs/twitter"

// InDir represents a tweet with a specific directory path
type InDir struct {
	tweet *twitter.Tweet
	path  string
}

func (pt InDir) GetTweet() *twitter.Tweet {
	return pt.tweet
}

func (pt InDir) GetPath() string {
	return pt.path
}
