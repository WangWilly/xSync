package resolveworker

import (
	"context"

	"github.com/WangWilly/xSync/pkgs/commonpkg/clients/twitterclient"
	"github.com/WangWilly/xSync/pkgs/commonpkg/utils"
	"github.com/WangWilly/xSync/pkgs/downloading/dtos/dldto"
	"github.com/WangWilly/xSync/pkgs/downloading/dtos/smartpathdto"
)

type MediaDownloadHelper interface {
	SafeDownload(ctx context.Context, client *twitterclient.Client, meta *dldto.NewEntity) error
}

type HeapHelper interface {
	GetHeap() *utils.Heap[*smartpathdto.UserSmartPath]
	GetDepth(userSmartPath *smartpathdto.UserSmartPath) int
	GetUserByTwitterId(twitterId uint64) *twitterclient.User
}
