package main

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestRefreshReplaysMultipartExactlyOnce(t *testing.T) {
	attempts, refreshes := 0, 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/cli/v1/auth/token" {
			refreshes++
			io.WriteString(w, `{"code":0,"data":{"access_token":"new","refresh_token":"rotated","expires_at":4102444800}}`)
			return
		}
		attempts++
		raw, _ := io.ReadAll(r.Body)
		if string(raw) != "multipart payload" {
			t.Error("body not replayed")
		}
		if attempts == 1 {
			w.WriteHeader(401)
			io.WriteString(w, `{"code":401,"msg":"expired"}`)
			return
		}
		if r.Header.Get("Authorization") != "Bearer new" {
			t.Error("wrong token")
		}
		io.WriteString(w, `{"code":0,"data":{"saved":true}}`)
	}))
	defer server.Close()
	t.Setenv("SCREEN_CLI_CONFIG_DIR", t.TempDir())
	c, e := newClient(server.URL, server.URL)
	if e != nil {
		t.Fatal(e)
	}
	c.config.Access = "old"
	c.config.Refresh = "refresh"
	c.config.ExpiresAt = time.Now().Add(time.Hour).Unix()
	if _, e = c.send("POST", "/upload", []byte("multipart payload"), "multipart/form-data", true); e != nil {
		t.Fatal(e)
	}
	if attempts != 2 || refreshes != 1 {
		t.Fatal("unexpected retry", attempts, refreshes)
	}
	info, e := os.Stat(c.configPath)
	if e != nil {
		t.Fatal(e)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0600 {
		t.Fatal("credentials not private")
	}
}
func TestServerSelectionNeverReusesAnotherIdentity(t *testing.T) {
	t.Setenv("SCREEN_CLI_CONFIG_DIR", t.TempDir())
	c, e := newClient("https://first.test", "https://first.test")
	if e != nil {
		t.Fatal(e)
	}
	c.config.Access = "private"
	c.config.Refresh = "refresh"
	c.save()
	other, e := newClient("https://second.test", "")
	if e != nil {
		t.Fatal(e)
	}
	if other.config.Access != "" || other.config.Refresh != "" {
		t.Fatal("identity crossed systems")
	}
}
func TestRejectsRemotePlainHTTPAndCredentialURL(t *testing.T) {
	for _, raw := range []string{"http://host.test", "https://user:password@host.test", "https://host.test/private", "https://host.test?token=x"} {
		if validServerURL(raw) == nil {
			t.Errorf("accepted %s", raw)
		}
	}
}
func TestConfigDirectoryWithoutHome(t *testing.T) {
	t.Setenv("SCREEN_CLI_CONFIG_DIR", "")
	t.Setenv("HOME", "")
	t.Setenv("XDG_CONFIG_HOME", "")
	dir, e := configDirectory()
	if e != nil || dir == "" {
		t.Fatal(dir, e)
	}
	if strings.Contains(dir, os.TempDir()) {
		defer os.RemoveAll(dir)
	}
}
func TestCommandsAndMutuallyExclusiveUpload(t *testing.T) {
	if _, e := run([]string{"--help"}); e != nil {
		t.Fatal(e)
	}
	args, server, web, dir, e := globalArguments([]string{"push", "--server=https://host.test", "--preview", "--project", "/tmp/a", "--web-url", "https://web.test"})
	if e != nil || server != "https://host.test" || web != "https://web.test" || dir != "/tmp/a" || len(args) != 2 {
		t.Fatal(args, e)
	}
	c := &client{}
	if _, e = c.push(t.TempDir(), "", "token", true); e == nil {
		t.Fatal("ambiguous push accepted")
	}
}
func TestCredentialsNeverEnterWorkspaceRecord(t *testing.T) {
	w := workspace{TaskID: "task", PageID: 1, Server: "https://host.test", Files: []codeFile{{Path: "index.html", Content: "<html></html>"}}}
	dir := t.TempDir()
	if _, e := writeWorkspace(dir, w, false); e != nil {
		t.Fatal(e)
	}
	raw, _ := os.ReadFile(workspacePath(dir))
	var object map[string]any
	json.Unmarshal(raw, &object)
	if _, ok := object["access_token"]; ok {
		t.Fatal("credentials leaked")
	}
	if filepath.Dir(workspacePath(dir)) != filepath.Join(dir, ".screen") {
		t.Fatal("wrong metadata path")
	}
}

func TestWaitCheckRejectsOldReadyRevision(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"code":0,"data":{"revisions":[{"id":7,"status":"ready","source_matches_current":false}]}}`)
	}))
	defer server.Close()
	t.Setenv("SCREEN_CLI_CONFIG_DIR", t.TempDir())
	c, err := newClient(server.URL, server.URL)
	if err != nil {
		t.Fatal(err)
	}
	c.config.ExpiresAt = time.Now().Add(time.Hour).Unix()
	_, err = c.waitCheck("/cli/v1/workspaces/t", 7)
	var result *apiError
	if !errors.As(err, &result) || result.Kind != "DRAFT_CHANGED" {
		t.Fatal("older ready check accepted", err)
	}
}
