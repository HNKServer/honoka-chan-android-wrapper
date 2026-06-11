#!/usr/bin/env bash
set -euo pipefail

PROJECT_DIR="${1:-}"
if [[ -z "$PROJECT_DIR" ]]; then
  echo "Usage: scripts/install_patch.sh /path/to/honoka-chan-termux" >&2
  exit 1
fi

PATCH_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$PROJECT_DIR"

mkdir -p config handler router
cp "$PATCH_DIR/go_patch/config/android_reload.go" config/android_reload.go
cp "$PATCH_DIR/go_patch/handler/android_reload.go" handler/android_reload.go
cp "$PATCH_DIR/go_patch/router/android.go" router/android.go

if [[ -f main.go && ! -f main.go.before_android_wrapper ]]; then
  cp main.go main.go.before_android_wrapper
fi
cp "$PATCH_DIR/go_patch/root/main.go" main.go

rm -rf android-wrapper
cp -R "$PATCH_DIR/android-wrapper" android-wrapper

echo "Patch installed. Next run: scripts/prepare_server_base_zip.sh and scripts/build_go_android.sh"
