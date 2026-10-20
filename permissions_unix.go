//go:build !windows

package main

import "os"

func protectConfigDirectory(path string) error { return os.Chmod(path, 0700) }
