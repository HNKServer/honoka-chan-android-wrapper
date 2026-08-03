param(
    [string]$ProjectDir = (Get-Location).Path
)

$ErrorActionPreference = "Stop"
Set-Location $ProjectDir

foreach ($required in @("go.mod", "main.go", "config\config.go", "internal\router\router.go")) {
    if (!(Test-Path $required)) {
        throw "Missing mainline source file: $required"
    }
}

$goVersion = (& go version) 2>&1
if ($LASTEXITCODE -ne 0) {
    throw "Go is not available in PATH"
}
Write-Host $goVersion

$out = "android-wrapper\app\src\main\jniLibs\arm64-v8a\libhonoka.so"
New-Item -ItemType Directory -Force -Path (Split-Path $out -Parent) | Out-Null
if (Test-Path $out) { Remove-Item $out -Force }

$oldGOOS = $env:GOOS
$oldGOARCH = $env:GOARCH
$oldCGO = $env:CGO_ENABLED
try {
    $env:GOOS = "android"
    $env:GOARCH = "arm64"
    $env:CGO_ENABLED = "0"

    & go build -trimpath -ldflags="-s -w" -o $out .
    if ($LASTEXITCODE -ne 0) {
        throw "go build failed with exit code $LASTEXITCODE"
    }
} finally {
    $env:GOOS = $oldGOOS
    $env:GOARCH = $oldGOARCH
    $env:CGO_ENABLED = $oldCGO
}

if (!(Test-Path $out) -or (Get-Item $out).Length -le 0) {
    throw "go build returned success but output is missing or empty: $out"
}

Write-Host "Built $out"
