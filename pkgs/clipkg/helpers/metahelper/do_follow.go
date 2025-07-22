package metahelper

import (
	"context"

	"github.com/WangWilly/xSync/pkgs/commonpkg/clients/twitterclient"
	log "github.com/sirupsen/logrus"
)

func (h *helper) DoFollow(ctx context.Context, metas []twitterclient.TitledUserList) {
	logger := log.
		WithField("caller", "downloading.doFollow")

	client := h.twitterClientManager.GetMasterClient()

	for _, meta := range metas {
		for _, user := range meta.Users {
			if user == nil {
				logger.Warnln("doFollow: user is nil, skipping")
				continue
			}

			if !(user.IsProtected && user.Followstate == twitterclient.FS_UNFOLLOW) {
				logger.Debugln("user is not protected or already followed, skipping")
				continue
			}

			if err := client.DoFollowByUserId(ctx, user.TwitterId); err != nil {
				logger.Warnln("failed to follow user:", err)
				continue
			}
			logger.Debugln("follow request has been sent")
		}
	}
}
