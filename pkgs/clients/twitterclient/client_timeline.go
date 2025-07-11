package twitterclient

import (
	"context"
	"fmt"
	"net/url"

	"github.com/tidwall/gjson"
)

////////////////////////////////////////////////////////////////////////////////
// Constants and Timeline Types
////////////////////////////////////////////////////////////////////////////////

const (
	timelineTweet = iota // Timeline item type for tweets
	timelineUser         // Timeline item type for users
)

////////////////////////////////////////////////////////////////////////////////
// Timeline API Interface
////////////////////////////////////////////////////////////////////////////////

// timelineApi extends the basic api interface with cursor functionality for paginated endpoints
type timelineApi interface {
	SetCursor(cursor string)
	Path() string
	QueryParam() url.Values
}

////////////////////////////////////////////////////////////////////////////////
// URL Construction Utilities
////////////////////////////////////////////////////////////////////////////////

// makeUrl constructs a complete URL from an API endpoint
func (c *Client) makeUrl(api timelineApi) string {
	u, _ := url.Parse(API_HOST)
	u = u.JoinPath(api.Path())
	u.RawQuery = api.QueryParam().Encode()
	return u.String()
}

////////////////////////////////////////////////////////////////////////////////
// JSON Response Parsing Utilities
////////////////////////////////////////////////////////////////////////////////

// getInstructions extracts instructions from Twitter API response
func (c *Client) getInstructions(resp []byte, path string) gjson.Result {
	inst := gjson.GetBytes(resp, path)
	if !inst.Exists() {
		panic(fmt.Sprintf("unable to get instructions: %s path: '%s'", resp, path))
	}
	return inst
}

// getEntries extracts entries from timeline instructions
func (c *Client) getEntries(instructions gjson.Result) gjson.Result {
	for _, inst := range instructions.Array() {
		if inst.Get("type").String() == "TimelineAddEntries" {
			return inst.Get("entries")
		}
	}
	return gjson.Result{}
}

// getModuleItems extracts module items from timeline instructions
func (c *Client) getModuleItems(instructions gjson.Result) gjson.Result {
	for _, inst := range instructions.Array() {
		if inst.Get("type").String() == "TimelineAddToModule" {
			return inst.Get("moduleItems")
		}
	}
	return gjson.Result{}
}

// getNextCursor extracts the next cursor for pagination
func (c *Client) getNextCursor(entries gjson.Result) string {
	array := entries.Array()
	// if len(array) == 2 {
	// 	return "" // no next page
	// }

	for i := len(array) - 1; i >= 0; i-- {
		if array[i].Get("content.entryType").String() == "TimelineTimelineCursor" &&
			array[i].Get("content.cursorType").String() == "Bottom" {
			return array[i].Get("content.value").String()
		}
	}

	panic(fmt.Sprintf("invalid entries: %s", entries.String()))
}

////////////////////////////////////////////////////////////////////////////////
// Timeline Entry Processing
////////////////////////////////////////////////////////////////////////////////

// getItemContentFromModuleItem extracts item content from module items
func (c *Client) getItemContentFromModuleItem(moduleItem gjson.Result) gjson.Result {
	res := moduleItem.Get("item.itemContent")
	if !res.Exists() {
		panic(fmt.Errorf("invalid ModuleItem: %s", moduleItem.String()))
	}
	return res
}

// getItemContentsFromEntry extracts item contents from timeline entries
func (c *Client) getItemContentsFromEntry(entry gjson.Result) []gjson.Result {
	content := entry.Get("content")
	entryType := content.Get("entryType").String()
	switch entryType {
	case "TimelineTimelineModule":
		return content.Get("items.#.item.itemContent").Array()
	case "TimelineTimelineItem":
		return []gjson.Result{content.Get("itemContent")}
	}

	panic(fmt.Sprintf("invalid entry: %s", entry.String()))
}

// getResults extracts results based on item type (tweet or user)
func (c *Client) getResults(itemContent gjson.Result, itemType int) gjson.Result {
	switch itemType {
	case timelineTweet:
		return itemContent.Get("tweet_results")
	case timelineUser:
		return itemContent.Get("user_results")
	}

	panic(fmt.Sprintf("invalid itemContent: %s", itemContent.String()))
}

////////////////////////////////////////////////////////////////////////////////
// Timeline API Operations
////////////////////////////////////////////////////////////////////////////////

// getTimelineResp makes HTTP request to timeline API endpoint
func (c *Client) getTimelineResp(ctx context.Context, api timelineApi) ([]byte, error) {
	url := c.makeUrl(api)
	resp, err := c.restyClient.R().SetContext(ctx).Get(url)
	if err != nil {
		return nil, err
	}
	return resp.Body(), nil
}

// getTimelineItemContents retrieves timeline item contents for a single page
// 获取时间线 API 并返回所有 itemContent 和 底部 cursor
func (c *Client) getTimelineItemContents(ctx context.Context, api timelineApi, instPath string) ([]gjson.Result, string, error) {
	resp, err := c.getTimelineResp(ctx, api)
	if err != nil {
		return nil, "", err
	}

	// is temporarily unavailable because it violates the Twitter Media Policy.
	// Protected User's following: Permission denied
	if string(resp) == "{\"data\":{\"user\":{}}}" {
		return nil, "", nil
	}
	instructions := c.getInstructions(resp, instPath)
	entries := c.getEntries(instructions)
	moduleItems := c.getModuleItems(instructions)
	if !entries.Exists() && !moduleItems.Exists() {
		panic(fmt.Sprintf("invalid instructions: %s", instructions.String()))
	}

	itemContents := make([]gjson.Result, 0)
	if entries.IsArray() {
		for _, entry := range entries.Array() {
			if entry.Get("content.entryType").String() != "TimelineTimelineCursor" {
				itemContents = append(itemContents, c.getItemContentsFromEntry(entry)...)
			}
		}
	}
	if moduleItems.IsArray() {
		for _, moduleItem := range moduleItems.Array() {
			itemContents = append(itemContents, c.getItemContentFromModuleItem(moduleItem))
		}
	}
	return itemContents, c.getNextCursor(entries), nil
}

// getTimelineItemContentsTillEnd retrieves all timeline item contents across multiple pages
func (c *Client) getTimelineItemContentsTillEnd(ctx context.Context, api timelineApi, instPath string) ([]gjson.Result, error) {
	res := make([]gjson.Result, 0)

	for {
		page, next, err := c.getTimelineItemContents(ctx, api, instPath)
		if err != nil {
			return nil, err
		}

		if len(page) == 0 {
			break // empty page
		}

		res = append(res, page...)
		api.SetCursor(next)
	}

	return res, nil
}
