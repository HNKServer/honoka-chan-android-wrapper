param(
    [Parameter(Mandatory=$true)]
    [string]$ProjectDir
)

$ErrorActionPreference = "Stop"
$PatchDir = Split-Path -Parent (Split-Path -Parent $MyInvocation.MyCommand.Path)

New-Item -ItemType Directory -Force -Path "$ProjectDir\config" | Out-Null
New-Item -ItemType Directory -Force -Path "$ProjectDir\handler" | Out-Null
New-Item -ItemType Directory -Force -Path "$ProjectDir\router" | Out-Null

Copy-Item "$PatchDir\go_patch\config\android_reload.go" "$ProjectDir\config\android_reload.go" -Force
Copy-Item "$PatchDir\go_patch\handler\android_reload.go" "$ProjectDir\handler\android_reload.go" -Force
Copy-Item "$PatchDir\go_patch\router\android.go" "$ProjectDir\router\android.go" -Force

if ((Test-Path "$ProjectDir\main.go") -and !(Test-Path "$ProjectDir\main.go.before_android_wrapper")) {
    Copy-Item "$ProjectDir\main.go" "$ProjectDir\main.go.before_android_wrapper"
}
Copy-Item "$PatchDir\go_patch\root\main.go" "$ProjectDir\main.go" -Force

if (Test-Path "$ProjectDir\android-wrapper") {
    Remove-Item "$ProjectDir\android-wrapper" -Recurse -Force
}
Copy-Item "$PatchDir\android-wrapper" "$ProjectDir\android-wrapper" -Recurse -Force

Write-Host "Patch installed. Next run scripts\prepare_server_base_zip.ps1 and scripts\build_go_android.ps1"
