# file-transfer-client

`file-transfer-client` 是 `file-transfer` 的桌面客户端实现，使用 `Wails + Vue 3 + TypeScript + Go` 构建。

## 当前能力

- 连接 `file-transfer` 服务端并调用 `/healthz`
- 浏览远程目录 JSON 接口
- 图形化查看文件详情
- 选择本地保存位置并发起下载
- 查看下载任务状态、失败信息和完成后打开本地目录
- 保存最近连接地址和默认下载目录

## 项目结构

```text
.
|-- main.go                     # Wails 入口
|-- app.go                      # 桌面端绑定接口
|-- internal/transfer/          # file-transfer HTTP client
|-- internal/downloads/         # 下载任务队列
|-- internal/settings/          # 本地配置持久化
|-- internal/model/             # 前后端共享数据结构
|-- frontend/                   # Vue GUI
`-- wails.json
```

## 本地开发

先准备服务端，例如在同级 `file-transfer` 仓库中启动：

```bash
go run ./cmd/file-transfer -root ./shared -addr :8080
```

然后在当前仓库启动桌面客户端开发模式：

```bash
cd frontend
npm install
npm run build
cd ..
wails dev
```

如果只想验证 Go 侧逻辑：

```bash
go test ./internal/...
```

## 说明

- `frontend/dist/index.html` 现在是一个占位构建产物，只用于让 Go 的嵌入资源在未执行前端构建时也能编译。
- 正式界面以 `frontend/src/` 中的 Vue 页面为准。
