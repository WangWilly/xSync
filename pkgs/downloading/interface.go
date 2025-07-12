package downloading

import (
	"context"

	"github.com/WangWilly/xSync/pkgs/clients/twitterclient"
	"github.com/WangWilly/xSync/pkgs/downloading/dtos/smartpathdto"
	"github.com/WangWilly/xSync/pkgs/utils"
	"github.com/jmoiron/sqlx"
)

type HeapHelper interface {
	GetDepth(userSmartPath *smartpathdto.UserSmartPath) int
	GetHeap() *utils.Heap[*smartpathdto.UserSmartPath]
	GetUserByTwitterId(twitterId uint64) *twitterclient.User
	MakeHeap(ctx context.Context, db *sqlx.DB, dir string, autoFollow bool) error
}
