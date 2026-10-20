package main

import (
	"encoding/json"
	"errors"
)

// Endpoint and credentials share one atomic private file so an endpoint switch
// cannot leave a token belonging to the previous server behind.
func configureClient(args []string, server, web string) (any, error) {
	if len(args) != 1 || args[0] != "set" && args[0] != "show" {
		return nil, errors.New("用法：config set --endpoint 地址 [--web-url 网页地址] 或 config show")
	}
	if args[0] == "show" && (server != "" || web != "") {
		return nil, errors.New("config show 不接受地址参数")
	}
	if args[0] == "set" && server == "" && web == "" {
		return nil, errors.New("请指定 --endpoint 或 --web-url")
	}
	c, e := loadClient(server, web, args[0] == "set")
	if e != nil {
		return nil, e
	}
	if args[0] == "set" {
		if e = c.save(); e != nil {
			return nil, e
		}
	}
	return map[string]any{"endpoint": c.config.Server, "web_url": c.config.WebURL, "username": c.config.Username, "configured": c.config.Server != "", "has_credentials": c.config.Refresh != ""}, nil
}
func (c *client) logout() (json.RawMessage, error) {
	if c.config.Access != "" || c.config.Refresh != "" {
		_, e := c.request("POST", "/cli/v1/auth/logout", map[string]any{}, true)
		var api *apiError
		// A revoked session is already logged out remotely. Connection failures must
		// remain visible so a still-live remote grant is not silently abandoned.
		if e != nil && (!errors.As(e, &api) || api.Status != 401) {
			return nil, e
		}
	}
	c.config.Access = ""
	c.config.Refresh = ""
	c.config.ExpiresAt = 0
	c.config.Username = ""
	if e := c.save(); e != nil {
		return nil, e
	}
	return json.RawMessage(`{"authenticated":false}`), nil
}
