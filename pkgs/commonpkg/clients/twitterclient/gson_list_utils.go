package twitterclient

import (
	"fmt"

	"github.com/tidwall/gjson"
)

// getItemContentFromModuleItem extracts item content from module items
func getItemContentFromModuleItem(moduleItem gjson.Result) gjson.Result {
	res := moduleItem.Get("item.itemContent")
	if !res.Exists() {
		panic(fmt.Errorf("invalid ModuleItem: %s", moduleItem.String()))
	}
	return res
}

// getItemContentsFromEntry extracts item contents from timeline entries
func getItemContentsFromEntry(entry gjson.Result) []gjson.Result {
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

// getModuleItems extracts module items from timeline instructions
func getModuleItems(instructions gjson.Result) gjson.Result {
	for _, inst := range instructions.Array() {
		if inst.Get("type").String() == "TimelineAddToModule" {
			return inst.Get("moduleItems")
		}
	}
	return gjson.Result{}
}

// getEntries extracts entries from timeline instructions
func getEntries(instructions gjson.Result) gjson.Result {
	for _, inst := range instructions.Array() {
		if inst.Get("type").String() == "TimelineAddEntries" {
			return inst.Get("entries")
		}
	}
	return gjson.Result{}
}

////////////////////////////////////////////////////////////////////////////////

// getNextCursor extracts the next cursor for pagination
func getNextCursor(entries gjson.Result) string {
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
