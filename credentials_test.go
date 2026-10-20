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

func TestCredentialsTransportConfirmationAndSecretStdin(t *testing.T) {
	var requests []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.Path)
		if r.Method == "POST" {
			var input map[string]any
			json.NewDecoder(r.Body).Decode(&input)
			if r.URL.Path == "/cli/v1/credentials" {
				if input["confirm"] != "reviewed" || len(input) != 1 {
					t.Error("confirmation changed")
				}
			} else if input["token"] != "private-fixture" || input["kind"] != "http_bearer" {
				t.Error("stdin changed")
			}
		}
		io.WriteString(w, `{"code":0,"data":{"saved":false}}`)
	}))
	defer server.Close()
	t.Setenv("SCREEN_CLI_CONFIG_DIR", t.TempDir())
	c, err := newClient(server.URL, server.URL)
	if err != nil {
		t.Fatal(err)
	}
	c.config.ExpiresAt = time.Now().Add(time.Hour).Unix()
	file := filepath.Join(t.TempDir(), "credential.json")
	os.WriteFile(file, []byte(`{"name":"fixture","kind":"http_bearer","allowed_host":"data.example","token":"private-fixture"}`), 0600)
	for _, args := range [][]string{{"create"}, {"create", "--confirm", "reviewed", "--file", file}, {"preview"}, {"list", "extra"}, {"preview", "--file", file, "--token", "unsafe"}, {"delete"}} {
		if _, err := c.credentials(args); err == nil {
			t.Fatal("invalid command accepted", args)
		}
	}
	if len(requests) != 0 {
		t.Fatal("invalid command made requests")
	}
	for _, args := range [][]string{{"list"}, {"preview", "--file", file}, {"create", "--confirm", "reviewed"}} {
		if _, err := c.credentials(args); err != nil {
			t.Fatal(err)
		}
	}
	stdin, err := os.Open(file)
	if err != nil {
		t.Fatal(err)
	}
	defer stdin.Close()
	previous := os.Stdin
	os.Stdin = stdin
	defer func() { os.Stdin = previous }()
	if _, err := c.credentials([]string{"preview", "--file", "-"}); err != nil {
		t.Fatal(err)
	}
	want := []string{"GET /cli/v1/credentials", "POST /cli/v1/credentials/previews", "POST /cli/v1/credentials", "POST /cli/v1/credentials/previews"}
	if stringMustJSON(requests) != stringMustJSON(want) {
		t.Fatal(requests)
	}
}
