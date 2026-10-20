package main

import (
	"encoding/json"
	"errors"
	"flag"
)

func (c *client) dashboardPlan(base string, args []string) (any, error) {
	if len(args) == 1 && args[0] == "get" {
		return c.request("GET", base+"/plan", nil, true)
	}
	if len(args) == 0 || args[0] != "save" {
		return nil, errors.New("plan 需要 get 或 save --file 设计计划.json --confirm")
	}
	file := ""
	confirm := false
	_, err := commandFlags("plan save", args[1:], func(f *flag.FlagSet) {
		f.StringVar(&file, "file", "", "")
		f.BoolVar(&confirm, "confirm", false, "")
	})
	if err != nil {
		return nil, err
	}
	if file == "" || !confirm {
		return nil, errors.New("plan save 需要 --file 设计计划.json|- --confirm")
	}
	raw, err := readJSONInput(file)
	if err != nil {
		return nil, err
	}
	return c.request("POST", base+"/plan", map[string]any{"plan": json.RawMessage(raw), "confirm": true}, true)
}
