# v30 — WebUI login compatibility

This patch is based on v28. It keeps the existing v28 migration and Android-wrapper changes and adds only the WebUI login compatibility fix.

## WebUI fix

- Restores login for Termux-era accounts stored as `" area-account"`.
- Accepts a bare legacy account when it uniquely matches one password-valid row.
- Accepts the complete `area-account` form when multiple legacy accounts need disambiguation.
- Checks session-save errors instead of reporting a false login success.

No source backup files are created.

## Install

PowerShell:

```powershell
.\scripts\install_patch.ps1 -ProjectDir "C:\honoka-chan-dev"
.\scripts\build_go_android.ps1 -ProjectDir "C:\honoka-chan-dev"
```

Linux/macOS:

```bash
./scripts/install_patch.sh /path/to/honoka-chan-dev
./scripts/build_go_android.sh /path/to/honoka-chan-dev
```

No `server-base.zip` rebuild is needed for this Go-only change.

## WebUI login

- New/mainline account: enter the account normally.
- Old Termux account: enter the bare account; when ambiguous, enter `86-account`, `852-account`, etc.
- Mainline default account remains `1` / `klsbgames`.
