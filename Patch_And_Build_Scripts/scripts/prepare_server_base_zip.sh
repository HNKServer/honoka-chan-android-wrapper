#!/usr/bin/env bash
set -euo pipefail

PROJECT_DIR="${1:-$(pwd)}"
cd "$PROJECT_DIR"

OUT_DIR="android-wrapper/app/src/main/assets"
OUT="$OUT_DIR/server-base.zip"
mkdir -p "$OUT_DIR"
rm -f "$OUT"

TMP_DEFAULT_CONFIG=""
if [[ ! -f config.json ]]; then
  TMP_DEFAULT_CONFIG="$(mktemp -d)"
  cat > "$TMP_DEFAULT_CONFIG/config.json" <<'JSON'
{
  "app_name": "honoka-chan",
  "settings": {
    "server_port": "8080",
    "sif_cdn_server": "http://127.0.0.1:8080/static",
    "as_cdn_server": "http://127.0.0.1:8080/static"
  },
  "user_prefs": {
    "name": "梦路 @bilibili",
    "level": 1028,
    "exp_numerator": 1089696,
    "exp_denominator": 1207185,
    "game_coin": 112124104,
    "sns_coin": 0,
    "energy_max": 417,
    "over_max_energy": 0,
    "invite_code": "377385143"
  }
}
JSON
  cp "$TMP_DEFAULT_CONFIG/config.json" ./config.json
fi

# Put small runtime files into APK assets. Huge archives must stay outside APK.
zip -r "$OUT" \
  assets \
  static \
  config.json \
  privatekey.pem \
  publickey.pem \
  certificate.crt \
  server.crt \
  -x "static/Android/archives/*" "static/Android/archives/**" >/dev/null

echo "Created $OUT"
echo "Do NOT put huge archives into APK. Put them on the phone under /storage/emulated/0/Download/HonokaData/static/Android/archives or choose another public path in the app."
