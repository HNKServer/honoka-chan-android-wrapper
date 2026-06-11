param(
    [string]$ProjectDir = (Get-Location).Path
)

$ErrorActionPreference = "Stop"
Set-Location $ProjectDir

$outDir = "android-wrapper\app\src\main\assets"
$out = "$outDir\server-base.zip"
New-Item -ItemType Directory -Force -Path $outDir | Out-Null
if (Test-Path $out) { Remove-Item $out -Force }

if (!(Test-Path "config.json")) {
@'
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
'@ | Set-Content -Encoding UTF8 "config.json"
}

$temp = Join-Path $env:TEMP ("honoka-server-base-" + [guid]::NewGuid().ToString())
New-Item -ItemType Directory -Force -Path $temp | Out-Null

foreach ($item in @("assets", "static", "config.json", "privatekey.pem", "publickey.pem", "certificate.crt", "server.crt")) {
    if (Test-Path $item) {
        Copy-Item $item (Join-Path $temp $item) -Recurse -Force
    }
}

$archives = Join-Path $temp "static\Android\archives"
if (Test-Path $archives) {
    Remove-Item $archives -Recurse -Force
}

Compress-Archive -Path (Join-Path $temp "*") -DestinationPath $out -Force
Remove-Item $temp -Recurse -Force

Write-Host "Created $out"
Write-Host "Do NOT put huge archives into APK. Put them on the phone under /storage/emulated/0/Download/HonokaData/static/Android/archives or choose another public path in the app."
