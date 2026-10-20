package main

import (
	"net/url"
	"strings"
)

// The settings workbench owns page preview and publication confirmation.
// The ordinary /ai-agent page is a separate conversation entry.
func taskPreviewURL(webURL, taskID string) string {
	return strings.TrimRight(webURL, "/") + "/setting?task_id=" + url.QueryEscape(taskID)
}
