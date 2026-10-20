package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestPersistentEndpointConfiguration(t *testing.T) {
	t.Setenv("SCREEN_CLI_CONFIG_DIR", t.TempDir())
	if _, e := run([]string{"config", "show"}); e != nil {
		t.Fatal(e)
	}
	if _, e := run([]string{"config", "set", "--endpoint", "https://first.example/", "--web-url", "https://web.example"}); e != nil {
		t.Fatal(e)
	}
	c, e := newClient("", "")
	if e != nil {
		t.Fatal(e)
	}
	c.config.Access = "secret-access"
	c.config.Refresh = "secret-refresh"
	c.config.Username = "alice"
	if e = c.save(); e != nil {
		t.Fatal(e)
	}
	result, e := run([]string{"config", "show"})
	if e != nil {
		t.Fatal(e)
	}
	raw, _ := json.Marshal(result)
	if strings.Contains(string(raw), "secret-") {
		t.Fatal("configuration disclosed credentials")
	}
	if _, e = run([]string{"config", "set", "--endpoint", "https://second.example"}); e != nil {
		t.Fatal(e)
	}
	c, e = newClient("", "")
	if e != nil || c.config.Access != "" || c.config.Refresh != "" || c.config.Username != "" || c.config.WebURL != "https://second.example" {
		t.Fatalf("identity crossed endpoints: %v", e)
	}
	if _, e = run([]string{"config", "set", "--endpoint", "http://unsafe.example"}); e == nil {
		t.Fatal("accepted insecure remote endpoint")
	}
	after, _ := os.ReadFile(c.configPath)
	if strings.Contains(string(after), "unsafe") {
		t.Fatal("invalid endpoint was saved")
	}
	if _, e = run([]string{"config", "set", "--endpoint", "https://one.example", "--server", "https://two.example"}); e == nil {
		t.Fatal("conflicting endpoint aliases accepted")
	}
}
func TestLogoutClearsRevokedIdentity(t *testing.T) {
	t.Setenv("SCREEN_CLI_CONFIG_DIR", t.TempDir())
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
		w.Write([]byte(`{"code":401,"msg":"revoked"}`))
	}))
	defer server.Close()
	c, _ := newClient(server.URL, "")
	c.config.Refresh = "revoked"
	c.config.Username = "alice"
	c.save()
	if _, e := c.logout(); e != nil {
		t.Fatal(e)
	}
	c, _ = newClient("", "")
	if c.config.Refresh != "" || c.config.Username != "" {
		t.Fatal("stale identity retained")
	}
}
