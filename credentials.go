package main

import (
	"encoding/json"
	"errors"
	"flag"
)

func (c *client) credentials(args []string) (any, error) {
	if len(args) == 0 || len(args) == 1 && args[0] == "list" {
		return c.request("GET", "/cli/v1/credentials", nil, true)
	}
	if args[0] == "create" {
		confirm := ""
		if _, err := commandFlags("credentials create", args[1:], func(f *flag.FlagSet) { f.StringVar(&confirm, "confirm", "", "") }); err != nil {
			return nil, err
		}
		if confirm == "" {
			return nil, errors.New("credentials create 需要 --confirm 凭证预览凭据")
		}
		return c.request("POST", "/cli/v1/credentials", map[string]string{"confirm": confirm}, true)
	}
	if args[0] != "preview" {
		return nil, errors.New("credentials 支持 list、preview --file、create --confirm")
	}
	file := ""
	if _, err := commandFlags("credentials preview", args[1:], func(f *flag.FlagSet) { f.StringVar(&file, "file", "", "") }); err != nil {
		return nil, err
	}
	if file == "" {
		return nil, errors.New("credentials preview 需要 --file 凭证.json|-；密钥建议从标准输入提交")
	}
	raw, err := readJSONInput(file)
	if err != nil {
		return nil, err
	}
	var input map[string]json.RawMessage
	if json.Unmarshal(raw, &input) != nil || input == nil {
		return nil, errors.New("凭证输入必须为 JSON 对象")
	}
	return c.request("POST", "/cli/v1/credentials/previews", input, true)
}
