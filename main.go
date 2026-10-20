package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

var version = "dev"

func writeOutput(value any) {
	if raw, ok := value.(json.RawMessage); ok {
		var decoded any
		if json.Unmarshal(raw, &decoded) == nil {
			value = decoded
		}
	}
	_ = json.NewEncoder(os.Stdout).Encode(value)
}
func main() {
	result, err := run(os.Args[1:])
	if err != nil {
		var api *apiError
		if errors.As(err, &api) {
			writeOutput(map[string]any{"ok": false, "error": api})
			if api.Status == 401 {
				os.Exit(2)
			}
			if api.Status == 409 {
				os.Exit(3)
			}
		} else {
			writeOutput(map[string]any{"ok": false, "error": map[string]string{"error": "COMMAND_FAILED", "message": err.Error()}})
		}
		os.Exit(1)
	}
	writeOutput(map[string]any{"ok": true, "data": result})
}
func globalArguments(args []string) ([]string, string, string, string, error) {
	server, web, dir := "", "", "."
	remaining := []string{}
	for i := 0; i < len(args); i++ {
		name, value, assigned := strings.Cut(args[i], "=")
		if name != "--endpoint" && name != "--server" && name != "--web-url" && name != "--project" {
			remaining = append(remaining, args[i])
			continue
		}
		if !assigned {
			if i+1 >= len(args) {
				return nil, "", "", "", fmt.Errorf("%s 需要参数", name)
			}
			i++
			value = args[i]
		}
		switch name {
		case "--server", "--endpoint":
			if server != "" && strings.TrimRight(server, "/") != strings.TrimRight(value, "/") {
				return nil, "", "", "", errors.New("服务端地址参数冲突")
			}
			server = value
		case "--web-url":
			web = value
		case "--project":
			dir = value
		}
	}
	return remaining, server, web, dir, nil
}
func commandFlags(name string, args []string, configure func(*flag.FlagSet)) (*flag.FlagSet, error) {
	f := flag.NewFlagSet(name, flag.ContinueOnError)
	f.SetOutput(io.Discard)
	configure(f)
	// Standard flag parsing stops at the first positional; allow init dir --kind
	// by placing positionals after recognized flags without shell evaluation.
	options := []string{}
	positionals := []string{}
	for i := 0; i < len(args); i++ {
		a := args[i]
		if !strings.HasPrefix(a, "-") || a == "-" {
			positionals = append(positionals, a)
			continue
		}
		options = append(options, a)
		key, _, assigned := strings.Cut(strings.TrimLeft(a, "-"), "=")
		definition := f.Lookup(key)
		if definition == nil {
			continue
		}
		boolean, ok := definition.Value.(interface{ IsBoolFlag() bool })
		if !assigned && (!ok || !boolean.IsBoolFlag()) {
			if i+1 >= len(args) {
				return nil, fmt.Errorf("--%s 需要参数", key)
			}
			i++
			options = append(options, args[i])
		}
	}
	if e := f.Parse(append(options, positionals...)); e != nil {
		return nil, e
	}
	if name != "init" && name != "pull" && f.NArg() != 0 {
		return nil, errors.New("此命令不接受位置参数")
	}
	return f, nil
}
func run(args []string) (any, error) {
	args, server, web, dir, e := globalArguments(args)
	if e != nil {
		return nil, e
	}
	if len(args) == 0 || args[0] == "--help" || args[0] == "help" {
		return help(), nil
	}
	if args[0] == "--version" || args[0] == "version" {
		return map[string]string{"version": version}, nil
	}
	command := args[0]
	args = args[1:]
	if len(args) > 0 && (args[0] == "--help" || args[0] == "-h") {
		return help(), nil
	}
	if command == "config" {
		return configureClient(args, server, web)
	}
	c, e := newClient(server, web)
	if e != nil {
		return nil, e
	}
	dir, e = filepath.Abs(dir)
	if e != nil {
		return nil, e
	}
	switch command {
	case "sources":
		return c.sources(args)
	case "auth":
		if len(args) == 0 {
			return nil, errors.New("auth 需要 login、status 或 logout")
		}
		if args[0] != "login" && len(args) != 1 {
			return nil, errors.New("此 auth 命令不接受额外参数")
		}
		switch args[0] {
		case "login":
			noBrowser := false
			account := ""
			_, e := commandFlags("auth login", args[1:], func(f *flag.FlagSet) {
				f.BoolVar(&noBrowser, "no-browser", false, "")
				f.StringVar(&account, "account", "", "")
			})
			if e != nil {
				return nil, e
			}
			return c.loginAs(noBrowser, account)
		case "status":
			return c.request("GET", "/cli/v1/auth/status", nil, true)
		case "logout":
			return c.logout()
		}
		return nil, errors.New("不支持的 auth 命令")
	case "rules":
		data, e := c.request("GET", "/cli/v1/policy", nil, true)
		if e != nil {
			return nil, e
		}
		if len(args) == 1 && args[0] == "sync" {
			w, e := readWorkspace(dir)
			if e != nil {
				return nil, e
			}
			var policy struct {
				Version    string            `json:"version"`
				Metadata   map[string]string `json:"metadata"`
				AgentGuide string            `json:"local_agent_guide"`
			}
			if e = json.Unmarshal(data, &policy); e != nil {
				return nil, e
			}
			for name, content := range policy.Metadata {
				if !safeLocalPath(name) {
					return nil, errors.New("规范路径无效")
				}
				if e = atomicFile(filepath.Join(dir, name), []byte(content), 0600); e != nil {
					return nil, e
				}
			}
			if policy.AgentGuide != "" {
				if e = atomicFile(filepath.Join(dir, ".screen", "LOCAL_AGENT.md"), []byte(policy.AgentGuide), 0600); e != nil {
					return nil, e
				}
				w.AgentGuide = policy.AgentGuide
			}
			w.PolicyVersion = policy.Version
			w.Metadata = policy.Metadata
			if e = atomicJSON(workspacePath(dir), w); e != nil {
				return nil, e
			}
		} else if len(args) > 0 {
			return nil, errors.New("rules 仅支持 sync")
		}
		return data, nil
	case "init", "pull":
		kind, name, sources := "react", "", ""
		page := 0
		confirm := false
		flags, e := commandFlags(command, args, func(f *flag.FlagSet) {
			f.StringVar(&kind, "kind", "react", "")
			f.StringVar(&name, "name", "", "")
			f.StringVar(&sources, "source-ids", "", "")
			f.IntVar(&page, "page-id", 0, "")
			f.BoolVar(&confirm, "confirm", false, "")
		})
		if e != nil {
			return nil, e
		}
		if flags.NArg() > 1 {
			return nil, errors.New("只允许一个工作目录")
		}
		if flags.NArg() == 1 {
			dir, e = filepath.Abs(flags.Arg(0))
			if e != nil {
				return nil, e
			}
		}
		if command == "pull" && page <= 0 {
			return nil, errors.New("pull 需要 --page-id")
		}
		if command == "init" && page != 0 {
			return nil, errors.New("已有页面请使用 pull")
		}
		if command == "init" {
			entries, e := os.ReadDir(dir)
			if e == nil && len(entries) > 0 {
				return nil, errors.New("init 需要空目录")
			}
			if e != nil && !os.IsNotExist(e) {
				return nil, e
			}
		}
		ids := []int{}
		if sources != "" {
			for _, v := range strings.Split(sources, ",") {
				id, e := strconv.Atoi(v)
				if e != nil || id <= 0 {
					return nil, errors.New("source-ids 必须为数字编号列表")
				}
				ids = append(ids, id)
			}
		}
		var data json.RawMessage
		existing, readError := readWorkspace(dir)
		if command == "pull" && readError == nil && existing.PageID == page && existing.Server == c.config.Server {
			data, e = c.request("GET", "/cli/v1/workspaces/"+existing.TaskID+"/export", nil, true)
			if e == nil {
				var remote workspace
				if e = json.Unmarshal(data, &remote); e == nil && (remote.Stage == "completed" || remote.Stage == "cancelled") {
					if sources == "" {
						ids = remote.SourceIDs
					}
					data, e = c.request("POST", "/cli/v1/workspaces", map[string]any{"name": name, "page_id": page, "source_ids": ids}, true)
				}
			}
		} else {
			data, e = c.request("POST", "/cli/v1/workspaces", map[string]any{"code_kind": kind, "name": name, "page_id": page, "source_ids": ids}, true)
		}
		if e != nil {
			return nil, e
		}
		var w workspace
		if e = json.Unmarshal(data, &w); e != nil {
			return nil, e
		}
		w.Server = c.config.Server
		w.WebURL = c.config.WebURL
		return writeWorkspace(dir, w, confirm)
	case "push":
		preview := false
		confirm, zipPath := "", ""
		f, e := commandFlags("push", args, func(f *flag.FlagSet) {
			f.BoolVar(&preview, "preview", false, "")
			f.StringVar(&confirm, "confirm", "", "")
			f.StringVar(&zipPath, "zip", "", "")
		})
		if e != nil {
			return nil, e
		}
		if f.NArg() != 0 {
			return nil, errors.New("push 不接受位置参数")
		}
		return c.push(dir, zipPath, confirm, preview)
	case "context", "status", "cancel", "check", "preview", "data", "propose", "assets", "plan":
		w, e := readWorkspace(dir)
		if e != nil {
			return nil, e
		}
		if w.Server != c.config.Server {
			return nil, errors.New("工作区属于其他系统地址")
		}
		base := "/cli/v1/workspaces/" + w.TaskID
		switch command {
		case "plan":
			return c.dashboardPlan(base, args)
		case "context", "status":
			if len(args) > 0 {
				return nil, errors.New("此命令不接受参数")
			}
			return c.request("GET", base+"/"+command, nil, true)
		case "cancel":
			if len(args) > 0 {
				return nil, errors.New("cancel 不接受参数")
			}
			return c.request("POST", base+"/cancel", map[string]any{}, true)
		case "check":
			wait := false
			_, e := commandFlags("check", args, func(f *flag.FlagSet) { f.BoolVar(&wait, "wait", false, "") })
			if e != nil {
				return nil, e
			}
			data, e := c.request("POST", base+"/check", map[string]any{}, true)
			if e != nil || !wait {
				return data, e
			}
			var revision struct {
				ID int `json:"id"`
			}
			if e = json.Unmarshal(data, &revision); e != nil {
				return nil, e
			}
			return c.waitCheck(base, revision.ID)
		case "preview":
			noOpen := false
			_, e := commandFlags("preview", args, func(f *flag.FlagSet) { f.BoolVar(&noOpen, "no-open", false, "") })
			if e != nil {
				return nil, e
			}
			data, e := c.request("POST", base+"/preview", map[string]any{}, true)
			if e != nil {
				return nil, e
			}
			link := taskPreviewURL(c.config.WebURL, w.TaskID)
			if !noOpen {
				if e = openBrowser(link); e != nil {
					return map[string]any{"preview": data, "web_url": link, "opened": false}, nil
				}
			}
			return map[string]any{"preview": data, "web_url": link, "opened": !noOpen}, nil
		case "data", "propose":
			file, operation := "", ""
			_, e := commandFlags(command, args, func(f *flag.FlagSet) {
				f.StringVar(&file, "file", "", "")
				if command == "data" {
					f.StringVar(&operation, "operation", "", "")
				}
			})
			if e != nil {
				return nil, e
			}
			var input any = map[string]any{}
			if file != "" {
				raw, e := readJSONInput(file)
				if e != nil {
					return nil, e
				}
				input = json.RawMessage(raw)
			}
			if command == "data" {
				if operation == "" {
					return nil, errors.New("data 需要 --operation")
				}
				return c.request("POST", base+"/data", map[string]any{"operation": operation, "input": input}, true)
			}
			if file == "" {
				return nil, errors.New("propose 需要 --file")
			}
			return c.request("POST", base+"/proposals", input, true)
		case "assets":
			if len(args) == 0 || args[0] == "list" {
				if len(args) > 1 {
					return nil, errors.New("assets list 不接受额外参数")
				}
				return c.request("GET", base+"/assets", nil, true)
			}
			if args[0] != "add" {
				return nil, errors.New("assets 需要 list 或 add")
			}
			file := ""
			confirm := false
			_, e := commandFlags("assets add", args[1:], func(f *flag.FlagSet) { f.StringVar(&file, "file", "", ""); f.BoolVar(&confirm, "confirm", false, "") })
			if e != nil {
				return nil, e
			}
			if file == "" || file == "-" || !confirm {
				return nil, errors.New("assets add 需要 --file 图片路径 --confirm")
			}
			f, e := os.Open(file)
			if e != nil {
				return nil, e
			}
			raw, e := io.ReadAll(io.LimitReader(f, (10<<20)+1))
			f.Close()
			if e != nil {
				return nil, e
			}
			if len(raw) > 10<<20 {
				return nil, errors.New("单张图片超过 10 MiB")
			}
			body, typ, e := multipartBody(raw, filepath.Base(file), map[string]string{"purpose": "asset"})
			if e != nil {
				return nil, e
			}
			return c.send("POST", base+"/assets", body, typ, true)
		}
	}
	return nil, errors.New("未知命令，请运行 screen-cli --help")
}
func readJSONInput(file string) ([]byte, error) {
	var r io.Reader
	var f *os.File
	if file == "-" {
		r = os.Stdin
	} else {
		var e error
		f, e = os.Open(file)
		if e != nil {
			return nil, e
		}
		defer f.Close()
		r = f
	}
	raw, e := io.ReadAll(io.LimitReader(r, (64<<10)+1))
	if e != nil {
		return nil, e
	}
	if len(raw) > 64<<10 || !json.Valid(raw) {
		return nil, errors.New("请输入不超过 64 KiB 的有效 JSON")
	}
	return raw, nil
}
func (c *client) waitCheck(base string, id int) (any, error) {
	deadline := time.Now().Add(10 * time.Minute)
	for time.Now().Before(deadline) {
		data, e := c.request("GET", base+"/status", nil, true)
		if e != nil {
			return nil, e
		}
		var status struct {
			Revisions []struct {
				ID                   int    `json:"id"`
				Status               string `json:"status"`
				Log                  string `json:"build_log"`
				SourceMatchesCurrent bool   `json:"source_matches_current"`
				DraftMatchesCurrent  *bool  `json:"draft_matches_current"`
			} `json:"revisions"`
		}
		if e = json.Unmarshal(data, &status); e != nil {
			return nil, e
		}
		for _, revision := range status.Revisions {
			if revision.ID != id {
				continue
			}
			switch revision.Status {
			case "ready", "published":
				matches := revision.SourceMatchesCurrent
				if revision.DraftMatchesCurrent != nil {
					matches = *revision.DraftMatchesCurrent
				}
				if !matches {
					return nil, &apiError{Kind: "DRAFT_CHANGED", Message: "源码或设计计划已变化，请检查最新草稿", Status: 409}
				}
				return data, nil
			case "build_failed", "data_failed", "visual_failed", "cancelled":
				return nil, &apiError{Kind: "CHECK_NOT_PASSED", Message: revision.Log, Status: 422}
			}
		}
		time.Sleep(time.Second)
	}
	return nil, errors.New("等待检查超时，请使用 status 查看保存的检查结果")
}
func help() any {
	return map[string]any{"name": "screen-cli", "version": version, "usage": []string{"screen-cli config set --endpoint https://系统地址 [--web-url https://网页地址]", "screen-cli config show", "screen-cli auth login [--account 用户名] [--no-browser]", "screen-cli auth status|logout", "screen-cli sources [list]", "screen-cli sources test --file 连接.json|- [--task-id 任务编号]", "screen-cli sources preview --file 连接.json|- [--task-id 任务编号]", "screen-cli sources create --confirm 接入预览凭据", "screen-cli init <目录> [--kind react|html] [--name 名称] [--source-ids 1,2]", "screen-cli --project <目录> pull --page-id 编号 [--confirm] [--source-ids 1,2]", "screen-cli --project <目录> rules [sync]", "screen-cli --project <目录> context|status|cancel", "screen-cli --project <目录> plan get", "screen-cli --project <目录> plan save --file 设计计划.json|- --confirm", "screen-cli --project <目录> push --preview [--zip 文件.zip]", "screen-cli --project <目录> push --confirm 预检凭据 [--zip 文件.zip]", "screen-cli --project <目录> check [--wait]", "screen-cli --project <目录> preview [--no-open]", "screen-cli --project <目录> data --operation 操作 [--file 参数.json|-]", "screen-cli --project <目录> propose --file 申请.json|-", "screen-cli --project <目录> assets list|add --file 图片 --confirm"}, "creation": "init 会创建系统页面草稿，受当前账号创建权限限制", "publication": "CLI 无发布命令；预览后在系统网页确认发布", "exit_codes": map[string]int{"ok": 0, "error": 1, "authentication": 2, "conflict": 3}}
}
