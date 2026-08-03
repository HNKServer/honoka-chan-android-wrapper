#!/usr/bin/env bash
set -euo pipefail

PROJECT_DIR="${1:-$(pwd)}"
cd "$PROJECT_DIR"

for required in \
  assets/main.db \
  assets/certs/privatekey.pem \
  assets/certs/publickey.pem \
  assets/certs/certificate.crt \
  assets/certs/server.crt \
  static; do
  [[ -e "$required" ]] || { echo "Missing mainline runtime resource: $required" >&2; exit 1; }
done

if [[ ! -f config.json ]]; then
cat > config.json <<'JSON'
{
    "app_name": "honoka-chan",
    "settings": {
        "listen_port": "8080",
        "cdn_server": "http://127.0.0.1:8080/static",
        "unlock_all_special_rotation": false
    }
}
JSON
fi

OUT_DIR="android-wrapper/app/src/main/assets"
OUT="$OUT_DIR/server-base.zip"
mkdir -p "$OUT_DIR"
rm -f "$OUT"

TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT
cp -R assets "$TMP/assets"
cp -R static "$TMP/static"
cp config.json "$TMP/config.json"
rm -f "$TMP/assets/data.db" "$TMP/assets/data.db-wal" "$TMP/assets/data.db-shm"
rm -rf "$TMP/static/Android/archives"

(
  cd "$TMP"
  zip -qr "$PROJECT_DIR/$OUT" assets static config.json
)

[[ -s "$OUT" ]] || { echo "Failed to create $OUT" >&2; exit 1; }
HASH="$(sha256sum "$OUT" | awk '{print $1}')"
printf '%s\n' "$HASH" > "$OUT_DIR/server-base.sha256"
echo "Created $OUT"
echo "SHA-256: $HASH"
echo "Mainline certs are packaged from assets/certs/."
echo "Huge archives remain outside APK and are selected from the Android GUI."
