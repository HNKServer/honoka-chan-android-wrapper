#!/usr/bin/env bash
set -euo pipefail

PROJECT_DIR=""
while [[ $# -gt 0 ]]; do
  case "$1" in
    --project-dir|-p)
      PROJECT_DIR="$2"; shift 2 ;;
    *) echo "Unknown argument: $1"; exit 1 ;;
  esac
done

if [[ -z "$PROJECT_DIR" ]]; then
  echo "Usage: $0 --project-dir /path/to/honoka-chan-termux" >&2
  exit 1
fi

PATCH_DIR="$(cd "$(dirname "$0")/.." && pwd)"
mkdir -p "$PROJECT_DIR/config" "$PROJECT_DIR/handler" "$PROJECT_DIR/router"

[[ -f "$PROJECT_DIR/config/json.go" && ! -f "$PROJECT_DIR/config/json.go.before_android_wrapper" ]] && cp "$PROJECT_DIR/config/json.go" "$PROJECT_DIR/config/json.go.before_android_wrapper"
[[ -f "$PROJECT_DIR/main.go" && ! -f "$PROJECT_DIR/main.go.before_android_wrapper" ]] && cp "$PROJECT_DIR/main.go" "$PROJECT_DIR/main.go.before_android_wrapper"
[[ -f "$PROJECT_DIR/handler/allstars.go" && ! -f "$PROJECT_DIR/handler/allstars.go.before_android_wrapper" ]] && cp "$PROJECT_DIR/handler/allstars.go" "$PROJECT_DIR/handler/allstars.go.before_android_wrapper"

cp "$PATCH_DIR/go_patch/config/json.go" "$PROJECT_DIR/config/json.go"
cp "$PATCH_DIR/go_patch/config/android_reload.go" "$PROJECT_DIR/config/android_reload.go"
cp "$PATCH_DIR/go_patch/handler/android_reload.go" "$PROJECT_DIR/handler/android_reload.go"
cp "$PATCH_DIR/go_patch/router/android.go" "$PROJECT_DIR/router/android.go"
cp "$PATCH_DIR/go_patch/root/main.go" "$PROJECT_DIR/main.go"

rm -rf "$PROJECT_DIR/android-wrapper"
cp -R "$PATCH_DIR/android-wrapper" "$PROJECT_DIR/android-wrapper"

ALLSTARS="$PROJECT_DIR/handler/allstars.go"
if [[ -f "$ALLSTARS" ]] && ! grep -q 'UnlockAllSpecialRotation' "$ALLSTARS"; then
python3 - "$ALLSTARS" <<'PY'
import re, sys
p = sys.argv[1]
s = open(p, encoding='utf-8').read()
new = '''liveDailyList := []model.LiveDaily{}
	liveDailyQuery := MainEng.Table("m_live_daily").Cols("id,live_id")
	if !config.Conf.Settings.UnlockAllSpecialRotation {
		liveDailyQuery = liveDailyQuery.Where("weekday = ?", weekday)
	}
	err := liveDailyQuery.OrderBy("id ASC").Find(&liveDailyList)
	CheckErr(err)'''
pat = r'liveDailyList := \[\]model\.LiveDaily\{\}\s*err := MainEng\.Table\("m_live_daily"\)\.Where\("weekday = \?", weekday\)\.Cols\("id,live_id"\)\.Find\(&liveDailyList\)\s*CheckErr\(err\)'
s2, n = re.subn(pat, new, s, count=1)
if n == 0:
    print('WARNING: could not patch handler/allstars.go daily rotation query', file=sys.stderr)
else:
    open(p, 'w', encoding='utf-8').write(s2)
PY
fi

gofmt -w "$PROJECT_DIR/config/json.go" "$PROJECT_DIR/config/android_reload.go" "$PROJECT_DIR/handler/android_reload.go" "$PROJECT_DIR/router/android.go" "$PROJECT_DIR/main.go" "$PROJECT_DIR/handler/allstars.go" 2>/dev/null || true

echo "Patch installed. Next run scripts/prepare_server_base_zip.sh and scripts/build_go_android.sh"
