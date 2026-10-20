package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/url"
	"os/exec"
	"runtime"
	"sync/atomic"
	"time"
)

func randomString() string {
	b := make([]byte, 32)
	if _, e := rand.Read(b); e != nil {
		panic(e)
	}
	return base64.RawURLEncoding.EncodeToString(b)
}
func challenge(verifier string) string {
	h := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(h[:])
}
func openBrowser(raw string) error {
	var command *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		command = exec.Command("open", raw)
	case "windows":
		command = exec.Command("rundll32", "url.dll,FileProtocolHandler", raw)
	default:
		command = exec.Command("xdg-open", raw)
	}
	return command.Run()
}
func (c *client) login(noBrowser bool) (any, error) { return c.loginAs(noBrowser, "") }
func (c *client) loginAs(noBrowser bool, account string) (any, error) {
	listener, e := net.Listen("tcp", "127.0.0.1:0")
	if e != nil {
		return nil, e
	}
	defer listener.Close()
	redirect := "http://" + listener.Addr().String() + "/callback"
	state, verifier := randomString(), randomString()
	data, e := c.request("POST", "/cli/v1/auth/requests", map[string]string{"redirect_uri": redirect, "state": state, "code_challenge": challenge(verifier), "code_challenge_method": "S256", "expected_username": account}, false)
	if e != nil {
		return nil, e
	}
	var authorization struct {
		RequestID string `json:"request_id"`
	}
	if e = json.Unmarshal(data, &authorization); e != nil {
		return nil, e
	}
	codeChannel := make(chan string, 1)
	mux := http.NewServeMux()
	var consumed atomic.Bool
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.URL.Query().Get("state") != state || r.URL.Query().Get("code") == "" {
			http.Error(w, "授权无效", 400)
			return
		}
		if !consumed.CompareAndSwap(false, true) {
			http.Error(w, "授权已完成", 409)
			return
		}
		select {
		case codeChannel <- r.URL.Query().Get("code"):
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.Header().Set("Cache-Control", "no-store")
			_, _ = w.Write([]byte("本机大屏开发已授权，可以关闭此页面。"))
		default:
			http.Error(w, "授权已完成", 409)
		}
	})
	server := &http.Server{Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	defer server.Close()
	go server.Serve(listener)
	link := c.config.WebURL + "/cli/authorize?request_id=" + url.QueryEscape(authorization.RequestID)
	if noBrowser {
		writeOutput(map[string]any{"authorization_url": link, "waiting": true})
	} else if e = openBrowser(link); e != nil {
		writeOutput(map[string]any{"authorization_url": link, "waiting": true})
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	var code string
	select {
	case code = <-codeChannel:
	case <-ctx.Done():
		return nil, errors.New("授权等待超时，请重新登录")
	}
	data, e = c.request("POST", "/cli/v1/auth/token", map[string]string{"grant_type": "authorization_code", "code": code, "redirect_uri": redirect, "code_verifier": verifier}, false)
	if e != nil {
		return nil, e
	}
	var tokens credentials
	if e = json.Unmarshal(data, &tokens); e != nil {
		return nil, e
	}
	if tokens.Access == "" || tokens.Refresh == "" || tokens.ExpiresAt <= time.Now().Unix() {
		return nil, errors.New("授权结果无效")
	}
	c.config.Access = tokens.Access
	c.config.Refresh = tokens.Refresh
	c.config.ExpiresAt = tokens.ExpiresAt
	identity, e := c.request("GET", "/cli/v1/auth/status", nil, true)
	if e != nil {
		return nil, e
	}
	var status struct {
		Username      string `json:"username"`
		Authenticated bool   `json:"authenticated"`
	}
	if json.Unmarshal(identity, &status) != nil || !status.Authenticated || status.Username == "" || account != "" && status.Username != account {
		return nil, errors.New("登录账号与预期不一致，请重新授权")
	}
	c.config.Username = status.Username
	if e = c.save(); e != nil {
		return nil, e
	}
	return identity, nil
}
