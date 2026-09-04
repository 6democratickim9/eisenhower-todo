#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")" && pwd)"
cd "$ROOT"

if [[ "$(uname -s)" != "Darwin" ]]; then
  echo "ERROR: run_macos_dev.sh must be run on macOS."
  exit 1
fi

VENV="$ROOT/.venv-macos"
if [[ ! -x "$VENV/bin/python" ]]; then
  python3 -m venv "$VENV"
fi
"$VENV/bin/python" -m pip install --upgrade pip
"$VENV/bin/pip" install -r requirements-macos.txt

pushd frontend >/dev/null
if [[ -f package-lock.json ]]; then npm ci; else npm install; fi
npm run build
popd >/dev/null

export PYTHONPATH="$ROOT"
exec "$VENV/bin/python" launcher.py
