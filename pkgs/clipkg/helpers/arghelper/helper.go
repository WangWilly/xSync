package arghelper

import (
	"context"

	"github.com/WangWilly/xSync/pkgs/commonpkg/clients/twitterclient"

	log "github.com/sirupsen/logrus"
)

type helper struct {
	client TwitterClient

	userTwitterIdsArg             UserTwitterIdsArg
	userTwitterScreenNamesArg     UserTwitterScreenNamesArg
	twitterListIdsArg             TwitterListIdsArg
	userTwitterIdsForFollowersArg UserTwitterIdsArg
}

func New(
	client TwitterClient,
	userTwitterIdsArg UserTwitterIdsArg,
	userTwitterScreenNamesArg UserTwitterScreenNamesArg,
	twitterListIdsArg TwitterListIdsArg,
	userTwitterIdsForFollowersArg UserTwitterIdsArg,
) *helper {
	return &helper{
		client:                        client,
		userTwitterIdsArg:             userTwitterIdsArg,
		userTwitterScreenNamesArg:     userTwitterScreenNamesArg,
		twitterListIdsArg:             twitterListIdsArg,
		userTwitterIdsForFollowersArg: userTwitterIdsForFollowersArg,
	}
}

////////////////////////////////////////////////////////////////////////////////

func (h *helper) GetTitledUserLists(
	ctx context.Context,
) []twitterclient.TitledUserList {
	logger := log.WithField("function", "GetTitledUserLists")

	titledUserLists := make([]twitterclient.TitledUserList, 0)

	if h.userTwitterIdsArg != nil {
		for _, userId := range h.userTwitterIdsArg {
			user, err := h.client.GetUserById(ctx, userId)
			if err != nil {
				logger.Errorln("failed to get user by id:", err)
				continue
			}
			titledUserLists = append(
				titledUserLists,
				*twitterclient.NewTitledUserListByUser(user),
			)
		}
	}

	if h.userTwitterScreenNamesArg != nil {
		for _, screenName := range h.userTwitterScreenNamesArg {
			user, err := h.client.GetUserByScreenName(ctx, screenName)
			if err != nil {
				logger.Errorln("failed to get user by screen name:", err)
				continue
			}
			titledUserLists = append(
				titledUserLists,
				*twitterclient.NewTitledUserListByUser(user),
			)
		}
	}

	if h.twitterListIdsArg != nil {
		for _, listId := range h.twitterListIdsArg {
			gjson, err := h.client.GetRawListByteById(ctx, listId)
			if err != nil {
				logger.Errorln("failed to get raw list byte by id:", err)
				continue
			}
			members, err := h.client.GetAllListMembers(ctx, listId)
			if err != nil {
				logger.Errorln("failed to get all list members:", err)
				continue
			}
			titledUserList, err := twitterclient.NewTulByRawListByteAndMembers(gjson, members)
			if err != nil {
				logger.Errorln("failed to create TitledUserList by listId:", err)
				continue
			}
			titledUserLists = append(titledUserLists, *titledUserList)
		}
	}

	if h.userTwitterIdsForFollowersArg != nil {
		for _, userId := range h.userTwitterIdsForFollowersArg {
			user, err := h.client.GetUserById(ctx, userId)
			if err != nil {
				logger.Errorln("failed to get user by id for followers:", err)
				continue
			}
			followers, err := h.client.GetAllFollowingMembers(ctx, userId)
			if err != nil {
				logger.Errorln("failed to get all following members:", err)
				continue
			}
			titledUserLists = append(
				titledUserLists,
				*twitterclient.NewTulByUserAndFollowers(user, followers),
			)
		}
	}

	return titledUserLists
}
