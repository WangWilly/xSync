package heaphelper

import (
	"errors"
	"sync"

	"github.com/WangWilly/xSync/pkgs/commonpkg/clients/twitterclient"
	"github.com/WangWilly/xSync/pkgs/commonpkg/utils"
	"github.com/WangWilly/xSync/pkgs/downloading/dtos/smartpathdto"
	log "github.com/sirupsen/logrus"
)

////////////////////////////////////////////////////////////////////////////////

type userSmartPathToDepthMap map[*smartpathdto.UserSmartPath]int

////////////////////////////////////////////////////////////////////////////////

func New(metas []twitterclient.TitledUserList, smartPathes []*smartpathdto.UserSmartPath) (*helper, error) {
	logger := log.WithField("function", "NewHeapHelper")

	uidToUserMap := make(map[uint64]*twitterclient.User)
	for _, meta := range metas {
		for _, user := range meta.Users {
			if user == nil {
				logger.Warnln("skipping nil user in meta", meta.Type)
				continue
			}
			uidToUserMap[user.TwitterId] = user
		}
	}

	userSmartPathToDepthMap := make(userSmartPathToDepthMap)
	for _, sp := range smartPathes {
		if sp == nil {
			logger.Warnln("skipping nil user smart path")
			continue
		}

		userSmartPathToDepthMap[sp] = sp.Depth
	}

	lessFunc := func(lhs, rhs *smartpathdto.UserSmartPath) bool {
		luser, ruser := uidToUserMap[lhs.TwitterId()], uidToUserMap[rhs.TwitterId()]
		lOnlyMater := luser.IsProtected && luser.Followstate == twitterclient.FS_FOLLOWING
		rOnlyMaster := ruser.IsProtected && ruser.Followstate == twitterclient.FS_FOLLOWING

		if lOnlyMater == rOnlyMaster {
			return userSmartPathToDepthMap[lhs] > userSmartPathToDepthMap[rhs]
		}
		return lOnlyMater // 优先让 master 获取只有他能看到的
	}

	userUserSmartPathHeap := utils.NewByHeapify(smartPathes, lessFunc)

	if userUserSmartPathHeap.Empty() {
		logger.Infoln("no user to process")
		return nil, errors.New("no user to process")
	}

	return &helper{
		uidToUserMap:         uidToUserMap,
		userSmartPathToDepth: userSmartPathToDepthMap,
		heap:                 userUserSmartPathHeap,
		mtx:                  sync.Mutex{},
	}, nil
}

////////////////////////////////////////////////////////////////////////////////

func (h *helper) GetHeap() *utils.Heap[*smartpathdto.UserSmartPath] {
	h.mtx.Lock()
	defer h.mtx.Unlock()

	if h.heap == nil {
		panic("heap is not initialized, call MakeHeap first")
	}
	return h.heap
}

func (h *helper) GetDepth(userSmartPath *smartpathdto.UserSmartPath) int {
	h.mtx.Lock()
	defer h.mtx.Unlock()

	if depth, ok := h.userSmartPathToDepth[userSmartPath]; ok {
		return depth
	}
	return 0
}

func (h *helper) GetUserByTwitterId(twitterId uint64) *twitterclient.User {
	h.mtx.Lock()
	defer h.mtx.Unlock()

	if user, ok := h.uidToUserMap[twitterId]; ok {
		return user
	}
	return nil
}
