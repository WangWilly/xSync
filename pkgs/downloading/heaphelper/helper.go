package heaphelper

import (
	"sync"

	"github.com/WangWilly/xSync/pkgs/commonpkg/clients/twitterclient"
	"github.com/WangWilly/xSync/pkgs/commonpkg/utils"
	"github.com/WangWilly/xSync/pkgs/downloading/dtos/smartpathdto"
)

type helper struct {
	uidToUserMap         map[uint64]*twitterclient.User
	users                []UserWithinListEntity // TODO: rm
	userSmartPathToDepth map[*smartpathdto.UserSmartPath]int

	userRepo             UserRepo               // TODO: rm
	listRepo             ListRepo               // TODO: rm
	twitterClientManager *twitterclient.Manager // TODO: rm

	syncedUserSmartPaths *utils.SyncMap[uint64, *smartpathdto.UserSmartPath]   // TODO: rm
	syncedListToUsersMap *utils.SyncMap[int, *utils.SyncMap[uint64, struct{}]] // TODO: rm

	heap *utils.Heap[*smartpathdto.UserSmartPath]
	mtx  sync.Mutex
}
