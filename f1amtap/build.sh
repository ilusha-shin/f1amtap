#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")"

echo "=== f1amtap build ==="
if ! command -v go >/dev/null 2>&1; then
    echo "[ERROR] Go compiler not found in PATH!"
    echo "Install: https://go.dev/dl/  (or: brew install go / sudo apt install golang-go)"
    exit 1
fi

go version
echo
echo "go mod tidy..."
go mod tidy

OUT="f1amtap"
# Linux GUI (Gio + dialog/GTK) обычно нужен CGO
export CGO_ENABLED="${CGO_ENABLED:-1}"

go build -ldflags="-s -w" -o "${OUT}" ./cmd/f1amtap

echo "OK: $(pwd)/${OUT}"
echo "  CLI: ./${OUT} input.mid"
echo "  GUI: ./${OUT} --gui"
