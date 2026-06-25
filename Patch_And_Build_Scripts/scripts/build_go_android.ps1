param(
    [string]$ProjectDir = (Get-Location).Path
)

$ErrorActionPreference = "Stop"
Set-Location $ProjectDir

New-Item -ItemType Directory -Force -Path "android-wrapper\app\src\main\jniLibs\arm64-v8a" | Out-Null
$env:GOOS = "android"
$env:GOARCH = "arm64"
$env:CGO_ENABLED = "0"
go build -trimpath -ldflags="-s -w" -o "android-wrapper\app\src\main\jniLibs\arm64-v8a\libhonoka.so" .

if ($LASTEXITCODE -ne 0) {
    throw "go build failed with exit code $LASTEXITCODE"
}

Write-Host "Built android-wrapper\app\src\main\jniLibs\arm64-v8a\libhonoka.so"
