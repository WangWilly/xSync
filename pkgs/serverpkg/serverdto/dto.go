package serverdto

import (
	"time"

	"github.com/WangWilly/xSync/pkgs/commonpkg/model"
)

// UserStats represents user statistics for display
type UserStats struct {
	User           *model.User
	Entity         *model.UserEntity
	TotalMedias    int
	LatestActivity time.Time
}

// DashboardData represents data for the dashboard
type DashboardData struct {
	Users       []*UserStats
	TotalUsers  int
	TotalTweets int
	TotalMedias int
	LastUpdated time.Time
}

// TweetData represents tweet data for display
type TweetData struct {
	UserName string
	Tweets   []map[string]interface{}
}

// TweetWithMedia represents a tweet with its associated media files
type TweetWithMedia struct {
	ID         int64     `json:"id"`
	Content    string    `json:"content"`
	TweetTime  time.Time `json:"tweet_time"`
	MediaFiles []string  `json:"media_files"`
	MediaCount int       `json:"media_count"`
}

// MediaResponse represents media response data
type MediaResponse struct {
	UserName string         `json:"user_name"`
	Medias   []*model.Media `json:"medias"`
}

// TweetsWithMediaResponse represents tweets with media response data
type TweetsWithMediaResponse struct {
	User   *model.User      `json:"user"`
	Tweets []TweetWithMedia `json:"tweets"`
}
