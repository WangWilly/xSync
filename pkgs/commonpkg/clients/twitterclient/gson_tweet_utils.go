package twitterclient

import (
	"fmt"
	"time"

	"github.com/tidwall/gjson"
)

// itemContentsToTweets converts timeline item contents to Tweet objects
func itemContentsToTweets(itemContents []gjson.Result) []*Tweet {
	res := make([]*Tweet, 0, len(itemContents))
	for _, itemContent := range itemContents {
		tweetResults := getResults(itemContent, timelineTweet)
		if tw := parseTweetResults(&tweetResults); tw != nil {
			res = append(res, tw)
		}
	}
	return res
}

// parseTweetResults parses tweet data from Twitter API JSON response
func parseTweetResults(tweet_results *gjson.Result) *Tweet {
	var tweet Tweet
	var err error = nil

	result := tweet_results.Get("result")
	if !result.Exists() || result.Get("__typename").String() == "TweetTombstone" {
		return nil
	}
	if result.Get("__typename").String() == "TweetWithVisibilityResults" {
		result = result.Get("tweet")
	}
	legacy := result.Get("legacy")
	// TODO: 利用 rest_id 重新获取推文信息
	if !legacy.Exists() {
		return nil
	}
	user_results := result.Get("core.user_results")

	tweet.Id = result.Get("rest_id").Uint()
	tweet.Text = legacy.Get("full_text").String()
	tweet.Creator, _ = parseUserJson(&user_results)
	tweet.CreatedAt, err = time.Parse(time.RubyDate, legacy.Get("created_at").String())
	if err != nil {
		panic(fmt.Errorf("invalid time format %v", err))
	}
	media := legacy.Get("extended_entities.media")
	if media.Exists() {
		tweet.Urls = getUrlsFromMedia(&media)
	}
	return &tweet
}

// getUrlsFromMedia extracts media URLs from tweet media entities
func getUrlsFromMedia(media *gjson.Result) []string {
	results := []string{}
	for _, m := range media.Array() {
		typ := m.Get("type").String()
		switch typ {
		case "video", "animated_gif":
			results = append(results, m.Get("video_info.variants.@reverse.0.url").String())
		case "photo":
			results = append(results, m.Get("media_url_https").String())
		}
	}
	return results
}
