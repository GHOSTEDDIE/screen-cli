package main

import (
	"encoding/json"
	"errors"
	"flag"
)

func (c *client) sources(args []string) (any, error) {
	if len(args) == 0 || len(args) == 1 && args[0] == "list" {
		return c.request("GET", "/cli/v1/sources", nil, true)
	}
	if args[0] == "create" {
		confirm := ""
		_, err := commandFlags("sources create", args[1:], func(f *flag.FlagSet) { f.StringVar(&confirm, "confirm", "", "") })
		if err != nil {
			return nil, err
		}
		if confirm == "" {
			return nil, errors.New("sources create 需要 --confirm 接入预览凭据")
		}
		return c.request("POST", "/cli/v1/sources", map[string]string{"confirm": confirm}, true)
	}
	if args[0] != "test" && args[0] != "preview" {
		return nil, errors.New("sources 支持 list、test --file、preview --file、create --confirm")
	}
	file, taskID := "", ""
	_, err := commandFlags("sources "+args[0], args[1:], func(f *flag.FlagSet) {
		f.StringVar(&file, "file", "", "")
		f.StringVar(&taskID, "task-id", "", "")
	})
	if err != nil {
		return nil, err
	}
	if file == "" {
		return nil, errors.New("sources " + args[0] + " 需要 --file 连接.json|-")
	}
	raw, err := readJSONInput(file)
	if err != nil {
		return nil, err
	}
	var input map[string]json.RawMessage
	if err := json.Unmarshal(raw, &input); err != nil || input == nil {
		return nil, errors.New("连接输入必须为 JSON 对象")
	}
	if taskID != "" {
		if _, exists := input["task_id"]; exists {
			return nil, errors.New("task_id 只能通过文件或 --task-id 其中一个指定")
		}
		input["task_id"], _ = json.Marshal(taskID)
	}
	path := "/cli/v1/sources/test"
	if args[0] == "preview" {
		path = "/cli/v1/sources/previews"
	}
	return c.request("POST", path, input, true)
}
