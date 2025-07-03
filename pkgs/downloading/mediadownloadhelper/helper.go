package mediadownloadhelper

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/WangWilly/xSync/pkgs/downloading/dtos/dldto"
	"github.com/WangWilly/xSync/pkgs/twitter"
	"github.com/WangWilly/xSync/pkgs/utils"
	"github.com/go-resty/resty/v2"
	"github.com/gookit/color"
)

type helper struct {
	mutex sync.Mutex
}

func NewHelper() *helper {
	return &helper{
		mutex: sync.Mutex{},
	}
}

func (h *helper) SafeDownload(ctx context.Context, client *resty.Client, meta dldto.TweetDlMeta) error {
	path := meta.GetPath()
	if path == "" {
		return errors.New("path is empty")

	}

	err := h.download(ctx, client, path, meta.GetTweet())
	// 403: Dmcaed
	if err != nil && !(utils.IsStatusCode(err, 404) || utils.IsStatusCode(err, 403)) {
		return err
	}

	return nil
}

// download downloads all media files for a given tweet
// 任何一个 url 下载失败直接返回
// TODO: 要么全做，要么不做
func (h *helper) download(ctx context.Context, client *resty.Client, dir string, tweet *twitter.Tweet) error {
	text := utils.ToLegalWindowsFileName(tweet.Text)

	for _, u := range tweet.Urls {
		ext, err := utils.GetExtFromUrl(u)
		if err != nil {
			return err
		}

		// 请求
		resp, err := client.R().SetContext(ctx).SetQueryParam("name", "4096x4096").Get(u)
		if err != nil {
			return err
		}

		h.mutex.Lock()
		path, err := utils.UniquePath(filepath.Join(dir, text+ext))
		if err != nil {
			h.mutex.Unlock()
			return err
		}
		file, err := os.Create(path)
		h.mutex.Unlock()
		if err != nil {
			return err
		}

		defer os.Chtimes(path, time.Time{}, tweet.CreatedAt)
		defer file.Close()

		_, err = file.Write(resp.Body())
		if err != nil {
			return err
		}
	}

	fmt.Printf("%s %s\n", color.FgLightMagenta.Render("["+tweet.Creator.Title()+"]"), text)
	return nil
}
