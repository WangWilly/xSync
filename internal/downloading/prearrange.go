package downloading

import (
	"context"
	"sync"

	"github.com/WangWilly/xSync/internal/database"
	"github.com/WangWilly/xSync/internal/downloading/dtos/smartpathdto"
	"github.com/WangWilly/xSync/internal/tasks"
	"github.com/WangWilly/xSync/internal/twitter"
	"github.com/WangWilly/xSync/internal/utils"
	"github.com/go-resty/resty/v2"
	"github.com/jmoiron/sqlx"
	log "github.com/sirupsen/logrus"
)

// userWithinListEntity represents a user within a list entity context
type userWithinListEntity struct {
	user *twitter.User
	leid *int
}

////////////////////////////////////////////////////////////////////////////////

/**
// Deprecated: BatchDownloadAny is deprecated and will be removed in future versions.
// BatchDownloadAny orchestrates the complete download process for lists and users
func BatchDownloadAny(
	ctx context.Context,
	client *resty.Client,
	db *sqlx.DB,
	task *tasks.Task,
	dir string,
	realDir string,
	autoFollow bool,
	additionalClients []*resty.Client,
) ([]*packedtweetdto.InEntity, error) {
	log.Debugln("start collecting users")

	ctx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)

	packgedUsers := make([]userWithinListEntity, 0)

	wg := sync.WaitGroup{}
	mtx := sync.Mutex{}
	for _, twitterPostList := range task.Lists {
		wg.Add(1)
		go func(lst twitter.ListBase) {
			defer wg.Done()
			res, err := syncLstAndGetMembers(ctx, client, db, lst, dir)
			if err != nil {
				cancel(err)
			}
			log.Debugf("members of %s: %d", lst.Title(), len(res))
			mtx.Lock()
			defer mtx.Unlock()
			packgedUsers = append(packgedUsers, res...)
		}(twitterPostList)
	}
	wg.Wait()
	if err := context.Cause(ctx); err != nil {
		return nil, err
	}

	for _, interestedTwitterUser := range task.Users {
		packgedUsers = append(packgedUsers, userWithinListEntity{user: interestedTwitterUser, leid: nil})
	}

	log.Debugln("collected users:", len(packgedUsers))
	return BatchUserDownload(ctx, client, db, packgedUsers, realDir, autoFollow, additionalClients)
}
*/

func WrapToUsersWithinListEntity(
	ctx context.Context,
	client *resty.Client,
	db *sqlx.DB,
	task *tasks.Task,
	rootDir string,
) ([]userWithinListEntity, error) {
	ctx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)

	userList := make([]userWithinListEntity, 0)

	wg := sync.WaitGroup{}
	mtx := sync.Mutex{}
	for _, twitterPostList := range task.Lists {
		wg.Add(1)
		go func(lst twitter.ListBase) {
			defer wg.Done()
			res, err := syncLstAndGetMembers(ctx, client, db, lst, rootDir)
			if err != nil {
				cancel(err)
			}
			log.Debugf("members of %s: %d", lst.Title(), len(res))
			mtx.Lock()
			defer mtx.Unlock()
			userList = append(userList, res...)
		}(twitterPostList)
	}
	wg.Wait()
	if err := context.Cause(ctx); err != nil {
		return nil, err
	}

	for _, interestedTwitterUser := range task.Users {
		userList = append(userList, userWithinListEntity{user: interestedTwitterUser, leid: nil})
	}

	return userList, nil
}

////////////////////////////////////////////////////////////////////////////////

// syncList updates the database record for a list
func syncList(db *sqlx.DB, list *twitter.List) error {
	listdb, err := database.GetLst(db, list.Id)
	if err != nil {
		return err
	}
	if listdb == nil {
		return database.CreateLst(db, &database.Lst{Id: list.Id, Name: list.Name, OwnerId: list.Creator.TwitterId})
	}
	return database.UpdateLst(db, &database.Lst{Id: list.Id, Name: list.Name, OwnerId: list.Creator.TwitterId})
}

func syncLstAndGetMembers(ctx context.Context, client *resty.Client, db *sqlx.DB, lst twitter.ListBase, dir string) ([]userWithinListEntity, error) {
	if v, ok := lst.(*twitter.List); ok {
		if err := syncList(db, v); err != nil {
			return nil, err
		}
	}

	// update lst path and record
	expectedTitle := utils.ToLegalWindowsFileName(lst.Title())
	entity, err := smartpathdto.NewListSmartPath(db, lst.GetId(), dir)
	if err != nil {
		return nil, err
	}
	if err := syncPath(entity, expectedTitle); err != nil {
		return nil, err
	}

	// get all members
	members, err := lst.GetMembers(ctx, client)
	if err != nil || len(members) == 0 {
		return nil, err
	}

	// bind lst entity to users for creating symlink
	packgedUsers := make([]userWithinListEntity, 0, len(members))
	eid := entity.Id()
	for _, user := range members {
		packgedUsers = append(packgedUsers, userWithinListEntity{user: user, leid: &eid})
	}
	return packgedUsers, nil
}
