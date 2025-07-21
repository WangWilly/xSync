package metahelper

import (
	"context"

	"github.com/WangWilly/xSync/pkgs/commonpkg/clients/twitterclient"
	"github.com/WangWilly/xSync/pkgs/downloading/dtos/smartpathdto"
	log "github.com/sirupsen/logrus"
)

func (h *helper) ToUserSmartPaths(
	ctx context.Context,
	metas []twitterclient.TitledUserList,
) []*smartpathdto.UserSmartPath {
	logger := log.WithField("caller", "ToUserSmartPaths")

	res := make([]*smartpathdto.UserSmartPath, 0)
	for _, meta := range metas {
		for _, user := range meta.Users {
			if user == nil {
				logger.Warnln("skipping nil user in meta", meta.Type)
				continue
			}

			if user.Blocking || user.Muting {
				logger.Infoln("user is ignored, skipping")
				continue
			}

			userEntity, err := h.userRepo.GetEntityByTwitterId(ctx, h.db, user.TwitterId)
			if err != nil {
				logger.Warnln("failed to get user entity by Twitter ID", user.TwitterId, ":", err)
				continue
			}
			if userEntity == nil {
				logger.Warnln("user entity not found for Twitter ID", user.TwitterId)
				continue
			}

			res = append(
				res,
				smartpathdto.New(
					userEntity,
					calcUserDepth(int(userEntity.MediaCount.Int32), user.MediaCount),
				),
			)
		}
	}

	return res
}

////////////////////////////////////////////////////////////////////////////////

func calcUserDepth(exist int, total int) int {
	if exist >= total {
		return 1
	}

	miss := total - exist
	depth := miss / twitterclient.AvgTweetsPerPage
	if miss%twitterclient.AvgTweetsPerPage != 0 {
		depth++
	}
	if exist == 0 {
		depth++ //对于新用户，需要多获取一个空页
	}
	return depth
}
