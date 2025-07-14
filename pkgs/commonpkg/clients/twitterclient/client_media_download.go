package twitterclient

import (
	"context"
	"fmt"
	"time"

	"github.com/WangWilly/xSync/pkgs/commonpkg/utils"
	log "github.com/sirupsen/logrus"
)

// MediaData represents downloaded media content with metadata
type MediaData struct {
	URL       string    // Original URL
	Content   []byte    // Downloaded content
	Extension string    // File extension
	CreatedAt time.Time // Tweet creation time
}

////////////////////////////////////////////////////////////////////////////////

// DownloadTweetMediaContent downloads all media content from a tweet without saving to disk
// Returns a slice of MediaData containing the downloaded content and metadata
func (c *Client) DownloadTweetMediaContent(ctx context.Context, tweet *Tweet, quality string) ([]*MediaData, error) {
	if tweet == nil {
		return nil, fmt.Errorf("tweet cannot be nil")
	}

	if len(tweet.Urls) == 0 {
		return []*MediaData{}, nil // No media to download
	}

	if quality == "" {
		quality = "4096x4096"
	}

	var mediaData []*MediaData

	// Download each media URL
	for _, url := range tweet.Urls {
		data, err := c.downloadMediaContent(ctx, url, quality, tweet.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to download media from URL %s: %w", url, err)
		}
		mediaData = append(mediaData, data)
	}

	return mediaData, nil
}

// downloadMediaContent downloads media content from a single URL without saving to disk
func (c *Client) downloadMediaContent(ctx context.Context, url, quality string, createdAt time.Time) (*MediaData, error) {
	// Get file extension from URL
	ext, err := utils.GetExtFromUrl(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get extension from URL: %w", err)
	}

	// Make HTTP request with quality parameter
	req := c.restyClient.R().SetContext(ctx)
	if quality != "" {
		req = req.SetQueryParam("name", quality)
	}

	resp, err := req.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to download from URL: %w", err)
	}

	return &MediaData{
		URL:       url,
		Content:   resp.Body(),
		Extension: ext,
		CreatedAt: createdAt,
	}, nil
}

////////////////////////////////////////////////////////////////////////////////

func (c *Client) MustDownloadToStorageByUrl(ctx context.Context, url, targetPath, quality string) error {
	logger := log.WithFields(log.Fields{
		"caller":  "Client.MustDownloadToStorageByUrl",
		"client":  c.screenName,
		"url":     url,
		"target":  targetPath,
		"quality": quality,
	})

	if quality == "" {
		quality = "4096x4096"
	}

	resp, err := c.restyClient.R().
		SetContext(ctx).
		SetOutput(targetPath).
		Get(url)
	if err != nil {
		return err
	}
	if resp.StatusCode() != 200 {
		logger.
			WithFields(log.Fields{
				"status_code": resp.StatusCode(),
			}).
			Error("media download returned non-200 status")
		return nil
	}

	logger.WithFields(log.Fields{
		"size": len(resp.Body()),
	}).Debug("successfully downloaded media to database location")
	return nil
}

func (c *Client) DownloadToStorageByUrl(ctx context.Context, url, targetPath, quality string) error {
	logger := log.WithFields(log.Fields{
		"caller":  "Client.DownloadToStorageByUrl",
		"client":  c.screenName,
		"url":     url,
		"target":  targetPath,
		"quality": quality,
	})

	err := c.MustDownloadToStorageByUrl(ctx, url, targetPath, quality)
	if err != nil {
		// 403: Dmcaed
		if utils.IsStatusCode(err, 404) || utils.IsStatusCode(err, 403) {
			logger.WithError(err).Errorf("failed to download media from URL %s", url)
			return nil
		}
		return fmt.Errorf("failed to download media from URL %s: %w", url, err)
	}
	return nil
}
