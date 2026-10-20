package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"mime/multipart"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type codeFile struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}
type workspace struct {
	TaskID        string            `json:"task_id"`
	Stage         string            `json:"stage,omitempty"`
	SourceIDs     []int             `json:"source_ids,omitempty"`
	PageID        int               `json:"page_id"`
	Kind          string            `json:"code_kind"`
	LockVersion   int               `json:"lock_version"`
	PolicyVersion string            `json:"policy_version"`
	Files         []codeFile        `json:"files"`
	Metadata      map[string]string `json:"metadata"`
	AgentGuide    string            `json:"local_agent_guide,omitempty"`
	Server        string            `json:"server"`
	WebURL        string            `json:"web_url"`
}

func workspacePath(dir string) string { return filepath.Join(dir, ".screen", "workspace.json") }
func readWorkspace(dir string) (workspace, error) {
	var w workspace
	raw, e := os.ReadFile(workspacePath(dir))
	if e != nil {
		return w, errors.New("当前目录尚未初始化，请运行 init 或 pull")
	}
	e = json.Unmarshal(raw, &w)
	if e == nil && (w.TaskID == "" || w.PageID <= 0) {
		e = errors.New("工作区记录无效")
	}
	return w, e
}
func safeLocalPath(name string) bool {
	if name == "" || filepath.IsAbs(name) || strings.ContainsAny(name, "\\\x00") || strings.Contains(name, ":") {
		return false
	}
	for _, p := range strings.Split(name, "/") {
		if p == ".." || p == "." || p == "" {
			return false
		}
	}
	return true
}
func writeWorkspace(dir string, w workspace, confirm bool) (any, error) {
	if w.TaskID == "" || w.PageID <= 0 {
		return nil, errors.New("系统未返回有效工作区")
	}
	files := append([]codeFile{}, w.Files...)
	for path, content := range w.Metadata {
		files = append(files, codeFile{Path: path, Content: content})
	}
	if w.AgentGuide != "" {
		files = append(files, codeFile{Path: ".screen/LOCAL_AGENT.md", Content: w.AgentGuide})
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	changes := []map[string]string{}
	for _, f := range files {
		if !safeLocalPath(f.Path) {
			return nil, errors.New("系统返回非法文件路径")
		}
		target := filepath.Join(dir, filepath.FromSlash(f.Path))
		for ancestor := target; ancestor != filepath.Dir(dir) && ancestor != "."; ancestor = filepath.Dir(ancestor) {
			info, e := os.Lstat(ancestor)
			if e == nil && info.Mode()&os.ModeSymlink != 0 {
				return nil, errors.New("本地写入路径包含链接")
			}
			if ancestor == dir {
				break
			}
		}
		old, e := os.ReadFile(target)
		if e != nil && !os.IsNotExist(e) {
			return nil, e
		}
		if string(old) == f.Content && e == nil {
			continue
		}
		action := "added"
		if e == nil {
			action = "modified"
		}
		changes = append(changes, map[string]string{"path": f.Path, "action": action})
	}
	if len(changes) > 0 && !confirm {
		existing := false
		for _, v := range changes {
			if v["action"] == "modified" {
				existing = true
			}
		}
		if existing {
			return map[string]any{"saved": false, "changes": changes, "next_action": "pull --confirm"}, nil
		}
	}
	for _, f := range files {
		if e := atomicFile(filepath.Join(dir, filepath.FromSlash(f.Path)), []byte(f.Content), 0600); e != nil {
			return nil, e
		}
	}
	if e := atomicJSON(workspacePath(dir), w); e != nil {
		return nil, e
	}
	return map[string]any{"saved": true, "task_id": w.TaskID, "page_id": w.PageID, "directory": dir, "changes": changes}, nil
}
func packDirectory(dir string) ([]byte, []string, error) {
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	excluded := []string{}
	skipped := map[string]bool{".git": true, ".screen": true, ".agents": true, ".codex": true, "node_modules": true, "dist": true, "out": true, "build": true, "__MACOSX": true}
	entries := []string{}
	e := filepath.WalkDir(dir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == dir {
			return nil
		}
		relative, e := filepath.Rel(dir, path)
		if e != nil {
			return e
		}
		relative = filepath.ToSlash(relative)
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("不允许打包链接：%s", relative)
		}
		if entry.IsDir() {
			if skipped[entry.Name()] {
				excluded = append(excluded, relative+"/")
				return filepath.SkipDir
			}
			if strings.HasPrefix(entry.Name(), ".") {
				return fmt.Errorf("不允许隐藏目录：%s", relative)
			}
			return nil
		}
		if strings.HasPrefix(entry.Name(), ".") {
			return fmt.Errorf("不允许隐藏文件：%s", relative)
		}
		info, e := entry.Info()
		if e != nil {
			return e
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("不允许特殊文件：%s", relative)
		}
		entries = append(entries, relative)
		return nil
	})
	if e != nil {
		return nil, nil, e
	}
	sort.Strings(entries)
	for _, name := range entries {
		file, e := os.Open(filepath.Join(dir, filepath.FromSlash(name)))
		if e != nil {
			return nil, nil, e
		}
		info, e := file.Stat()
		if e != nil {
			file.Close()
			return nil, nil, e
		}
		if info.Size() > 20<<20 {
			file.Close()
			return nil, nil, fmt.Errorf("文件过大：%s", name)
		}
		destination, e := writer.CreateHeader(&zip.FileHeader{Name: name, Method: zip.Deflate})
		if e == nil {
			_, e = io.Copy(destination, file)
		}
		file.Close()
		if e != nil {
			return nil, nil, e
		}
		if buffer.Len() > 21<<20 {
			return nil, nil, errors.New("工程包过大")
		}
	}
	if e = writer.Close(); e != nil {
		return nil, nil, e
	}
	return buffer.Bytes(), excluded, nil
}
func multipartBody(raw []byte, name string, fields map[string]string) ([]byte, string, error) {
	var b bytes.Buffer
	writer := multipart.NewWriter(&b)
	for k, v := range fields {
		if e := writer.WriteField(k, v); e != nil {
			return nil, "", e
		}
	}
	f, e := writer.CreateFormFile("file", name)
	if e != nil {
		return nil, "", e
	}
	if _, e = f.Write(raw); e != nil {
		return nil, "", e
	}
	if e = writer.Close(); e != nil {
		return nil, "", e
	}
	return b.Bytes(), writer.FormDataContentType(), nil
}
func (c *client) push(dir, zipPath, confirm string, preview bool) (any, error) {
	if preview == (confirm != "") {
		return nil, errors.New("请选择 push --preview 或 push --confirm <预检凭据>")
	}
	w, e := readWorkspace(dir)
	if e != nil {
		return nil, e
	}
	if w.Server != c.config.Server {
		return nil, errors.New("当前工作区属于另一个系统地址")
	}
	var raw []byte
	excluded := []string{}
	if zipPath != "" {
		f, err := os.Open(zipPath)
		if err != nil {
			return nil, err
		}
		raw, e = io.ReadAll(io.LimitReader(f, (20<<20)+1))
		f.Close()
	} else {
		raw, excluded, e = packDirectory(dir)
	}
	if e != nil {
		return nil, e
	}
	if len(raw) > 20<<20 {
		return nil, errors.New("工程包超过 20 MiB")
	}
	body, contentType, e := multipartBody(raw, "project.zip", map[string]string{"policy_version": w.PolicyVersion, "lock_version": fmt.Sprint(w.LockVersion), "confirm": confirm})
	if e != nil {
		return nil, e
	}
	data, e := c.send("POST", "/cli/v1/workspaces/"+w.TaskID+"/uploads", body, contentType, true)
	if e != nil {
		return nil, e
	}
	var result map[string]any
	if e = json.Unmarshal(data, &result); e != nil {
		return nil, e
	}
	result["excluded"] = excluded
	if confirm != "" {
		next, e := c.request("GET", "/cli/v1/workspaces/"+w.TaskID+"/export", nil, true)
		if e != nil {
			return nil, e
		}
		var updated workspace
		if e = json.Unmarshal(next, &updated); e != nil {
			return nil, e
		}
		updated.Server = w.Server
		updated.WebURL = w.WebURL
		if e = atomicJSON(workspacePath(dir), updated); e != nil {
			return nil, e
		}
	}
	return result, nil
}
