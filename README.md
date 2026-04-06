# E4 API

一个以个人日记为核心的轻量应用，后端使用 Go + Echo，前端使用 Svelte 5。

## 项目概览

- Go 后端提供 JSON API，并在生产构建时嵌入前端静态资源
- Svelte 前端负责登录、日记、目标、打卡、统计等交互页面
- 数据默认存储在 SQLite，适合单机部署与快速迭代
- 认证基于签名 Session Cookie，登出吊销状态持久化到 SQLite，支持可选 TOTP 二步验证

## 技术栈

### 后端

- Go 1.25+ / Echo / GORM / SQLite / Viper / bcrypt / TOTP

### 前端

- Svelte 5 / TypeScript / Vite / 纯 CSS

## 目录结构

```
e4-api/
├── main.go                    # 应用入口，路由、CORS、静态资源回退
├── internal/
│   ├── config/                # 配置加载与默认值
│   ├── db/                    # 数据库初始化
│   ├── handlers/              # HTTP 处理器（auth、diary、goals、json_store、common）
│   ├── middleware/            # 认证中间件
│   └── models/                # 数据模型
├── pkg/
│   └── embed.go               # 嵌入 pkg/dist
├── web/
│   └── src/                   # Svelte 源码
├── scripts/                   # 工具脚本
├── deploy/                    # systemd / Nginx 部署配置
└── docs/                      # 详细文档
```

## 快速开始

### 1. 准备环境

- Go 1.25+
- Node.js / npm（兼容 SvelteKit 2 / Vite 7）

### 2. 启动开发环境

```bash
./dev.sh
```

默认访问地址：
- 前端开发服务：`http://localhost:5173`
- 后端服务：`http://localhost:8080`

## 构建与运行

### 生产构建

```bash
./build.sh
```

### 运行

```bash
./e4-api
```

查看可用参数：
```bash
./e4-api --help
```

## 详细文档

各模块的详细说明、功能介绍、API 参考见 [docs/](docs/) 目录：

- [docs/auth.md](docs/auth.md) - 认证模块（登录、TOTP、会话管理）
- [docs/diary.md](docs/diary.md) - 日记模块（CRUD、搜索、统计）
- [docs/goals.md](docs/goals.md) - 目标模块（目标、打卡、年度统计）
- [docs/json-store.md](docs/json-store.md) - JSON 存储模块
- [docs/config.md](docs/config.md) - 配置说明
- [docs/deployment.md](docs/deployment.md) - 部署指南
- [docs/development.md](docs/development.md) - 开发指南

## 测试与检查

```bash
go test ./...          # 后端测试
cd web && npm run check  # 前端类型检查
```

## License

MIT
