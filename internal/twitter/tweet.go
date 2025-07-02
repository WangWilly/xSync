package twitter

import (
	"fmt"
	"time"

	"github.com/tidwall/gjson"
)

////////////////////////////////////////////////////////////////////////////////
// Tweet Structure and Types
////////////////////////////////////////////////////////////////////////////////

// Tweet represents a Twitter tweet with its metadata and content
type Tweet struct {
	Id        uint64    // Unique identifier for the tweet
	Text      string    // Tweet content text
	CreatedAt time.Time // When the tweet was created
	Creator   *User     // User who created the tweet
	Urls      []string  // Media URLs associated with the tweet
}

////////////////////////////////////////////////////////////////////////////////
// Tweet Parsing and Processing
////////////////////////////////////////////////////////////////////////////////

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

// ended audio space

/*
id = ?
media_key = audio_space_by_id()
live_video_stream = get https://x.com/i/api/1.1/live_video_stream/status/{media_key}?client=web&use_syndication_guest_id=false&cookie_set_host=x.com
playlist = live_video_stream.source.location
handle playlist...
*/
