# honoka-chan-android-server

LoveLive! 学园偶像祭、学园偶像季: 群星闪耀 自用私服。

原本项目的服务端功能已经比较完善了，但是其为 Android 设备支持而设置的 Termux 分支运行起来却略显麻烦，而该项目本身并不深度依赖 Termux 环境且其使用的 Golang 是可跨平台移植的，因此本项目以便捷为最终目的，将原本的项目源码进行微小改动后，套壳移植成了 APK 安装包，使其能够在 Android 平台上开箱即用；

核心功能有以下三点：
 - 在GUI中加入了可以直接手动选取存储服务端数据包的 Android 本地路径并映射给 Golang 程序的选择界面
 - 在GUI中加入了一键启动、停止和重启HTTP服务的控制界面以及服务运行的状态指示界面
 - 在GUI中加入了一键修改config.json配置文件的文本编辑界面

我主要做的就是三件事，很惭愧，只做了一点微小的工作

## 其他所需文件

[MEGA网盘](https://mega.nz/folder/X7JB3bwI#L9eLbOQsCLMkSK0TO3_hfw)

## 特别感谢

 - YumeMichi 的 原始项目 [honoka-chan](https://github.com/YumeMichi/honoka-chan/tree/termux)
