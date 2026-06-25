param(
    [Parameter(Mandatory=$true)]
    [string]$ProjectDir
)

$ErrorActionPreference = "Stop"
$PatchDir = Split-Path -Parent (Split-Path -Parent $MyInvocation.MyCommand.Path)

New-Item -ItemType Directory -Force -Path "$ProjectDir\config" | Out-Null
New-Item -ItemType Directory -Force -Path "$ProjectDir\handler" | Out-Null
New-Item -ItemType Directory -Force -Path "$ProjectDir\router" | Out-Null

# Keep one backup of the user's original files.
if ((Test-Path "$ProjectDir\config\json.go") -and !(Test-Path "$ProjectDir\config\json.go.before_android_wrapper")) {
    Copy-Item "$ProjectDir\config\json.go" "$ProjectDir\config\json.go.before_android_wrapper"
}
if ((Test-Path "$ProjectDir\main.go") -and !(Test-Path "$ProjectDir\main.go.before_android_wrapper")) {
    Copy-Item "$ProjectDir\main.go" "$ProjectDir\main.go.before_android_wrapper"
}
if ((Test-Path "$ProjectDir\handler\allstars.go") -and !(Test-Path "$ProjectDir\handler\allstars.go.before_android_wrapper")) {
    Copy-Item "$ProjectDir\handler\allstars.go" "$ProjectDir\handler\allstars.go.before_android_wrapper"
}

# v13: install a known-good config/json.go instead of trying to regex-patch UserPrefs.
# This preserves the original config.json schema and only adds the optional
# settings.unlock_all_special_rotation field for the daily-rotation feature.
Copy-Item "$PatchDir\go_patch\config\json.go" "$ProjectDir\config\json.go" -Force
Copy-Item "$PatchDir\go_patch\config\android_reload.go" "$ProjectDir\config\android_reload.go" -Force
Copy-Item "$PatchDir\go_patch\handler\android_reload.go" "$ProjectDir\handler\android_reload.go" -Force
Copy-Item "$PatchDir\go_patch\router\android.go" "$ProjectDir\router\android.go" -Force
Copy-Item "$PatchDir\go_patch\root\main.go" "$ProjectDir\main.go" -Force

if (Test-Path "$ProjectDir\android-wrapper") {
    Remove-Item "$ProjectDir\android-wrapper" -Recurse -Force
}
Copy-Item "$PatchDir\android-wrapper" "$ProjectDir\android-wrapper" -Recurse -Force

# Backend feature patch: unlock_all_special_rotation.
$allstarsGo = Join-Path $ProjectDir "handler\allstars.go"
if (Test-Path $allstarsGo) {
    $txt = Get-Content -Raw -Path $allstarsGo
    if ($txt -notmatch 'UnlockAllSpecialRotation') {
        $old = "liveDailyList := []model.LiveDaily{}`r`n`terr := MainEng.Table(`"m_live_daily`").Where(`"weekday = ?`", weekday).Cols(`"id,live_id`").Find(&liveDailyList)`r`n`tCheckErr(err)"
        $new = "liveDailyList := []model.LiveDaily{}`r`n`tliveDailyQuery := MainEng.Table(`"m_live_daily`").Cols(`"id,live_id`")`r`n`tif !config.Conf.Settings.UnlockAllSpecialRotation {`r`n`t`tliveDailyQuery = liveDailyQuery.Where(`"weekday = ?`", weekday)`r`n`t}`r`n`terr := liveDailyQuery.OrderBy(`"id ASC`").Find(&liveDailyList)`r`n`tCheckErr(err)"
        if ($txt.Contains($old)) {
            $txt = $txt.Replace($old, $new)
        } else {
            $pattern = 'liveDailyList := \[\]model\.LiveDaily\{\}\s*err := MainEng\.Table\("m_live_daily"\)\.Where\("weekday = \?", weekday\)\.Cols\("id,live_id"\)\.Find\(&liveDailyList\)\s*CheckErr\(err\)'
            $txt = [regex]::Replace($txt, $pattern, $new, 1)
        }
        Set-Content -Encoding UTF8 -Path $allstarsGo -Value $txt
    }
}

try { gofmt -w "$ProjectDir\config\json.go" "$ProjectDir\config\android_reload.go" "$ProjectDir\handler\android_reload.go" "$ProjectDir\router\android.go" "$ProjectDir\main.go" "$ProjectDir\handler\allstars.go" } catch { }

Write-Host "Patch installed. Next run scripts\prepare_server_base_zip.ps1 and scripts\build_go_android.ps1"
