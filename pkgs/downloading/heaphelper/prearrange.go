package heaphelper

import (
	"context"
	"os"
	"sync"

	"github.com/WangWilly/xSync/pkgs/clients/twitterclient"
	"github.com/WangWilly/xSync/pkgs/database"
	"github.com/WangWilly/xSync/pkgs/downloading/dtos/smartpathdto"
	"github.com/WangWilly/xSync/pkgs/tasks"
	"github.com/WangWilly/xSync/pkgs/utils"
	"github.com/jmoiron/sqlx"
	log "github.com/sirupsen/logrus"
)

// UserWithinListEntity represents a user within a list entity context
type UserWithinListEntity struct {
	User *twitterclient.User
	Leid *int
}

////////////////////////////////////////////////////////////////////////////////

func WrapToUsersWithinListEntity(
	ctx context.Context,
	client *twitterclient.Client,
	db *sqlx.DB,
	task *tasks.Task,
	rootDir string,
) ([]UserWithinListEntity, error) {
	ctx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)

	userList := make([]UserWithinListEntity, 0)

	wg := sync.WaitGroup{}
	mtx := sync.Mutex{}
	for _, twitterPostList := range task.Lists {
		wg.Add(1)
		go func(lst twitterclient.ListBase) {
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
		userList = append(userList, UserWithinListEntity{User: &twitterclient.User{
			TwitterId:   interestedTwitterUser.TwitterId,
			Name:        interestedTwitterUser.Name,
			ScreenName:  interestedTwitterUser.ScreenName,
			IsProtected: interestedTwitterUser.IsProtected,
			Followstate: twitterclient.FollowState(interestedTwitterUser.Followstate),
			MediaCount:  interestedTwitterUser.MediaCount,
			Muting:      interestedTwitterUser.Muting,
			Blocking:    interestedTwitterUser.Blocking,
		}, Leid: nil})
	}

	return userList, nil
}

////////////////////////////////////////////////////////////////////////////////

// syncList updates the database record for a list
func syncList(db *sqlx.DB, list *twitterclient.List) error {
	listdb, err := database.GetLst(db, list.Id)
	if err != nil {
		return err
	}
	if listdb == nil {
		return database.CreateLst(db, &database.Lst{Id: list.Id, Name: list.Name, OwnerId: list.Creator.TwitterId})
	}
	return database.UpdateLst(db, &database.Lst{Id: list.Id, Name: list.Name, OwnerId: list.Creator.TwitterId})
}

func syncLstAndGetMembers(ctx context.Context, client *twitterclient.Client, db *sqlx.DB, lst twitterclient.ListBase, dir string) ([]UserWithinListEntity, error) {
	if v, ok := lst.(*twitterclient.List); ok {
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
	if err := SyncPath(entity, expectedTitle); err != nil {
		return nil, err
	}

	// get all members
	members, err := lst.GetMembers(ctx, client)
	if err != nil || len(members) == 0 {
		return nil, err
	}

	// bind lst entity to users for creating symlink
	packgedUsers := make([]UserWithinListEntity, 0, len(members))
	eid := entity.Id()
	for _, user := range members {
		packgedUsers = append(
			packgedUsers,
			UserWithinListEntity{
				User: &twitterclient.User{
					TwitterId:   user.TwitterId,
					Name:        user.Name,
					ScreenName:  user.ScreenName,
					IsProtected: user.IsProtected,
					Followstate: twitterclient.FollowState(user.Followstate),
					MediaCount:  user.MediaCount,
					Muting:      user.Muting,
					Blocking:    user.Blocking,
				},
				Leid: &eid,
			},
		)
	}
	return packgedUsers, nil
}

func SyncPath(path smartpathdto.SmartPath, expectedName string) error {
	if !path.IsSyncToDb() {
		return path.Create(expectedName)
	}

	if path.Name() != expectedName {
		return path.Rename(expectedName)
	}

	p, err := path.Path()
	if err != nil {
		return err
	}

	return os.MkdirAll(p, 0755)
}
