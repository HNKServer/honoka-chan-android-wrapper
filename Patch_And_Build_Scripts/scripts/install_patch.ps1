param(
    [Parameter(Mandatory=$true)]
    [string]$ProjectDir
)

$ErrorActionPreference = "Stop"
$PatchDir = Split-Path -Parent (Split-Path -Parent $MyInvocation.MyCommand.Path)
$ProjectDir = (Resolve-Path $ProjectDir).Path

$replacementRoot = Join-Path $PatchDir "replacement_files"
$files = @(
    @{ Source = Join-Path $replacementRoot "main.go"; Target = Join-Path $ProjectDir "main.go" },
    @{ Source = Join-Path $replacementRoot "config\config.go"; Target = Join-Path $ProjectDir "config\config.go" },
    @{ Source = Join-Path $replacementRoot "internal\router\router.go"; Target = Join-Path $ProjectDir "internal\router\router.go" },
    @{ Source = Join-Path $replacementRoot "internal\startup\database.go"; Target = Join-Path $ProjectDir "internal\startup\database.go" },
    @{ Source = Join-Path $replacementRoot "internal\startup\startup.go"; Target = Join-Path $ProjectDir "internal\startup\startup.go" },
    @{ Source = Join-Path $replacementRoot "internal\startup\legacy_common_units.go"; Target = Join-Path $ProjectDir "internal\startup\legacy_common_units.go" },
    @{ Source = Join-Path $replacementRoot "internal\startup\legacy_ownership.go"; Target = Join-Path $ProjectDir "internal\startup\legacy_ownership.go" },
    @{ Source = Join-Path $replacementRoot "internal\startup\legacy_reconcile.go"; Target = Join-Path $ProjectDir "internal\startup\legacy_reconcile.go" },
    @{ Source = Join-Path $replacementRoot "internal\middleware\common.go"; Target = Join-Path $ProjectDir "internal\middleware\common.go" },
    @{ Source = Join-Path $replacementRoot "internal\handler\api\api.go"; Target = Join-Path $ProjectDir "internal\handler\api\api.go" },
    @{ Source = Join-Path $replacementRoot "internal\handler\webui\login.go"; Target = Join-Path $ProjectDir "internal\handler\webui\login.go" }
)

foreach ($item in $files) {
    if (!(Test-Path $item.Source)) {
        throw "Patch file missing: $($item.Source)"
    }
}

$goMod = Join-Path $ProjectDir "go.mod"
if (!(Test-Path $goMod)) {
    throw "Not a honoka-chan Go project: missing go.mod"
}
$goModText = Get-Content -Raw $goMod
if ($goModText -notmatch '(?m)^module\s+honoka-chan\s*$') {
    throw "Unsupported project: go.mod module is not honoka-chan"
}

# Direct full-file replacement. This script intentionally creates no backup files.
foreach ($item in $files) {
    $targetDir = Split-Path $item.Target -Parent
    New-Item -ItemType Directory -Force -Path $targetDir | Out-Null
    Copy-Item $item.Source $item.Target -Force
    Write-Host "Replaced $($item.Target)"
}

# Remove additive files from older patch generations. Their logic is already
# integrated into the complete replacement files above.
foreach ($stale in @(
    (Join-Path $ProjectDir "config\android_reload.go"),
    (Join-Path $ProjectDir "internal\router\android.go")
)) {
    if (Test-Path $stale) {
        Remove-Item $stale -Force
        Write-Host "Removed stale patch file: $stale"
    }
}

# Remove the obsolete termux-era config source when present. No backup is made.
$legacyJsonGo = Join-Path $ProjectDir "config\json.go"
if (Test-Path $legacyJsonGo) {
    $legacyText = Get-Content -Raw $legacyJsonGo
    if ($legacyText -match 'SifCdnServer|AsCdnServer|type UserPrefs struct') {
        Remove-Item $legacyJsonGo -Force
        Write-Host "Removed obsolete config/json.go"
    }
}

# Clean backup artifacts created by v17-v21 so they do not pollute Git commits.
$backupPatterns = @(
    "main.go.before_*",
    "config\config.go.before_*",
    "config\json.go.before_*",
    "internal\router\router.go.before_*",
    "internal\startup\database.go.before_*"
)
foreach ($pattern in $backupPatterns) {
    Get-ChildItem -Path (Join-Path $ProjectDir $pattern) -File -ErrorAction SilentlyContinue |
        Remove-Item -Force -ErrorAction SilentlyContinue
}

# Merge the unchanged Android GUI. Existing custom icons and extra resources remain.
$wrapperSource = Join-Path $PatchDir "android-wrapper"
$wrapperTarget = Join-Path $ProjectDir "android-wrapper"
New-Item -ItemType Directory -Force -Path $wrapperTarget | Out-Null
Copy-Item (Join-Path $wrapperSource "*") $wrapperTarget -Recurse -Force
Write-Host "Merged unchanged Android GUI."

& gofmt -w `
    (Join-Path $ProjectDir "main.go") `
    (Join-Path $ProjectDir "config\config.go") `
    (Join-Path $ProjectDir "internal\router\router.go") `
    (Join-Path $ProjectDir "internal\startup\database.go") `
    (Join-Path $ProjectDir "internal\startup\startup.go") `
    (Join-Path $ProjectDir "internal\startup\legacy_common_units.go") `
    (Join-Path $ProjectDir "internal\startup\legacy_ownership.go") `
    (Join-Path $ProjectDir "internal\startup\legacy_reconcile.go") `
    (Join-Path $ProjectDir "internal\middleware\common.go") `
    (Join-Path $ProjectDir "internal\handler\api\api.go") `
    (Join-Path $ProjectDir "internal\handler\webui\login.go")
if ($LASTEXITCODE -ne 0) {
    throw "gofmt failed with exit code $LASTEXITCODE"
}

Write-Host "v30 WebUI-only compatibility patch installed."
Write-Host "No source backups were created."
Write-Host "Next run scripts\build_go_android.ps1, then rebuild the APK."
