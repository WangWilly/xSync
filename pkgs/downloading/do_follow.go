package downloading

import (
	"context"

	"github.com/WangWilly/xSync/pkgs/commonpkg/clients/twitterclient"
	log "github.com/sirupsen/logrus"
)

func (h *helper) doFollow(ctx context.Context, metas []*twitterclient.TitledUserList) {
	logger := log.
		WithField("caller", "downloading.doFollow")

	client := h.twitterClientManager.GetMasterClient()

	for _, meta := range metas {
		if meta == nil {
			logger.Warnln("doFollow: meta or meta.User is nil, skipping")
			continue
		}

		for _, user := range meta.Users {
			if user == nil {
				logger.Warnln("doFollow: user is nil, skipping")
				continue
			}

			if !(user.IsProtected && user.Followstate == twitterclient.FS_UNFOLLOW) {
				logger.Debugln("user is not protected or already followed, skipping")
				continue
			}

			if err := client.FollowUser(ctx, user.TwitterId); err != nil {
				logger.Warnln("failed to follow user:", err)
				continue
			}
			logger.Debugln("follow request has been sent")
		}
	}
}
