#!/usr/bin/env bash
set -euo pipefail

PROJECT_DIR="${1:-$(pwd)}"
cd "$PROJECT_DIR"

for required in go.mod main.go config/config.go internal/router/router.go; do
  [[ -f "$required" ]] || { echo "Missing mainline source file: $required" >&2; exit 1; }
done

go version

OUT="android-wrapper/app/src/main/jniLibs/arm64-v8a/libhonoka.so"
mkdir -p "$(dirname "$OUT")"
rm -f "$OUT"

GOOS=android GOARCH=arm64 CGO_ENABLED=0 \
  go build -trimpath -ldflags='-s -w' -o "$OUT" .

[[ -s "$OUT" ]] || { echo "Build output is missing or empty: $OUT" >&2; exit 1; }
echo "Built $OUT"
