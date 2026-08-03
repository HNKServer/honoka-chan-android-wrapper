# v28 mainline API signature fix

This release fixes the v27 `internal/handler/api/api.go` calls so they exactly match the uploaded mainline dev core signatures (`BannerApi(action)`, `LiveSeApi(action)`, `SubscenarioApi(ctx, action)`, etc.). Android GUI and the v27 migration/reconciliation logic are unchanged.

# Honoka Android Wrapper v28 — source-reconciled mainline migration

This patch keeps the Android GUI unchanged and updates only the Go compatibility layer for an in-place upgrade from the working Termux core to the current mainline core.

## Why this patch exists

The JNI branch does not implement a different SIF protocol. Its JNI layer only starts and stops the same Gin router in-process. The important difference is that a fresh JNI installation creates a self-consistent mainline User DB, while an upgraded Termux installation keeps ownership IDs and table rows created by the old core.

The working Termux core returned cards from `common_unit_m` plus `user_unit_m`. Its shared `unit_owning_user_id` values started at 38383 in the natural row order of `common_unit_m`. The current mainline creates `user_unit_data` from `unit_m ORDER BY unit_id`. Consequently, an old numeric ownership ID can point to a different card after upgrade. Old decks, centre member, accessory wear and removable-skill equipment continue referencing the old IDs. The client then receives internally inconsistent `unitAll` / `deckInfo` / linked data in the `/main.php/api` batch and may abort in native `libGame.so` even though the HTTP response is 200.

## v28 changes

The patch directly replaces complete Go files; it does not use regex source edits and creates no source backup files.

It preserves all current mainline modules and features, including achievements and the native `unlock_all_special_rotation` setting, while adding an idempotent legacy reconciliation layer:

- keeps the exact old `common_unit_m` ownership order for translation only;
- ensures every current `common_unit_data` card has a corresponding `user_unit_data` row;
- imports duplicate/custom rows from old `user_unit_m` into `user_unit_data`;
- translates old ownership IDs used by `user_preference_m`, `deck_unit_m`, `skill_equip_m` and `accessory_wear_m`;
- repairs centre-member references;
- repairs or rebuilds malformed decks so all nine positions reference cards present in `unitAll`;
- retains the current mainline card database and all newly implemented handlers;
- restores the Android/Termux item-level status-600 fallback only for unimplemented items in `/main.php/api`, instead of aborting the entire batch;
- keeps the Android GUI exactly as in the current wrapper.

## Install

```powershell
cd <extracted-v28-directory>
.\scripts\install_patch.ps1 -ProjectDir "C:\honoka-chan-dev"
.\scripts\build_go_android.ps1 -ProjectDir "C:\honoka-chan-dev"
```

Then open `C:\honoka-chan-dev\android-wrapper` in Android Studio and run Clean/Rebuild.

Install over the existing app with the same signing key. Do not uninstall the app and do not clear app data, because the reconciliation needs the existing User DB and old Termux tables.

The install script creates no `.before_*` backups and cleans backup artifacts left by older patch generations.
