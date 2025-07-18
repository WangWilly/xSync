package downloading

import (
	"context"

	"github.com/WangWilly/xSync/pkgs/commonpkg/clients/twitterclient"
)

func (h *helper) SaveToStorage(ctx context.Context, rootDir string, metas []*twitterclient.TitledUserList) error
