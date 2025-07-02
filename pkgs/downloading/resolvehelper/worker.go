package resolvehelper

import (
	"sync"

	"github.com/WangWilly/xSync/pkgs/downloading/dtos/smartpathdto"
	"github.com/WangWilly/xSync/pkgs/utils"
)

type worker struct {
	pq                   *utils.Heap[*smartpathdto.UserSmartPath]
	syncedUserSmartPaths sync.Map // map[uint64]*smartpathdto.UserSmartPath
}

func newWorker(pq *utils.Heap[*smartpathdto.UserSmartPath]) *worker {
	return &worker{
		pq: pq,
	}
}

////////////////////////////////////////////////////////////////////////////////

func (w *worker) Pre() {

}
