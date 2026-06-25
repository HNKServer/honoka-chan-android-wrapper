# Honoka Android Wrapper Patch v16

基于 v15，继续修两个 UI 小问题：

1. 修复 `Ui.kt` 里 `insetLeft` / `insetRight` 在当前 Material Components 版本下无法解析的问题。
   - 已删除这两个属性。
   - 改用 `setPaddingRelative(0, 0, 0, 0)` 控制小三角按钮内边距。

2. 修复主界面和 config.json 编辑界面的 “Fightだよ！” 显示/隐藏按钮不同步的问题。
   - `MainActivity.onResume()` 会重新读取 `show_fight_title` 偏好。
   - 如果发现 config 页面改过显示状态，返回主界面时会自动重建 UI。
   - 避免出现从 config 页切换后主界面需要连续点两次按钮才正常的问题。

使用方式：

```powershell
cd <v16_patch目录>
.\scripts\install_patch.ps1 -ProjectDir "C:\honoka-chan-termux"
.\scripts\prepare_server_base_zip.ps1 -ProjectDir "C:\honoka-chan-termux"
.\scripts\build_go_android.ps1 -ProjectDir "C:\honoka-chan-termux"
```

然后在 Android Studio 里：

```text
Sync Now
Build > Clean Project
Build > Rebuild Project
```

这版只改 Android UI 壳，不改 Go 后端功能逻辑。
