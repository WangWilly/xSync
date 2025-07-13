package heaphelper

import (
	"context"
	"sync"

	"github.com/WangWilly/xSync/pkgs/clients/twitterclient"
	"github.com/WangWilly/xSync/pkgs/downloading/dtos/smartpathdto"
	"github.com/WangWilly/xSync/pkgs/model"
	"github.com/WangWilly/xSync/pkgs/tasks"
	"github.com/WangWilly/xSync/pkgs/utils"
	"github.com/jmoiron/sqlx"
	log "github.com/sirupsen/logrus"
)

////////////////////////////////////////////////////////////////////////////////

// UserWithinListEntity represents a user within a list entity context (nil if not in a list)
type UserWithinListEntity struct {
	User        *twitterclient.User
	MaybeListId *int
}

////////////////////////////////////////////////////////////////////////////////

func (h *helper) getUsersWithinListEntity(
	ctx context.Context,
	client *twitterclient.Client,
	db *sqlx.DB,
	task *tasks.Task,
	rootDir string,
) ([]UserWithinListEntity, error) {
	logger := log.WithField("caller", "heaphelper.GetUsersWithinListEntity")

	userList, err := h.parseLists(ctx, client, db, task.Lists, rootDir)
	if err != nil {
		logger.Errorln("failed to parse list:", err)
		return nil, err
	}

	for _, twitterClientUser := range task.Users {
		userList = append(
			userList,
			UserWithinListEntity{
				User: &twitterclient.User{
					TwitterId:   twitterClientUser.TwitterId,
					Name:        twitterClientUser.Name,
					ScreenName:  twitterClientUser.ScreenName,
					IsProtected: twitterClientUser.IsProtected,
					Followstate: twitterclient.FollowState(twitterClientUser.Followstate),
					MediaCount:  twitterClientUser.MediaCount,
					Muting:      twitterClientUser.Muting,
					Blocking:    twitterClientUser.Blocking,
				},
				MaybeListId: nil,
			},
		)
	}

	return userList, nil
}

////////////////////////////////////////////////////////////////////////////////

func (h *helper) parseLists(
	ctx context.Context,
	client *twitterclient.Client,
	db *sqlx.DB,
	lists []twitterclient.ListBase,
	rootDir string,
) ([]UserWithinListEntity, error) {
	ctx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)

	logger := log.WithField("caller", "heaphelper.ParseList")

	userList := make([]UserWithinListEntity, 0)
	userListMtx := sync.Mutex{}

	wg := sync.WaitGroup{}
	for _, twitterPostList := range lists {
		wg.Add(1)
		go func(lst twitterclient.ListBase) {
			defer wg.Done()

			res, err := h.syncListAndGetMembers(ctx, client, db, lst, rootDir)
			if err != nil {
				logger.Errorln("failed to sync list and get members:", err)
				cancel(err)
			}
			logger.Debugf("members of %s: %d", lst.Title(), len(res))

			userListMtx.Lock()
			userList = append(userList, res...)
			userListMtx.Unlock()
		}(twitterPostList)
	}
	wg.Wait()

	if err := context.Cause(ctx); err != nil {
		return nil, err
	}
	return userList, nil
}

func (h *helper) syncListAndGetMembers(
	ctx context.Context,
	client *twitterclient.Client,
	db *sqlx.DB,
	twitterList twitterclient.ListBase,
	dir string,
) ([]UserWithinListEntity, error) {
	logger := log.WithField("caller", "heaphelper.syncLstAndGetMembers")

	syncListToDbOut, err := h.syncListToDb(db, twitterList, dir)
	if err != nil {
		return nil, err
	}

	if err := h.syncListToStorage(
		&syncListToStorageInput{
			twitterList:   twitterList,
			listSmartPath: syncListToDbOut.listSmartPath,
		},
	); err != nil {
		logger.Errorln("failed to sync list to storage:", err)
		return nil, err
	}

	return h.getMembersFromList(
		ctx,
		&getMembersFromListInput{
			client:        client,
			twitterList:   twitterList,
			listSmartPath: syncListToDbOut.listSmartPath,
		},
	)
}

////////////////////////////////////////////////////////////////////////////////

type syncListToDbOutput struct {
	listSmartPath *smartpathdto.ListSmartPath
}

func (h *helper) syncListToDb(
	db *sqlx.DB,
	twitterList twitterclient.ListBase,
	dir string,
) (*syncListToDbOutput, error) {
	logger := log.WithField("caller", "heaphelper.syncListToDb")

	if v, ok := twitterList.(*twitterclient.List); ok {
		logger.Infof("process twitterclient.List with id: %d", v.Id)
		if err := h.listRecordCreateOrUpdate(db, v); err != nil {
			return nil, err
		}
	}

	entity, err := smartpathdto.NewListSmartPath(db, twitterList.GetId(), dir)
	if err != nil {
		logger.Errorln("failed to create new list smart path to db:", err)
		return nil, err
	}

	return &syncListToDbOutput{listSmartPath: entity}, nil
}

func (h *helper) listRecordCreateOrUpdate(db *sqlx.DB, list *twitterclient.List) error {
	logger := log.WithField("caller", "heaphelper.createOrUpdate")

	listdb, err := h.listRepo.GetById(db, list.Id)
	if err != nil {
		logger.Errorln("failed to get list from database:", err)
		return err
	}

	updated := &model.List{
		Id:      list.Id,
		Name:    list.Name,
		OwnerId: list.Creator.TwitterId,
	}
	if listdb == nil {
		logger.
			WithField("list", list.Title()).
			Infoln("list not found in database, creating new entry")
		return h.listRepo.Create(db, updated)
	}
	return h.listRepo.Update(db, updated)
}

////////////////////////////////////////////////////////////////////////////////

type syncListToStorageInput struct {
	twitterList   twitterclient.ListBase
	listSmartPath *smartpathdto.ListSmartPath
}

func (h *helper) syncListToStorage(
	input *syncListToStorageInput,
) error {
	logger := log.WithField("caller", "heaphelper.syncListToStorage")

	expectedTitle := utils.ToLegalWindowsFileName(input.twitterList.Title())
	if err := ExpectNameMustExistOnStorage(input.listSmartPath, expectedTitle); err != nil {
		logger.Errorln("failed to sync path to storage for list:", err)
		return err
	}

	return nil
}

////////////////////////////////////////////////////////////////////////////////

type getMembersFromListInput struct {
	client        *twitterclient.Client
	twitterList   twitterclient.ListBase
	listSmartPath *smartpathdto.ListSmartPath
}

func (h *helper) getMembersFromList(
	ctx context.Context,
	input *getMembersFromListInput,
) ([]UserWithinListEntity, error) {
	logger := log.WithField("caller", "heaphelper.getMembersFromList")

	// get all members
	members, err := input.twitterList.GetMembers(ctx, input.client)
	if err != nil || len(members) == 0 {
		logger.Warnln("failed to get members from list or no members found:", err)
		return nil, err
	}

	// bind lst entity to users for creating symlink
	packedUsers := make([]UserWithinListEntity, 0, len(members))
	eid := input.listSmartPath.Id()
	for _, user := range members {
		packedUsers = append(
			packedUsers,
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
				MaybeListId: &eid,
			},
		)
	}

	return packedUsers, nil
}
