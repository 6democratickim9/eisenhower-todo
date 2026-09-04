#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")" && pwd)"
cd "$ROOT"

if [[ "$(uname -s)" != "Darwin" ]]; then
  echo "ERROR: build_macos.sh must be run on macOS."
  exit 1
fi

for cmd in python3 npm ditto; do
  if ! command -v "$cmd" >/dev/null 2>&1; then
    echo "ERROR: '$cmd' is required but was not found."
    exit 1
  fi
done

HOST_ARCH="$(uname -m)"
TARGET_ARCH="${EISEN_DAY_TARGET_ARCH:-$HOST_ARCH}"
case "$TARGET_ARCH" in
  arm64|x86_64|universal2) ;;
  *)
    echo "ERROR: EISEN_DAY_TARGET_ARCH must be arm64, x86_64, or universal2."
    exit 1
    ;;
esac

if [[ "$TARGET_ARCH" == "universal2" ]]; then
  echo "NOTE: universal2 requires a universal2 Python installation and universal2-compatible native dependencies."
fi

VENV="$ROOT/.venv-build-macos"
PY="$VENV/bin/python"
PIP="$VENV/bin/pip"
PYINSTALLER="$VENV/bin/pyinstaller"

printf '\n[1/6] Preparing macOS Python build environment (%s)...\n' "$TARGET_ARCH"
if [[ ! -x "$PY" ]]; then
  python3 -m venv "$VENV"
fi
"$PY" -m pip install --upgrade pip
"$PIP" install -r requirements-macos.txt

printf '\n[2/6] Building React frontend...\n'
pushd frontend >/dev/null
if [[ -f package-lock.json ]]; then
  npm ci
else
  npm install
fi
npm run build
popd >/dev/null

printf '\n[3/6] Running backend tests...\n'
export PYTHONPATH="$ROOT"
"$PY" -m pytest -q backend/tests

printf '\n[4/6] Building macOS .app bundle...\n'
rm -rf build/EisenDay_macos dist/EisenDay dist/EisenDay.app
export EISEN_DAY_TARGET_ARCH="$TARGET_ARCH"
"$PYINSTALLER" --noconfirm --clean EisenDay_macos.spec

APP="$ROOT/dist/EisenDay.app"
if [[ ! -d "$APP" ]]; then
  echo "ERROR: PyInstaller did not produce dist/EisenDay.app"
  exit 1
fi

printf '\n[5/6] Verifying app bundle...\n'
if command -v codesign >/dev/null 2>&1; then
  if codesign --verify --deep --strict "$APP" >/dev/null 2>&1; then
    echo "Code signature check: OK"
  else
    echo "WARNING: codesign verification reported a problem. The app was still built."
  fi
fi

printf '\n[6/6] Creating macOS ZIP...\n'
ARCH_LABEL="$TARGET_ARCH"
RELEASE_DIR="$ROOT/release"
mkdir -p "$RELEASE_DIR"
ZIP="$RELEASE_DIR/EisenDay-macos-${ARCH_LABEL}.zip"
rm -f "$ZIP"
# ditto preserves macOS app bundle metadata, resource forks, and symlinks.
# The release archive contains only EisenDay.app; no backend/frontend source folders.
ditto -c -k --sequesterRsrc --keepParent "$APP" "$ZIP"

printf '\nDone:\n'
printf '  %s\n' "$APP"
printf '  %s  (user release; contains only EisenDay.app)\n' "$ZIP"
printf '\nRuntime use: double-click EisenDay.app. Python/Node/npm are not required on the target Mac.\n'
if [[ -z "${EISEN_DAY_CODESIGN_IDENTITY:-}" ]]; then
  printf 'This local build is not Developer ID/notarized. On another Mac, Gatekeeper may show a warning; use Finder > right-click > Open for the first launch.\n'
fi
