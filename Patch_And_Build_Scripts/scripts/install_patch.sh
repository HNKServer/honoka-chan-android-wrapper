#!/usr/bin/env bash
set -euo pipefail

if [[ $# -lt 1 ]]; then
  echo "Usage: $0 /path/to/honoka-chan-dev" >&2
  exit 2
fi

PROJECT_DIR="$(cd "$1" && pwd)"
PATCH_DIR="$(cd "$(dirname "$0")/.." && pwd)"

if [[ ! -f "$PROJECT_DIR/go.mod" ]] || ! grep -Eq '^module[[:space:]]+honoka-chan[[:space:]]*$' "$PROJECT_DIR/go.mod"; then
  echo "Unsupported project: go.mod module is not honoka-chan" >&2
  exit 3
fi

replace_file() {
  local src="$1"
  local dst="$2"
  [[ -f "$src" ]] || { echo "Patch file missing: $src" >&2; exit 4; }
  mkdir -p "$(dirname "$dst")"
  cp -f "$src" "$dst"
  echo "Replaced $dst"
}

# Direct full-file replacement. No backup files are created.
replace_file "$PATCH_DIR/replacement_files/main.go" "$PROJECT_DIR/main.go"
replace_file "$PATCH_DIR/replacement_files/config/config.go" "$PROJECT_DIR/config/config.go"
replace_file "$PATCH_DIR/replacement_files/internal/router/router.go" "$PROJECT_DIR/internal/router/router.go"
replace_file "$PATCH_DIR/replacement_files/internal/startup/database.go" "$PROJECT_DIR/internal/startup/database.go"
replace_file "$PATCH_DIR/replacement_files/internal/startup/startup.go" "$PROJECT_DIR/internal/startup/startup.go"
replace_file "$PATCH_DIR/replacement_files/internal/startup/legacy_common_units.go" "$PROJECT_DIR/internal/startup/legacy_common_units.go"
replace_file "$PATCH_DIR/replacement_files/internal/startup/legacy_ownership.go" "$PROJECT_DIR/internal/startup/legacy_ownership.go"
replace_file "$PATCH_DIR/replacement_files/internal/startup/legacy_reconcile.go" "$PROJECT_DIR/internal/startup/legacy_reconcile.go"
replace_file "$PATCH_DIR/replacement_files/internal/middleware/common.go" "$PROJECT_DIR/internal/middleware/common.go"
replace_file "$PATCH_DIR/replacement_files/internal/handler/api/api.go" "$PROJECT_DIR/internal/handler/api/api.go"

rm -f "$PROJECT_DIR/config/android_reload.go" "$PROJECT_DIR/internal/router/android.go"

if [[ -f "$PROJECT_DIR/config/json.go" ]] && grep -Eq 'SifCdnServer|AsCdnServer|type UserPrefs struct' "$PROJECT_DIR/config/json.go"; then
  rm -f "$PROJECT_DIR/config/json.go"
  echo "Removed obsolete config/json.go"
fi

# Clean backup artifacts created by earlier patch generations.
rm -f \
  "$PROJECT_DIR"/main.go.before_* \
  "$PROJECT_DIR"/config/config.go.before_* \
  "$PROJECT_DIR"/config/json.go.before_* \
  "$PROJECT_DIR"/internal/router/router.go.before_* \
  "$PROJECT_DIR"/internal/startup/database.go.before_* 2>/dev/null || true

mkdir -p "$PROJECT_DIR/android-wrapper"
cp -a "$PATCH_DIR/android-wrapper/." "$PROJECT_DIR/android-wrapper/"
echo "Merged unchanged Android GUI."

gofmt -w \
  "$PROJECT_DIR/main.go" \
  "$PROJECT_DIR/config/config.go" \
  "$PROJECT_DIR/internal/router/router.go" \
  "$PROJECT_DIR/internal/startup/database.go" \
  "$PROJECT_DIR/internal/startup/startup.go" \
  "$PROJECT_DIR/internal/startup/legacy_common_units.go" \
  "$PROJECT_DIR/internal/startup/legacy_ownership.go" \
  "$PROJECT_DIR/internal/startup/legacy_reconcile.go" \
  "$PROJECT_DIR/internal/middleware/common.go" \
  "$PROJECT_DIR/internal/handler/api/api.go"

echo "v28 source-reconciled mainline compatibility patch installed."
echo "No source backups were created."
