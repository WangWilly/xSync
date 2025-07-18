package arghelper

import (
	"context"

	"github.com/WangWilly/xSync/pkgs/commonpkg/clients/twitterclient"
)

type helper struct {
	client                        *twitterclient.Client
	userTwitterIdsArg             UserTwitterIdsArg
	userTwitterScreenNamesArg     UserTwitterScreenNamesArg
	twitterListIdsArg             TwitterListIdsArg
	userTwitterIdsForFollowersArg UserTwitterIdsArg
}

func New(
	client *twitterclient.Client,
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

func (h *helper) GetTitledUserLists(ctx context.Context) ([]twitterclient.TitledUserList, error) {
	titledUserLists := make([]twitterclient.TitledUserList, 0)

	if h.userTwitterIdsArg != nil {
		for _, userId := range h.userTwitterIdsArg {
			titledUserList, err := twitterclient.NewTulByTwitterUserId(ctx, h.client, userId)
			if err != nil {
				return nil, err
			}
			titledUserLists = append(titledUserLists, *titledUserList)
		}
	}

	if h.userTwitterScreenNamesArg != nil {
		for _, screenName := range h.userTwitterScreenNamesArg {
			titledUserList, err := twitterclient.NewTulByTwitterUserName(ctx, h.client, screenName)
			if err != nil {
				return nil, err
			}
			titledUserLists = append(titledUserLists, *titledUserList)
		}
	}

	if h.twitterListIdsArg != nil {
		for _, listId := range h.twitterListIdsArg {
			titledUserList, err := twitterclient.NewTulByTwitterListId(ctx, h.client, listId)
			if err != nil {
				return nil, err
			}
			titledUserLists = append(titledUserLists, *titledUserList)
		}
	}

	if h.userTwitterIdsForFollowersArg != nil {
		for _, userId := range h.userTwitterIdsForFollowersArg {
			titledUserList, err := twitterclient.NewTulByTwitterFollowingUserId(ctx, h.client, userId)
			if err != nil {
				return nil, err
			}
			titledUserLists = append(titledUserLists, *titledUserList)
		}
	}

	return titledUserLists, nil
}
