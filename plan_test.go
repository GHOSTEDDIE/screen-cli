package main

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestPlanCommandsRequireConfirmationAndTransportExactPlan(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Path != "/cli/v1/workspaces/task/plan" {
			t.Error("wrong workspace path", r.URL.Path)
		}
		if r.Method == "POST" {
			var input struct {
				Confirm bool              `json:"confirm"`
				Plan    map[string]string `json:"plan"`
			}
			if json.NewDecoder(r.Body).Decode(&input) != nil || !input.Confirm || input.Plan["goal"] != "保留正确订单" {
				t.Error("plan changed during transport")
			}
		} else if r.Method != "GET" {
			t.Error("incorrect HTTP method", r.Method)
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
	file := filepath.Join(t.TempDir(), "plan.json")
	if err := os.WriteFile(file, []byte(`{"goal":"保留正确订单"}`), 0600); err != nil {
		t.Fatal(err)
	}
	base := "/cli/v1/workspaces/task"
	for _, args := range [][]string{{}, {"get", "--confirm"}, {"save", "--file", file}, {"save", "--file", file, "--confirm=false"}, {"save", "--confirm"}, {"save", "--file", file, "--confirm", "unexpected"}} {
		if _, err := c.dashboardPlan(base, args); err == nil {
			t.Fatal("ambiguous/unconfirmed command accepted", args)
		}
	}
	if calls != 0 {
		t.Fatal("invalid commands made requests")
	}
	if _, err := c.dashboardPlan(base, []string{"get"}); err != nil {
		t.Fatal(err)
	}
	if _, err := c.dashboardPlan(base, []string{"save", "--file", file, "--confirm"}); err != nil {
		t.Fatal(err)
	}
	if calls != 2 {
		t.Fatal("unexpected request count", calls)
	}
}

func TestWaitCheckRejectsChangedPlanEvenIfSourceUnchanged(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"code":0,"data":{"revisions":[{"id":8,"status":"ready","source_matches_current":true,"draft_matches_current":false}]}}`)
	}))
	defer server.Close()
	t.Setenv("SCREEN_CLI_CONFIG_DIR", t.TempDir())
	c, err := newClient(server.URL, server.URL)
	if err != nil {
		t.Fatal(err)
	}
	c.config.ExpiresAt = time.Now().Add(time.Hour).Unix()
	_, err = c.waitCheck("/cli/v1/workspaces/task", 8)
	var failure *apiError
	if !errors.As(err, &failure) || failure.Kind != "DRAFT_CHANGED" {
		t.Fatal("old plan falsely accepted", err)
	}
}
