package downloading

import (
	"context"

	"github.com/WangWilly/xSync/pkgs/commonpkg/clients/twitterclient"
	"github.com/WangWilly/xSync/pkgs/commonpkg/model"
	log "github.com/sirupsen/logrus"
)

func (h *helper) saveToDb(ctx context.Context, metas []*twitterclient.TitledUserList) error {
	logger := log.WithField("function", "SaveToDb")
	logger.Infoln("saving entities to database")

	for _, meta := range metas {
		if meta == nil {
			logger.Warnln("skipping nil meta")
			continue
		}

		switch meta.Type {
		case twitterclient.TITLED_TYPE_TWITTER_USER:
			if err := h.SaveUserToDb(ctx, meta.BelongsTo); err != nil {
				return err
			}
		case twitterclient.TITLED_TYPE_TWITTER_LIST:
			if err := h.SaveListToDb(ctx, meta); err != nil {
				return err
			}
			for _, user := range meta.Users {
				if err := h.SaveUserToDb(ctx, user); err != nil {
					return err
				}
			}
		case twitterclient.TITLED_TYPE_TWITTER_FOLLOWERS:
			for _, user := range meta.Users {
				if err := h.SaveUserToDb(ctx, user); err != nil {
					return err
				}
			}
		default:
			logger.Warnf("unknown meta type: %s", meta.Type)
			continue
		}
	}

	logger.Infoln("successfully saved all entities to database")
	return nil
}

func (h *helper) SaveUserToDb(ctx context.Context, user *twitterclient.User) error {
	logger := log.WithField("function", "SaveUserToDb")

	if user == nil {
		logger.Warnln("skipping nil user")
		return nil
	}

	if err := h.userRepo.CreatePreviousName(h.db, user.TwitterId, user.Name, user.ScreenName); err != nil {
		logger.Errorln("failed to create previous name:", err)
		return err
	}

	if err := h.userRepo.Upsert(
		h.db,
		&model.User{
			Id:           user.TwitterId,
			Name:         user.Name,
			ScreenName:   user.ScreenName,
			IsProtected:  user.IsProtected,
			FriendsCount: user.FriendsCount,
		},
	); err != nil {
		logger.Errorln("failed to upsert user:", err)
		return err
	}

	return nil
}

func (h *helper) SaveListToDb(ctx context.Context, list *twitterclient.TitledUserList) error {
	logger := log.WithField("function", "SaveListToDb")

	if list == nil {
		logger.Warnln("skipping nil list")
		return nil
	}

	if err := h.listRepo.Upsert(h.db, &model.List{
		Id:      list.Id,
		Name:    list.TwitterName,
		OwnerId: list.BelongsTo.TwitterId,
	}); err != nil {
		logger.Errorln("failed to upsert list:", err)
		return err
	}

	return nil
}
