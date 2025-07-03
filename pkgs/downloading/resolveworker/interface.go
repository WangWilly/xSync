package resolveworker

import (
	"context"

	"github.com/WangWilly/xSync/pkgs/downloading/dtos/dldto"
	"github.com/WangWilly/xSync/pkgs/downloading/dtos/smartpathdto"
	"github.com/WangWilly/xSync/pkgs/twitter"
	"github.com/WangWilly/xSync/pkgs/utils"
	"github.com/go-resty/resty/v2"
)

type MediaDownloadHelper interface {
	SafeDownload(ctx context.Context, client *resty.Client, meta dldto.TweetDlMeta) error
}

type HeapHelper interface {
	GetHeap() *utils.Heap[*smartpathdto.UserSmartPath]
	GetDepth(userSmartPath *smartpathdto.UserSmartPath) int
	GetUserByTwitterId(twitterId uint64) *twitter.User
}
