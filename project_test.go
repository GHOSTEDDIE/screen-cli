package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"testing"
)

func TestDeterministicPackAndDeclaredExclusions(t *testing.T) {
	dir := t.TempDir()
	atomicFile(filepath.Join(dir, "index.html"), []byte("旧业务结果：0"), 0600)
	atomicFile(filepath.Join(dir, ".git", "config"), []byte("private"), 0600)
	atomicFile(filepath.Join(dir, ".screen", "workspace.json"), []byte("metadata"), 0600)
	first, excluded, e := packDirectory(dir)
	if e != nil {
		t.Fatal(e)
	}
	second, _, e := packDirectory(dir)
	if e != nil || !bytes.Equal(first, second) {
		t.Fatal("same files changed package fingerprint")
	}
	if len(excluded) != 2 {
		t.Fatal(excluded)
	}
	r, e := zip.NewReader(bytes.NewReader(first), int64(len(first)))
	if e != nil || len(r.File) != 1 || r.File[0].Name != "index.html" {
		t.Fatal("unexpected archive")
	}
}
func TestPackRejectsSymlinksAndSecrets(t *testing.T) {
	dir := t.TempDir()
	atomicFile(filepath.Join(dir, ".env"), []byte("private"), 0600)
	if _, _, e := packDirectory(dir); e == nil {
		t.Fatal("secret accepted")
	}
	os.Remove(filepath.Join(dir, ".env"))
	if e := os.Symlink("/etc/passwd", filepath.Join(dir, "file.ts")); e != nil {
		t.Fatal(e)
	}
	if _, _, e := packDirectory(dir); e == nil {
		t.Fatal("symlink accepted")
	}
}
func TestPullRequiresConfirmationBeforeOverwriting(t *testing.T) {
	dir := t.TempDir()
	atomicFile(filepath.Join(dir, "index.html"), []byte("用户未保存的修改"), 0600)
	w := workspace{TaskID: "task", PageID: 1, Files: []codeFile{{Path: "index.html", Content: "系统草稿"}}}
	result, e := writeWorkspace(dir, w, false)
	if e != nil {
		t.Fatal(e)
	}
	raw, _ := os.ReadFile(filepath.Join(dir, "index.html"))
	if string(raw) != "用户未保存的修改" || result.(map[string]any)["saved"] != false {
		t.Fatal("overwrite before confirmation")
	}
	if _, e = writeWorkspace(dir, w, true); e != nil {
		t.Fatal(e)
	}
}
func TestPullRejectsPathTraversalAndSymlinkAncestor(t *testing.T) {
	dir := t.TempDir()
	for _, path := range []string{"../secret.ts", "/tmp/secret.ts", "src/a\\b.ts"} {
		if _, e := writeWorkspace(dir, workspace{TaskID: "task", PageID: 1, Files: []codeFile{{Path: path}}}, true); e == nil {
			t.Errorf("accepted %s", path)
		}
	}
	other := t.TempDir()
	os.Symlink(other, filepath.Join(dir, "src"))
	if _, e := writeWorkspace(dir, workspace{TaskID: "task", PageID: 1, Files: []codeFile{{Path: "src/App.tsx", Content: "new"}}}, true); e == nil {
		t.Fatal("symlink write accepted")
	}
}
func TestInitFlagsAfterDirectoryAndExplicitFalse(t *testing.T) {
	kind := ""
	confirm := true
	f, e := commandFlags("init", []string{"directory", "--kind", "html", "--confirm=false"}, func(f *flag.FlagSet) {
		f.StringVar(&kind, "kind", "react", "")
		f.BoolVar(&confirm, "confirm", true, "")
	})
	if e != nil || kind != "html" || confirm || f.Arg(0) != "directory" {
		t.Fatal(kind, confirm, e)
	}
}
func TestMultipartRetainsConfirmationAndFile(t *testing.T) {
	body, kind, e := multipartBody([]byte("zip"), "project.zip", map[string]string{"confirm": "reviewed"})
	if e != nil || kind == "" || !bytes.Contains(body, []byte("reviewed")) {
		t.Fatal(e)
	}
}
func TestWorkspaceRoundTrip(t *testing.T) {
	w := workspace{TaskID: "t", PageID: 4, Kind: "react", LockVersion: 7, PolicyVersion: "policy", Metadata: map[string]string{"AGENTS.md": "rules"}}
	dir := t.TempDir()
	if _, e := writeWorkspace(dir, w, true); e != nil {
		t.Fatal(e)
	}
	saved, e := readWorkspace(dir)
	if e != nil || saved.LockVersion != 7 || saved.TaskID != "t" {
		t.Fatal(saved, e)
	}
	raw, _ := json.Marshal(saved)
	if !bytes.Contains(raw, []byte("policy")) {
		t.Fatal("policy lost")
	}
}
