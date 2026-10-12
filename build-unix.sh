#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")" && pwd)"
FRONTEND="$ROOT/frontend"
BACKEND="$ROOT/backend"
EMBED="$BACKEND/internal/webui/dist"
OUT="$ROOT/dist"
command -v node >/dev/null || { echo 'Node.js is required to build.' >&2; exit 1; }
command -v npm >/dev/null || { echo 'npm is required to build.' >&2; exit 1; }
command -v go >/dev/null || { echo 'Go is required to build.' >&2; exit 1; }
cd "$FRONTEND"
if [[ -f package-lock.json ]]; then npm ci; else npm install; fi
npm run build
rm -rf "$EMBED" && mkdir -p "$EMBED"
cp -R "$FRONTEND/dist/." "$EMBED/"
cd "$BACKEND"
go test ./...
mkdir -p "$OUT"
case "$(uname -s)" in
  Darwin) go build -trimpath -ldflags '-s -w' -o "$OUT/EisenDay-macos" ./cmd/eisenday ;;
  Linux)  go build -trimpath -ldflags '-s -w' -o "$OUT/EisenDay-linux" ./cmd/eisenday ;;
  *) echo 'Unsupported OS. Use build-combined-windows.ps1 on Windows.' >&2; exit 1 ;;
esac
