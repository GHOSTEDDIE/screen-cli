// Release assets use only the standard library; no server implementation is linked.
package main

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type asset struct {
	Name   string `json:"name"`
	Kind   string `json:"kind"`
	OS     string `json:"os,omitempty"`
	Arch   string `json:"arch,omitempty"`
	Size   int64  `json:"size"`
	SHA256 string `json:"sha256"`
}

func main() {
	version := flag.String("version", "", "release version")
	flag.Parse()
	if *version == "" {
		panic("version required")
	}
	if e := packageRelease(*version); e != nil {
		panic(e)
	}
}
func packageRelease(version string) error {
	zf, e := os.Create("dist/screen-cli-skill.zip")
	if e != nil {
		return e
	}
	zw := zip.NewWriter(zf)
	e = filepath.WalkDir("skills/screen-cli", func(path string, d os.DirEntry, e error) error {
		if e != nil {
			return e
		}
		if d.IsDir() {
			return nil
		}
		if d.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("skill contains link")
		}
		raw, e := os.ReadFile(path)
		if e != nil {
			return e
		}
		entry, e := zw.Create(strings.TrimPrefix(filepath.ToSlash(path), "skills/"))
		if e != nil {
			return e
		}
		_, e = entry.Write(raw)
		return e
	})
	closeErr := zw.Close()
	fileErr := zf.Close()
	if e != nil {
		return e
	}
	if closeErr != nil {
		return closeErr
	}
	if fileErr != nil {
		return fileErr
	}
	files := []asset{}
	for _, target := range []string{"darwin_amd64", "darwin_arm64", "linux_amd64", "linux_arm64", "windows_amd64", "windows_arm64"} {
		parts := strings.Split(target, "_")
		name := "screen-cli_" + target
		if parts[0] == "windows" {
			name += ".exe"
		}
		files = append(files, asset{Name: name, Kind: "binary", OS: parts[0], Arch: parts[1]})
	}
	files = append(files, asset{Name: "screen-cli-skill.zip", Kind: "skill"})
	for _, name := range []string{"install.sh", "install.ps1"} {
		raw, e := os.ReadFile("scripts/" + name)
		if e != nil {
			return e
		}
		if e = os.WriteFile("dist/"+name, raw, 0644); e != nil {
			return e
		}
		files = append(files, asset{Name: name, Kind: "installer"})
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Name < files[j].Name })
	var sums strings.Builder
	measure := func(a *asset) error {
		f, e := os.Open(filepath.Join("dist", a.Name))
		if e != nil {
			return e
		}
		defer f.Close()
		h := sha256.New()
		a.Size, e = io.Copy(h, f)
		a.SHA256 = hex.EncodeToString(h.Sum(nil))
		return e
	}
	for i := range files {
		if e = measure(&files[i]); e != nil {
			return e
		}
		fmt.Fprintf(&sums, "%s  %s\n", files[i].SHA256, files[i].Name)
	}
	if e = os.WriteFile("dist/SHA256SUMS", []byte(sums.String()), 0644); e != nil {
		return e
	}
	sum := asset{Name: "SHA256SUMS", Kind: "checksums"}
	if e = measure(&sum); e != nil {
		return e
	}
	files = append(files, sum)
	raw, e := json.MarshalIndent(struct {
		Version string  `json:"version"`
		Files   []asset `json:"files"`
	}{version, files}, "", "  ")
	if e != nil {
		return e
	}
	return os.WriteFile("dist/manifest.json", append(raw, '\n'), 0644)
}
