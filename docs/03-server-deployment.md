# 服务端部署

## 主程序部署

先确认已经安装 Go 和 Git，然后拉取项目：

```bash
git clone https://github.com/YumeMichi/honoka-chan
cd honoka-chan
```

编译：

```bash
go build
```

编译完成后，Windows 直接运行 `honoka-chan.exe`，Linux 运行 `./honoka-chan`。

## 使用 `run.sh` 管理进程

Linux 下如果希望常驻运行并方便管理，可以直接使用项目根目录的 `run.sh`：

```bash
./run.sh start
./run.sh stop
./run.sh restart
./run.sh status
./run.sh logs
```

当前行为：

- `start` 会以 `nohup` 后台运行 `honoka-chan`
- 如果可执行文件不存在，会自动执行 `go build -o honoka-chan main.go`
- PID 文件路径：`temp/run/honoka-chan.pid`
- 日志文件路径：`temp/run/honoka-chan.log`
- 每次启动都会覆盖旧日志

## 下载服务部署

### SIF 数据下载服务

先准备 [文件准备](./01-prerequisites.md) 里提到的 SIF 全量数据，并解压出：

- `list_CN_Android`
- `list_CN_iOS`

然后在项目根目录下创建：

- `static/Android`
- `static/iOS`

把两个目录分别移动进去，并统一改名为 `archives`。最终目录结构应为：

- `static/Android/archives`
- `static/iOS/archives`

如果只打算支持一个平台，可以删除另一个平台的数据。

### 准备 `99_0_115.zip`

由于数据包中本身也包含盛趣服务器地址，所以为了不反复修改游戏配置文件，可以额外准备一个 `99_0_115.zip`，让它在更新时最后下载，从而覆盖原始地址。

以 iOS 为例：

1. 进入 `static/iOS/archives`
2. 确认当前最后一个 `99` 包是 `99_0_114.zip`
3. 复制 `99_0_113.zip` 为 `99_0_115.zip`
4. 打开压缩包，删除没用的 `client_info.json`
5. 解压出 `server_info.json`
6. 按照 [客户端修改](./02-client-modification.md) 中相同的方法解密、修改、重新加密并放回压缩包

Android 同理。

### 个人分支附加配置

如果你直接使用别人的 CDN 资源，可能会遇到下面几类问题：

- `99_0_115.zip` 需要从单独地址下发
- 部分解压资源需要走和 `cdn_server` 不同的地址
- 第三方 CDN 上的实际文件大小和数据库里的 `pkg_size` 不一致

`personal` 分支里保留了几项用于处理这些情况的附加配置：

```json
{
	"settings": {
		"cdn_server": "http://127.0.0.1:8080/static",
		"backup_cdn_server": "",
		"override_server_config": {
			"enable": false,
			"android": {
				"url": "",
				"size": 0
			},
			"ios": {
				"url": "",
				"size": 0
			}
		},
		"override_file_size": []
	}
}
```

字段说明：

- `cdn_server`：更新包默认下载地址
- `backup_cdn_server`：仅用于 `/download/getUrl` 下发解压资源地址；为空时回退到 `cdn_server`
- `override_server_config`：固定只处理 `99_0_115.zip`
- `override_file_size`：按文件名覆盖返回给客户端的 `size`

`override_server_config` 的行为：

- `enable = false` 时，仍然按默认逻辑从 `cdn_server/{系统}/archives/99_0_115.zip` 探测
- 只有确认该文件可访问时，服务端才会把它加入下载列表
- `enable = true` 且对应平台 `url` 非空时，会优先使用该地址，不再回退到 `cdn_server`
- `size = 0` 时，服务端会自动请求该 URL 探测文件大小
- `size > 0` 时，会直接使用配置值返回给客户端

`override_file_size` 的配置项格式如下：

```json
{
	"settings": {
		"override_file_size": [
			{
				"target_os": "Android",
				"file_name": "99_0_113.zip"
			}
		]
	}
}
```

字段说明：

- `target_os`：可选，留空表示对所有平台生效
- `file_name`：要覆盖大小的包名

命中后，服务端会按该文件最终下发的 URL 自动探测真实大小，并覆盖响应里的 `size`，不会修改数据库原始值。

### 托管方式

如果要把下载服务放到云服务器，只需要把：

- `static/Android`
- `static/iOS`

上传到你的 HTTP 服务目录即可。具体 HTTP 服务搭建方式不在本文档展开。
