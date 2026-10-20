#!/bin/sh
set -eu
cli_source=$(dirname -- "$0")/../dist
cli_destination=${HOME:?User home required}/.local/bin
while [ "$#" -gt 0 ]; do
  [ "$#" -ge 2 ] || { echo 'usage: install.sh [--from dist] [--bin-dir directory]' >&2; exit 1; }
  case "$1" in --from) cli_source=$2;; --bin-dir) cli_destination=$2;; *) echo 'unknown option' >&2; exit 1;; esac
  shift 2
done
case "$(uname -s)" in Darwin) cli_os=darwin;; Linux) cli_os=linux;; *) echo 'Use install.ps1 on Windows' >&2; exit 1;; esac
case "$(uname -m)" in arm64|aarch64) cli_arch=arm64;; x86_64|amd64) cli_arch=amd64;; *) echo 'unsupported architecture' >&2; exit 1;; esac
cli_binary=screen-cli_${cli_os}_${cli_arch}
cli_source=$(CDPATH= cd -- "$cli_source" && pwd)
cli_expected=$(awk -v name="$cli_binary" '$2==name {print $1}' "$cli_source/SHA256SUMS")
[ "${#cli_expected}" -eq 64 ] || { echo 'checksum entry missing' >&2; exit 1; }
if command -v sha256sum >/dev/null 2>&1; then cli_actual=$(sha256sum "$cli_source/$cli_binary" | awk '{print $1}'); else cli_actual=$(shasum -a 256 "$cli_source/$cli_binary" | awk '{print $1}'); fi
[ "$cli_expected" = "$cli_actual" ] || { echo 'checksum mismatch' >&2; exit 1; }
mkdir -p "$cli_destination"
cli_temporary=$(mktemp "$cli_destination/.screen-cli.XXXXXX")
trap 'rm -f "$cli_temporary"' EXIT HUP INT TERM
cp "$cli_source/$cli_binary" "$cli_temporary"
chmod 755 "$cli_temporary"
mv -f "$cli_temporary" "$cli_destination/screen-cli"
"$cli_destination/screen-cli" --version
