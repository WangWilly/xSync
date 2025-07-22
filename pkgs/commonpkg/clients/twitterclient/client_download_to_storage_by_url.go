package twitterclient

import (
	"context"
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
		logger.WithError(err).Errorf("failed to download media from URL %s", url)
		// 403: Dmcaed
		if utils.IsStatusCode(err, 404) || utils.IsStatusCode(err, 403) {
			return nil
		}
		return err
	}
	return nil
}

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
