#!/usr/bin/env bash
set -euo pipefail

source "$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)/common.sh"

require_cmd go

ABI="${1:-arm64-v8a}"
MIN_SDK="${MIN_SDK:-26}"
NDK_ROOT="${ANDROID_NDK_HOME:-${ANDROID_NDK_ROOT:-}}"

[[ -n "$NDK_ROOT" ]] || die "ANDROID_NDK_HOME is not set"
[[ -d "$NDK_ROOT" ]] || die "ANDROID_NDK_HOME does not exist: $NDK_ROOT"

HOST_TAG="${ANDROID_NDK_HOST_TAG:-linux-x86_64}"
TOOLCHAIN="$NDK_ROOT/toolchains/llvm/prebuilt/$HOST_TAG/bin"
[[ -d "$TOOLCHAIN" ]] || die "NDK toolchain not found: $TOOLCHAIN"

case "$ABI" in
  arm64-v8a)
    export GOOS=android
    export GOARCH=arm64
    export CGO_ENABLED=1
    export CC="$TOOLCHAIN/aarch64-windows-android${MIN_SDK}-clang"
    OUTPUT_DIR="$ANDROID_DIR/app/src/main/jniLibs/arm64-v8a"
    ;;
  armeabi-v7a)
    export GOOS=android
    export GOARCH=arm
    export GOARM=7
    export CGO_ENABLED=1
    export CC="$TOOLCHAIN/armv7a-linux-androideabi${MIN_SDK}-clang"
    OUTPUT_DIR="$ANDROID_DIR/app/src/main/jniLibs/armeabi-v7a"
    ;;
  x86_64)
    export GOOS=android
    export GOARCH=amd64
    export CGO_ENABLED=1
    export CC="$TOOLCHAIN/x86_64-linux-android${MIN_SDK}-clang"
    OUTPUT_DIR="$ANDROID_DIR/app/src/main/jniLibs/x86_64"
    ;;
  *)
    die "unsupported abi: $ABI"
    ;;
esac

[[ -x "$CC" ]] || die "clang not found: $CC"

mkdir -p "$OUTPUT_DIR"

cd "$PROJECT_ROOT"

log "building Go shared library for $ABI"
go build -trimpath -buildmode=c-shared \
  -ldflags="-s -w" \
  -o "$OUTPUT_DIR/libhonokachan.so" \
  ./cmd/honoka-android

rm -f "$OUTPUT_DIR/libhonokachan.h"

log "built $OUTPUT_DIR/libhonokachan.so"
