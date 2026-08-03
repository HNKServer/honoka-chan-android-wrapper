# APK Server 修补与构建方法

本项目使用 Vibe Coding 方式生成，使用方法为依次执行 Patch_And_Build_Scripts 下的脚本，修补原始 Golang 代码程序，然后将其生成为完整的 Android 项目（即 android-wrapper），再将其编译为 Android 可执行文件，最后使用 Android Studio 构建 APK

本项目所包含的源码已经完成了 Golang 程序的修补与构建，并且也已经基于它生成了完整的 Android 项目和 APK 安装包，因此不再需要按照重复修补和构建，上述步骤仅供参考
