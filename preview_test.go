package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

func TestTaskPreviewURLKeepsTaskInOneQueryParameter(t *testing.T) {
	for _, taskID := range []string{"533a241c-bdd0-42e3-9a7e-0544bc6e838e", "task?x=1&other=2#fragment/任务"} {
		link, err := url.Parse(taskPreviewURL("https://screen.example/", taskID))
		if err != nil {
			t.Fatal(err)
		}
		if link.Scheme != "https" || link.Host != "screen.example" || link.Path != "/setting" || link.Fragment != "" || len(link.Query()) != 1 || link.Query().Get("task_id") != taskID {
			t.Fatalf("preview URL changed origin or task: %s", link)
		}
	}
}

func TestPreviewCommandReturnsWorkbenchURL(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/cli/v1/workspaces/cli-task/preview" || r.Header.Get("Authorization") != "Bearer test-access" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		requests++
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"code":0,"data":{"id":7,"page_id":9,"status":"dev_ready"}}`)
	}))
	defer server.Close()
	t.Setenv("SCREEN_CLI_CONFIG_DIR", t.TempDir())
	c, err := newClient(server.URL, "https://screen.example")
	if err != nil {
		t.Fatal(err)
	}
	c.config.Access = "test-access"
	c.config.ExpiresAt = time.Now().Add(time.Hour).Unix()
	if err := c.save(); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := atomicJSON(workspacePath(dir), workspace{TaskID: "cli-task", PageID: 9, Server: server.URL}); err != nil {
		t.Fatal(err)
	}
	result, err := run([]string{"--project", dir, "preview", "--no-open"})
	if err != nil {
		t.Fatal(err)
	}
	data := result.(map[string]any)
	if requests != 1 || data["web_url"] != "https://screen.example/setting?task_id=cli-task" || data["opened"] != false {
		t.Fatalf("preview returned the wrong workbench: %#v", data)
	}
	var revision struct {
		ID int `json:"id"`
	}
	if err := json.Unmarshal(data["preview"].(json.RawMessage), &revision); err != nil || revision.ID != 7 {
		t.Fatal("preview revision was lost", err)
	}
}
