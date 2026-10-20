#!/bin/sh
set -eu
cli_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
cli_version=$(cat "$cli_root/VERSION")
if [ "$#" -gt 0 ]; then [ "$#" -eq 2 ] && [ "$1" = --version ] || { echo 'usage: build.sh [--version 0.1.0]' >&2; exit 1; }; cli_version=$2; fi
case "$cli_version" in *[!a-zA-Z0-9._-]*|'') echo 'invalid version' >&2; exit 1;; esac
cd "$cli_root"
go test ./...
mkdir -p dist
for cli_target in darwin/amd64 darwin/arm64 linux/amd64 linux/arm64 windows/amd64 windows/arm64; do
  cli_os=${cli_target%/*}; cli_arch=${cli_target#*/}; cli_suffix=
  [ "$cli_os" != windows ] || cli_suffix=.exe
  CGO_ENABLED=0 GOOS=$cli_os GOARCH=$cli_arch go build -trimpath -buildvcs=false -ldflags "-s -w -X main.version=$cli_version" -o "dist/screen-cli_${cli_os}_${cli_arch}${cli_suffix}" .
done
go run ./scripts/package.go --version "$cli_version"
