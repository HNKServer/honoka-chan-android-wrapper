#!/usr/bin/env bash
set -euo pipefail

source "$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)/common.sh"

log "step 1/4: prepare runtime zip"
"$SCRIPT_DIR/prepare_runtime_zip.sh"

log "step 2/4: build Go library arm64-v8a"
"$SCRIPT_DIR/build_go_android_windows.sh" arm64-v8a

log "step 3/4: build Go library armeabi-v7a"
"$SCRIPT_DIR/build_go_android_windows.sh" armeabi-v7a

log "step 4/4: build Go library x86_64"
"$SCRIPT_DIR/build_go_android_windows.sh" x86_64

log "android runtime assets and JNI libraries are ready"
log "open android/ in Android Studio to build the APK"
