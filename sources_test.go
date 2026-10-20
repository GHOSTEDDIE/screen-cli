package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSourcesCommandMethodsConfirmationAndTaskScope(t *testing.T) {
	requests := []string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.Path)
		if r.Method == "POST" {
			var input map[string]any
			if json.NewDecoder(r.Body).Decode(&input) != nil {
				t.Error("invalid input")
			}
			if r.URL.Path == "/cli/v1/sources" {
				if input["confirm"] != "reviewed-token" || len(input) != 1 {
					t.Error("confirmation input changed")
				}
			} else if input["task_id"] != "task" || input["name"] != "全新示例" {
				t.Error("task or connection changed")
			}
		}
		io.WriteString(w, `{"code":0,"data":{"saved":true}}`)
	}))
	defer server.Close()
	t.Setenv("SCREEN_CLI_CONFIG_DIR", t.TempDir())
	c, err := newClient(server.URL, server.URL)
	if err != nil {
		t.Fatal(err)
	}
	c.config.ExpiresAt = time.Now().Add(time.Hour).Unix()
	file := filepath.Join(t.TempDir(), "connection.json")
	os.WriteFile(file, []byte(`{"name":"全新示例","kind":"feishu_bitable","config":{"app_name":"demo-only"}}`), 0600)
	for _, args := range [][]string{{"create"}, {"create", "--confirm", "reviewed-token", "--file", file}, {"test"}, {"preview", "--file", file, "unexpected"}, {"unknown"}} {
		if _, err := c.sources(args); err == nil {
			t.Fatal("ambiguous source command accepted", args)
		}
	}
	if len(requests) != 0 {
		t.Fatal("invalid command made requests")
	}
	for _, args := range [][]string{{}, {"list"}, {"test", "--file", file, "--task-id", "task"}, {"preview", "--file", file, "--task-id", "task"}, {"create", "--confirm", "reviewed-token"}} {
		if _, err := c.sources(args); err != nil {
			t.Fatal(err)
		}
	}
	expected := []string{"GET /cli/v1/sources", "GET /cli/v1/sources", "POST /cli/v1/sources/test", "POST /cli/v1/sources/previews", "POST /cli/v1/sources"}
	if stringMustJSON(requests) != stringMustJSON(expected) {
		t.Fatal(requests)
	}
	os.WriteFile(file, []byte(`{"name":"全新示例","task_id":"another"}`), 0600)
	if _, err := c.sources([]string{"preview", "--file", file, "--task-id", "task"}); err == nil {
		t.Fatal("ambiguous task accepted")
	}
}

func stringMustJSON(value any) string { raw, _ := json.Marshal(value); return string(raw) }
