# E4 API 文档

本文档详细描述 E4 API 的各个功能模块、开发指南和使用说明。

## 文档索引

### 模块文档

- [认证模块](auth.md) - 用户登录、TOTP 二步验证、会话管理
- [日记模块](diary.md) - 日记 CRUD、搜索、统计
- [目标模块](goals.md) - 目标创建、打卡记录、年度统计
- [JSON 存储模块](json-store.md) - 匿名 JSON 存取接口
- [配置说明](config.md) - 配置文件、环境变量、参数详解
- [部署指南](deployment.md) - systemd、Nginx、容器部署
- [开发指南](development.md) - 本地开发、测试、构建

## 项目概述

E4 API 是一个以个人日记为核心的轻量应用，同时包含目标打卡和临时数据存储功能。

### 核心功能

| 模块 | 说明 |
|------|------|
| 日记 | 每日记录、关键词搜索、月份筛选、统计概览 |
| 目标 | 目标创建与软删除、每日打卡、年度统计 |
| JSON 存储 | 匿名临时数据存取、带 TTL 过期管理 |
| 认证 | 用户密码登录、TOTP 二步验证、会话持久化 |

### 技术架构

- **后端**：Go + Echo + GORM + SQLite
- **前端**：Svelte 5 + TypeScript + Vite
- **部署**：单二进制 + 嵌入式前端资源

### 目录结构

```
e4-api/
├── main.go              # 应用入口，路由配置
├── internal/
│   ├── config/          # 配置加载（Viper）
│   ├── db/              # 数据库初始化（GORM + SQLite）
│   ├── handlers/        # HTTP 处理器
│   │   ├── auth.go     # 认证相关
│   │   ├── diary.go    # 日记相关
│   │   ├── goals.go    # 目标打卡相关
│   │   ├── json_store.go # JSON 存储相关
│   │   └── common.go   # 通用接口
│   ├── middleware/      # 认证中间件
│   └── models/          # 数据模型
├── pkg/
│   └── embed.go        # 前端资源嵌入
├── web/                 # Svelte 前端源码
│   └── src/
│       ├── lib/
│       │   ├── api.ts  # API 调用封装
│       │   └── stores.svelte.ts # 状态管理
│       └── routes/     # 页面路由
├── scripts/             # 工具脚本
│   └── generate-totp.sh # TOTP 密钥生成
└── deploy/             # 部署配置示例
    ├── e4-api.service  # systemd 服务
    └── nginx.conf.example # Nginx 反向代理
```

## API 概览

所有 API 基础路径为 `/api`。

### 认证接口（公开）

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/auth/login` | 单步登录（自动判断是否需要 2FA） |
| POST | `/api/auth/login-step1` | 分步登录第一步 |
| POST | `/api/auth/login-step2` | 分步登录第二步（提交 TOTP） |
| POST | `/api/auth/logout` | 登出 |
| GET | `/api/auth/status` | 登录状态查询 |

### 日记接口（需认证）

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/diary` | 获取日记列表 |
| POST | `/api/diary` | 创建日记 |
| GET | `/api/diary/:id` | 获取单篇日记 |
| PUT | `/api/diary/:id` | 更新日记 |
| DELETE | `/api/diary/:id` | 删除日记 |
| GET | `/api/diary/stats` | 获取统计信息 |

### 目标接口（需认证）

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/goals` | 获取目标列表 |
| POST | `/api/goals` | 创建目标 |
| GET | `/api/goals/dashboard` | 获取目标面板数据 |
| GET | `/api/goals/year-summary` | 获取年度统计 |
| PUT | `/api/goals/:id` | 更新目标 |
| DELETE | `/api/goals/:id` | 软删除目标 |
| PUT | `/api/goals/:id/records/:date` | 打卡（创建/更新） |
| DELETE | `/api/goals/:id/records/:date` | 删除打卡记录 |

### JSON 存储接口

| 方法 | 路径 | 说明 | 认证 |
|------|------|------|------|
| POST | `/api/json/:key` | 创建 JSON | 公开 |
| GET | `/api/json/:key` | 读取 JSON | 公开 |
| PUT | `/api/json/:key` | 创建或覆盖 JSON | 公开 |
| DELETE | `/api/json/:key` | 删除 JSON | 公开 |
| GET | `/api/admin/json` | 列出所有 JSON（分页） | 需认证 |
| GET | `/api/admin/json/:key/content` | 读取 JSON 完整内容 | 需认证 |
| DELETE | `/api/admin/json/:key` | 删除 JSON | 需认证 |

### 通用接口（需认证）

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/ip` | 获取客户端 IP 和 User-Agent |
