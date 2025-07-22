package twitterclient

import (
	"context"
	"fmt"
	"net/url"

	log "github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
)

type ListParams struct {
	VariablesForm string
	Features      string

	Id     uint64
	Count  int
	Cursor string

	Extras map[string]string
}

// getTimelineItemContentsTillEnd retrieves all timeline item contents across multiple pages
func (c *Client) getTimelineItemContentsTillEnd(ctx context.Context, path string, listParams ListParams, instPath string) ([]gjson.Result, error) {
	res := make([]gjson.Result, 0)
	for {
		page, next, err := c.getTimelineItemContents(ctx, path, listParams, instPath)
		if err != nil {
			return nil, err
		}

		if len(page) == 0 {
			break // empty page
		}

		res = append(res, page...)
		listParams.Cursor = next
	}

	return res, nil
}

// getTimelineItemContents retrieves timeline item contents for a single page
// 获取时间线 API 并返回所有 itemContent 和 底部 cursor
func (c *Client) getTimelineItemContents(ctx context.Context, path string, listParams ListParams, instPath string) ([]gjson.Result, string, error) {
	resp, err := c.getTimelineResp(ctx, path, listParams)
	if err != nil {
		return nil, "", err
	}

	// is temporarily unavailable because it violates the Twitter Media Policy.
	// Protected User's following: Permission denied
	if string(resp) == "{\"data\":{\"user\":{}}}" {
		return nil, "", nil
	}
	instructions := getInstructions(resp, instPath)
	entries := getEntries(instructions)
	moduleItems := getModuleItems(instructions)
	if !entries.Exists() && !moduleItems.Exists() {
		panic(fmt.Sprintf("invalid instructions: %s", instructions.String()))
	}

	itemContents := make([]gjson.Result, 0)
	if entries.IsArray() {
		for _, entry := range entries.Array() {
			if entry.Get("content.entryType").String() != "TimelineTimelineCursor" {
				itemContents = append(itemContents, getItemContentsFromEntry(entry)...)
			}
		}
	}
	if moduleItems.IsArray() {
		for _, moduleItem := range moduleItems.Array() {
			itemContents = append(itemContents, getItemContentFromModuleItem(moduleItem))
		}
	}
	return itemContents, getNextCursor(entries), nil
}

// getTimelineResp makes HTTP request to timeline API endpoint
func (c *Client) getTimelineResp(ctx context.Context, path string, listParams ListParams) ([]byte, error) {
	logger := log.WithFields(log.Fields{
		"caller": "Client.getTimelineResp",
		"client": c.screenName,
		"path":   path,
		"params": listParams,
	})

	u, _ := url.Parse(API_HOST)
	u = u.JoinPath(path)

	params := url.Values{}
	if listParams.VariablesForm != "" {
		params.Set("variables", fmt.Sprintf(listParams.VariablesForm, listParams.Id, listParams.Count, listParams.Cursor))
	}
	if listParams.Features != "" {
		params.Set("features", listParams.Features)
	}
	for k, v := range listParams.Extras {
		params.Set(k, v)
	}

	u.RawQuery = params.Encode()

	resp, err := c.restyClient.R().SetContext(ctx).Get(u.String())
	if err != nil {
		logger.WithError(err).Error("failed to get timeline response")
		return nil, err
	}
	return resp.Body(), nil
}

////////////////////////////////////////////////////////////////////////////////

// getInstructions extracts instructions from Twitter API response
func getInstructions(resp []byte, path string) gjson.Result {
	inst := gjson.GetBytes(resp, path)
	if !inst.Exists() {
		panic(fmt.Sprintf("unable to get instructions: %s path: '%s'", resp, path))
	}
	return inst
}
