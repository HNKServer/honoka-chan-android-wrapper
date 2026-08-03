param(
    [string]$ProjectDir = (Get-Location).Path
)

$ErrorActionPreference = "Stop"
Set-Location $ProjectDir

$required = @(
    "assets\main.db",
    "assets\certs\privatekey.pem",
    "assets\certs\publickey.pem",
    "assets\certs\certificate.crt",
    "assets\certs\server.crt",
    "static"
)
foreach ($item in $required) {
    if (!(Test-Path $item)) {
        throw "Missing mainline runtime resource: $item"
    }
}

if (!(Test-Path "config.json")) {
    $defaultConfig = @'
{
    "app_name": "honoka-chan",
    "settings": {
        "listen_port": "8080",
        "cdn_server": "http://127.0.0.1:8080/static",
        "unlock_all_special_rotation": false
    }
}
'@
    $utf8NoBom = New-Object System.Text.UTF8Encoding($false)
    [System.IO.File]::WriteAllText((Join-Path (Get-Location) "config.json"), $defaultConfig + "`n", $utf8NoBom)
}

$outDir = "android-wrapper\app\src\main\assets"
$out = "$outDir\server-base.zip"
New-Item -ItemType Directory -Force -Path $outDir | Out-Null
if (Test-Path $out) { Remove-Item $out -Force }

$temp = Join-Path $env:TEMP ("honoka-mainline-server-base-" + [guid]::NewGuid().ToString())
New-Item -ItemType Directory -Force -Path $temp | Out-Null
try {
    Copy-Item "assets" (Join-Path $temp "assets") -Recurse -Force
    Copy-Item "static" (Join-Path $temp "static") -Recurse -Force
    Copy-Item "config.json" (Join-Path $temp "config.json") -Force

    # User DB is runtime state. Never bake it into an APK update.
    foreach ($userDbFile in @("data.db", "data.db-wal", "data.db-shm")) {
        $candidate = Join-Path $temp ("assets\" + $userDbFile)
        if (Test-Path $candidate) { Remove-Item $candidate -Force }
    }

    $archives = Join-Path $temp "static\Android\archives"
    if (Test-Path $archives) {
        Remove-Item $archives -Recurse -Force
    }

    Compress-Archive -Path (Join-Path $temp "*") -DestinationPath $out -Force
} finally {
    if (Test-Path $temp) { Remove-Item $temp -Recurse -Force }
}

if (!(Test-Path $out) -or (Get-Item $out).Length -le 0) {
    throw "Failed to create $out"
}

$hash = (Get-FileHash -Algorithm SHA256 $out).Hash.ToLowerInvariant()
$hashFile = Join-Path $outDir "server-base.sha256"
$utf8NoBom = New-Object System.Text.UTF8Encoding($false)
[System.IO.File]::WriteAllText((Join-Path (Get-Location) $hashFile), $hash + "`n", $utf8NoBom)

Write-Host "Created $out"
Write-Host "SHA-256: $hash"
Write-Host "Mainline certs are packaged from assets/certs/."
Write-Host "Huge archives remain outside APK and are selected from the Android GUI."
