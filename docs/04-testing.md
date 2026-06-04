# 运行测试

前面的步骤完成后，就可以开始联调。

## 配置文件检查

第一次启动 `honoka-chan` 后会生成 `config.json`。需要确认：

- `settings.cdn_server` 指向你的数据下载地址
- 如果需要给 `/download/getUrl` 单独指定解压资源地址，可以额外设置 `settings.backup_cdn_server`
- `settings.unlock_all_special_rotation` 可选；设为 `true` 后会忽略日替时间限制，直接解锁全部日替特殊歌曲，不包含周替 `MASTER`
- 如果数据直接放在本项目的 `static` 目录下，通常可以配置成类似 `http://192.168.1.123/static`

如果 `cdn_server/{系统}/archives/99_0_115.zip` 存在，服务端会自动把它追加到更新下载列表；如果不存在，则会直接跳过。

如果启用了 `settings.override_server_config`：

- 服务端会改用你配置的平台专用 URL 下发 `99_0_115.zip`
- `size = 0` 时会自动探测远端文件大小
- 如果该 URL 请求失败，则该文件不会被下发，也不会再回退到 `cdn_server`

如果启用了 `settings.override_file_size`，命中的文件会按最终下发 URL 重新探测大小，并覆盖响应中的 `size` 字段。

## 登录与下载测试

客户端登录方式为 `手机号密码登录`。这里的手机号和密码都可以随便填写：

- 第一次使用的手机号会自动创建账号
- 之后再次使用同一个手机号时，会沿用之前保存的密码和游戏内设置

进入游戏后，按提示下载数据，确认下载过程正常完成。

下载完成后即可进入游戏。

## 补充说明

SIF 的“全量数据下载”实际上并不是真正的全量。部分剧情资源仍然需要通过 `/download/getUrl` 接口按路径拉取。

所以除了 `archives` 目录里的压缩包，还需要另外维护一份：

- zip 全部解压后的资源目录

这部分内容后面再补充。
