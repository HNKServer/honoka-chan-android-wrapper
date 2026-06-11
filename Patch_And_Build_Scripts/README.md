# Honoka Android Wrapper Patch v10

这是把 `honoka-chan-termux` 套成独立 Android APK 的最新补丁包。

v10 包含目前已验证的修复：

- Material 3 / Material You 风格 UI。
- AndroidX / Gradle JVM target 配置。
- APP 内置真实路径选择器，不再用 `content://` 目录选择器糊弄 Go 子进程。
- 首页运行状态卡片：显示 STARTING / RUNNING / FAILED、健康检查、运行目录、archives 目录、错误详情。
- ForegroundService + watchdog + WakeLock，用于后台保活与 Go 进程异常重启。
- `config.json` 结构保持原样，不新增字段，不破坏客户端兼容性。
- APP 内可直接编辑原始 `config.json`，并通过 `/__android/config/reload` 热加载到 Go 内存。
- `server-base.zip` 解压时兼容 Windows 反斜杠路径，例如 `assets\main.db` 会被正确解压成 `assets/main.db`。
- 不再使用 `Process.isAlive`，兼容 `minSdk 23`。
- Go stdout/stderr 会写入 Logcat 和 UI 状态详情，方便定位 panic。

## 目录结构

```text
honoka_android_wrapper_patch_v10/
├── android-wrapper/          # 完整 Android Studio 工程
├── go_patch/                 # Go 端最小补丁：热加载接口
├── scripts/                  # 安装补丁、生成资源包、编译 Go Android 二进制
└── README.md
```

## 1. 应用补丁

PowerShell：

```powershell
cd <补丁包解压目录>
.\scripts\install_patch.ps1 -ProjectDir "C:\honoka-chan-termux"
```

Linux/macOS/Git Bash：

```bash
cd <补丁包解压目录>
./scripts/install_patch.sh /path/to/honoka-chan-termux
```

> 注意：`install_patch` 会替换原项目的 `android-wrapper` 目录，并把原 `main.go` 备份为 `main.go.before_android_wrapper`。

## 2. 生成 APK 内置基础资源包

这个包只放小型基础资源，不放十几 GB 的 `static/Android/archives`。

```powershell
.\scripts\prepare_server_base_zip.ps1 -ProjectDir "C:\honoka-chan-termux"
```

生成位置：

```text
C:\honoka-chan-termux\android-wrapper\app\src\main\assets\server-base.zip
```

检查里面是否包含基础数据库：

```powershell
tar -tf "C:\honoka-chan-termux\android-wrapper\app\src\main\assets\server-base.zip" | findstr /i "assets/main.db assets/data.example.db"
```

如果没有输出，说明 `-ProjectDir` 指错了。

## 3. 编译 Go Android 二进制

```powershell
.\scripts\build_go_android.ps1 -ProjectDir "C:\honoka-chan-termux"
```

生成位置：

```text
C:\honoka-chan-termux\android-wrapper\app\src\main\jniLibs\arm64-v8a\libhonoka.so
```

它虽然叫 `.so`，本质是 Go 交叉编译出来的 Android arm64 可执行文件。

## 4. 构建 APK

Android Studio 打开：

```text
C:\honoka-chan-termux\android-wrapper
```

然后：

```text
Sync Now
Build > Clean Project
Build > Rebuild Project
```

或命令行：

```powershell
cd "C:\honoka-chan-termux\android-wrapper"
.\gradlew.bat clean
.\gradlew.bat assembleDebug
```

APK 输出：

```text
C:\honoka-chan-termux\android-wrapper\app\build\outputs\apk\debug\app-debug.apk
```

## 5. 安装前建议清旧数据

因为旧版本可能留下半残的 `files/server` 目录或旧 symlink，调试时建议：

```powershell
adb uninstall moe.honoka.wrapper
adb install "C:\honoka-chan-termux\android-wrapper\app\build\outputs\apk\debug\app-debug.apk"
```

或：

```powershell
adb shell pm clear moe.honoka.wrapper
adb install -r "C:\honoka-chan-termux\android-wrapper\app\build\outputs\apk\debug\app-debug.apk"
```

## 6. 手机端使用流程

1. 打开 APP。
2. 点“授予全部文件访问权限”。
3. 点“选择 archives 目录”。
4. 选择真正包含 zip 数据包的目录。
5. 点“启动服务端”。
6. 查看首页“运行状态”卡片。
7. 打开 `http://127.0.0.1:8080/admin/index`。

大包推荐放在：

```text
/storage/emulated/0/Download/HonokaData/static/Android/archives
```

但 APP 允许你选择其他公共目录。

## 7. config.json 热加载

APP 的配置页面直接编辑原始 `config.json`，不会新增字段，不会改变结构。

可热加载的通常是：

```text
settings.sif_cdn_server
settings.as_cdn_server
user_prefs.*
```

`settings.server_port` 保存后需要重启 Go 服务端才会改变监听端口。

## 8. 故障定位

如果状态卡片显示 FAILED，先看卡片里的错误详情。也可以抓 Logcat：

```powershell
adb logcat -c
# 手机里点启动
adb logcat -d -s HonokaServer
```

常见问题：

- `Go binary not found`：没有执行 `build_go_android.ps1`。
- `APK assets 中没有 server-base.zip`：没有执行 `prepare_server_base_zip.ps1` 或没有重新构建 APK。
- `基础资源缺失 assets/main.db`：旧 APP 数据没清，或 `server-base.zip` 解压失败；先 `adb shell pm clear moe.honoka.wrapper`。
- `archives directory not found`：选择的 archives 目录不存在或选错层级。
- `Android 11+ 尚未授予全部文件访问权限`：回 APP 点授权按钮。
- `bind: address already in use`：8080 已被占用，改 `config.json` 后重启服务端。
