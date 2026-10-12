#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")"
if [[ "$(uname -s)" != Darwin ]]; then
  echo "Run this script on macOS." >&2
  exit 1
fi
bash ./build-unix.sh
echo "Built: $PWD/dist/EisenDay-macos"
echo "Run that file to open EisenDay in your default browser. Press Ctrl+C to stop."
