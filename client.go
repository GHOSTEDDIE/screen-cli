package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type credentials struct {
	Server    string `json:"server"`
	WebURL    string `json:"web_url"`
	Access    string `json:"access_token,omitempty"`
	Refresh   string `json:"refresh_token,omitempty"`
	ExpiresAt int64  `json:"expires_at,omitempty"`
	Username  string `json:"username,omitempty"`
}
type client struct {
	config     credentials
	configPath string
	http       *http.Client
}
type apiError struct {
	Kind    string `json:"error"`
	Message string `json:"message"`
	Status  int    `json:"status"`
}

func (e *apiError) Error() string { return e.Message }

type envelope struct {
	Code    int             `json:"code"`
	Error   string          `json:"error"`
	Message string          `json:"msg"`
	Data    json.RawMessage `json:"data"`
}

func configDirectory() (string, error) {
	if explicit := os.Getenv("SCREEN_CLI_CONFIG_DIR"); explicit != "" {
		return filepath.Abs(explicit)
	}
	base, err := os.UserConfigDir()
	if err == nil && base != "" {
		return filepath.Join(base, "screen-cli"), nil
	}
	// A process-local directory is deliberately not a persistent identity when
	// the OS cannot identify the user. Authentication must be explicit next time.
	return os.MkdirTemp("", "screen-cli-config-*")
}
func newClient(server, web string) (*client, error) {
	return loadClient(server, web, true)
}
func loadClient(server, web string, required bool) (*client, error) {
	dir, err := configDirectory()
	if err != nil {
		return nil, err
	}
	c := &client{configPath: filepath.Join(dir, "credentials.json"), http: &http.Client{Timeout: 60 * time.Second, CheckRedirect: func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse }}}
	raw, err := os.ReadFile(c.configPath)
	if err == nil {
		if err = json.Unmarshal(raw, &c.config); err != nil {
			return nil, errors.New("CLI 配置无效")
		}
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	if server != "" && strings.TrimRight(server, "/") != c.config.Server {
		c.config = credentials{Server: strings.TrimRight(server, "/"), WebURL: strings.TrimRight(web, "/")}
	}
	if web != "" {
		c.config.WebURL = strings.TrimRight(web, "/")
	}
	if c.config.Server == "" {
		if required {
			return nil, errors.New("请先运行 config set --endpoint https://系统地址")
		}
		return c, nil
	}
	if err = validServerURL(c.config.Server); err != nil {
		return nil, err
	}
	if c.config.WebURL == "" {
		c.config.WebURL = c.config.Server
	}
	if err = validServerURL(c.config.WebURL); err != nil {
		return nil, err
	}
	return c, nil
}
func validServerURL(raw string) error {
	u, e := url.Parse(raw)
	if e != nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.Path != "" && u.Path != "/" {
		return errors.New("系统地址必须为完整站点地址")
	}
	if u.Scheme == "https" {
		return nil
	}
	if u.Scheme == "http" && (u.Hostname() == "localhost" || u.Hostname() == "127.0.0.1" || u.Hostname() == "::1") {
		return nil
	}
	return errors.New("远程系统地址必须使用 HTTPS")
}
func atomicJSON(path string, value any) error {
	raw, e := json.MarshalIndent(value, "", "  ")
	if e != nil {
		return e
	}
	return atomicFile(path, append(raw, '\n'), 0600)
}
func atomicFile(path string, data []byte, mode os.FileMode) error {
	if e := os.MkdirAll(filepath.Dir(path), 0700); e != nil {
		return e
	}
	f, e := os.CreateTemp(filepath.Dir(path), ".screen-write-*")
	if e != nil {
		return e
	}
	name := f.Name()
	defer os.Remove(name)
	if e = f.Chmod(mode); e == nil {
		_, e = f.Write(data)
	}
	if e == nil {
		e = f.Sync()
	}
	closeError := f.Close()
	if e == nil {
		e = closeError
	}
	if e != nil {
		return e
	}
	return os.Rename(name, path)
}
func (c *client) save() error {
	if e := os.MkdirAll(filepath.Dir(c.configPath), 0700); e != nil {
		return e
	}
	if e := protectConfigDirectory(filepath.Dir(c.configPath)); e != nil {
		return e
	}
	return atomicJSON(c.configPath, c.config)
}
func (c *client) refresh() error {
	if c.config.Refresh == "" {
		return &apiError{Kind: "AUTH_REQUIRED", Message: "请先运行 auth login", Status: 401}
	}
	data, e := c.request("POST", "/cli/v1/auth/token", map[string]any{"grant_type": "refresh_token", "refresh_token": c.config.Refresh}, false)
	if e != nil {
		return e
	}
	var tokens credentials
	if e = json.Unmarshal(data, &tokens); e != nil {
		return e
	}
	if tokens.Access == "" || tokens.Refresh == "" || tokens.ExpiresAt <= time.Now().Unix() {
		return errors.New("授权结果无效")
	}
	c.config.Access = tokens.Access
	c.config.Refresh = tokens.Refresh
	c.config.ExpiresAt = tokens.ExpiresAt
	return c.save()
}
func (c *client) request(method, path string, input any, auth bool) (json.RawMessage, error) {
	raw, e := json.Marshal(input)
	if e != nil {
		return nil, e
	}
	if input == nil {
		raw = nil
	}
	return c.send(method, path, raw, "application/json", auth)
}
func (c *client) send(method, path string, body []byte, contentType string, auth bool) (json.RawMessage, error) {
	if auth && c.config.ExpiresAt <= time.Now().Add(30*time.Second).Unix() {
		if e := c.refresh(); e != nil {
			return nil, e
		}
	}
	for attempt := 0; attempt < 2; attempt++ {
		req, e := http.NewRequest(method, c.config.Server+path, bytes.NewReader(body))
		if e != nil {
			return nil, e
		}
		req.Header.Set("Content-Type", contentType)
		if auth {
			req.Header.Set("Authorization", "Bearer "+c.config.Access)
		}
		response, e := c.http.Do(req)
		if e != nil {
			return nil, errors.New("系统连接失败，请核对地址和网络")
		}
		raw, e := io.ReadAll(io.LimitReader(response.Body, 32<<20))
		response.Body.Close()
		if e != nil {
			return nil, e
		}
		if response.StatusCode == 401 && auth && attempt == 0 {
			if e = c.refresh(); e != nil {
				return nil, e
			}
			continue
		}
		var result envelope
		if e = json.Unmarshal(raw, &result); e != nil {
			return nil, &apiError{Kind: "INVALID_RESPONSE", Message: "系统返回了无效结果", Status: response.StatusCode}
		}
		if response.StatusCode >= 300 || result.Code != 0 {
			kind := result.Error
			if kind == "" {
				kind = "REQUEST_REJECTED"
			}
			return nil, &apiError{Kind: kind, Message: result.Message, Status: response.StatusCode}
		}
		return result.Data, nil
	}
	return nil, fmt.Errorf("登录刷新后请求仍未通过")
}
