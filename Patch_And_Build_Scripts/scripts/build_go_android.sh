#!/usr/bin/env bash
set -euo pipefail

PROJECT_DIR="${1:-$(pwd)}"
cd "$PROJECT_DIR"

mkdir -p android-wrapper/app/src/main/jniLibs/arm64-v8a
GOOS=android GOARCH=arm64 CGO_ENABLED=0 \
  go build -trimpath -ldflags="-s -w" \
  -o android-wrapper/app/src/main/jniLibs/arm64-v8a/libhonoka.so .

echo "Built android-wrapper/app/src/main/jniLibs/arm64-v8a/libhonoka.so"
