package metahelper

import (
	"context"
	"os"
	"path"

	"github.com/WangWilly/xSync/pkgs/commonpkg/clients/twitterclient"
	"github.com/WangWilly/xSync/pkgs/commonpkg/model"
	"github.com/WangWilly/xSync/pkgs/commonpkg/utils"
	log "github.com/sirupsen/logrus"
)

func (h *helper) SaveToStorage(ctx context.Context, rootDir string, metas []twitterclient.TitledUserList) error {
	for _, meta := range metas {
		switch meta.Type {
		case twitterclient.TITLED_TYPE_TWITTER_USER:
			if err := h.saveUserToStorage(ctx, rootDir, meta.BelongsTo); err != nil {
				return err
			}
		case twitterclient.TITLED_TYPE_TWITTER_LIST:
			if err := h.saveListToStorage(ctx, rootDir, &meta); err != nil {
				return err
			}
			for _, user := range meta.Users {
				if err := h.saveUserToStorage(ctx, rootDir, user); err != nil {
					return err
				}
			}
		case twitterclient.TITLED_TYPE_TWITTER_FOLLOWERS:
			for _, user := range meta.Users {
				if err := h.saveUserToStorage(ctx, rootDir, user); err != nil {
					return err
				}
			}
		default:
			continue
		}
	}

	return nil
}

func (h *helper) saveUserToStorage(ctx context.Context, rootDir string, user *twitterclient.User) error {
	logger := log.WithField("function", "SaveUserToStorage")

	if user == nil {
		logger.Warnln("skipping nil user")
		return nil
	}

	folderName := utils.ToLegalWindowsFileName(user.Title())
	if err := h.userEntityRepo.Upsert(
		ctx,
		h.db,
		&model.UserEntity{
			Uid:        user.TwitterId,
			Name:       user.Name,
			ParentDir:  rootDir,
			FolderName: folderName,
		},
	); err != nil {
		logger.Errorln("failed to create user entity:", err)
		return err
	}

	path := path.Join(rootDir, folderName)
	logger.WithField("path", path).Info("creating user directory")
	if err := os.MkdirAll(path, 0755); err != nil {
		logger.Errorln("failed to create user directory:", err)
		return err
	}

	if err := h.userEntityRepo.UpdateStorageSavedByTwitterId(
		ctx,
		h.db,
		user.TwitterId,
		true,
	); err != nil {
		logger.Errorln("failed to update user entity storage saved:", err)
		return err
	}

	return nil
}

func (h *helper) saveListToStorage(ctx context.Context, rootDir string, list *twitterclient.TitledUserList) error {
	logger := log.WithField("function", "SaveListToStorage")

	if list == nil {
		logger.Warnln("skipping nil list")
		return nil
	}

	folderName := utils.ToLegalWindowsFileName(list.TwitterName)
	record := &model.ListEntity{
		LstId:      int64(list.Id), // TODO:
		Name:       list.TwitterName,
		ParentDir:  rootDir,
		FolderName: folderName,
	}
	if err := h.listEntityRepo.Upsert(
		ctx,
		h.db,
		record,
	); err != nil {
		logger.Errorln("failed to create list entity:", err)
		return err
	}

	path := path.Join(rootDir, folderName)
	logger.WithField("path", path).Info("creating list directory")
	if err := os.MkdirAll(path, 0755); err != nil {
		logger.Errorln("failed to create list directory:", err)
		return err
	}

	if err := h.listEntityRepo.UpdateStorageSavedByTwitterId(ctx, h.db, list.Id, true); err != nil {
		logger.Errorln("failed to update list entity storage saved:", err)
		return err
	}

	// TODO:
	for _, user := range list.Users {
		h.linkRepo.Upsert(
			ctx,
			h.db,
			&model.UserLink{
				UserTwitterId:        user.TwitterId,
				Name:                 utils.ToLegalWindowsFileName(user.Title()),
				ListEntityIdBelongTo: record.Id.Int32,
				StorageSaved:         false,
			},
		)
	}

	return nil
}
