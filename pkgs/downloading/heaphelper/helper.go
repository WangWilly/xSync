package heaphelper

import (
	"sync"

	"github.com/WangWilly/xSync/pkgs/commonpkg/clients/twitterclient"
	"github.com/WangWilly/xSync/pkgs/commonpkg/utils"
	"github.com/WangWilly/xSync/pkgs/downloading/dtos/smartpathdto"
)

type helper struct {
	uidToUserMap         map[uint64]*twitterclient.User
	userSmartPathToDepth map[*smartpathdto.UserSmartPath]int

	heap *utils.Heap[*smartpathdto.UserSmartPath]
	mtx  sync.Mutex
}
