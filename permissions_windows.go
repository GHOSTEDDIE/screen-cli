//go:build windows

package main

import (
	"errors"
	"os/exec"
	"os/user"
)

// Windows ignores POSIX permission bits. Reset only this dedicated CLI config
// directory's ACL, then remove inheritance and grant the current user's SID.
// New atomic credential files inherit that restricted ACL. No shell is used.
func protectConfigDirectory(path string) error {
	current, err := user.Current()
	if err != nil {
		return errors.New("无法读取当前 Windows 用户")
	}
	if err = exec.Command("icacls", path, "/reset", "/Q").Run(); err != nil {
		return errors.New("无法重置 CLI 配置目录权限")
	}
	if err = exec.Command("icacls", path, "/inheritancelevel:r", "/grant:r", "*"+current.Uid+":(OI)(CI)F", "/Q").Run(); err != nil {
		return errors.New("无法限制 CLI 配置目录为当前用户访问")
	}
	return nil
}
