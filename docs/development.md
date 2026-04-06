# 开发指南

## 环境准备

### 依赖要求

- **Go**: 1.25+
- **Node.js / npm**: 需兼容 SvelteKit 2 / Vite 7

## 本地开发

### 全栈开发（推荐）

使用 `dev.sh` 脚本同时启动前端和后端：

```bash
./dev.sh
```

启动后：
- 前端开发服务：`http://localhost:5173`
- 后端服务：`http://localhost:8080`

### 分别启动

**终端 1 - 前端**：
```bash
cd web
npm install
npm run dev
```

**终端 2 - 后端**：
```bash
go run main.go
```

## 生产构建

### 完整构建

```bash
./build.sh
```

构建流程：
1. 构建前端 `web/dist`
2. 复制前端产物到 `pkg/dist`
3. 编译 Go 二进制 `e4-api`

### 仅构建前端

```bash
cd web
npm install
npm run build
```

### 仅构建后端

确保 `pkg/dist` 已存在后：

```bash
go build -ldflags="-s -w" -o e4-api main.go
```

## 测试与检查

### 后端测试

```bash
go test ./...
```

运行单个包测试：
```bash
go test ./internal/handlers -v
```

运行单个测试：
```bash
go test ./internal/handlers -run TestLoginSuccess -v
```

运行子测试：
```bash
go test ./internal/handlers -run 'TestCalculateStats/single_diary' -v
```

### 前端类型检查

```bash
cd web
npm run check
```

监听模式：
```bash
npm run check:watch
```

### Go 格式化检查

```bash
gofmt -w <files>
```

### Go vet 检查

```bash
go vet ./...
```

## 项目结构

### 后端结构

```
internal/
├── config/           # 配置加载
│   └── config.go    # Viper 配置管理
├── db/              # 数据库初始化
│   └── db.go       # GORM + SQLite
├── handlers/        # HTTP 处理器
│   ├── auth.go     # 认证：登录、登出、状态
│   ├── diary.go    # 日记：CRUD、统计
│   ├── goals.go    # 目标：CRUD、打卡、统计
│   ├── json_store.go # JSON 存储
│   └── common.go   # 通用：IP 查询
├── middleware/      # 中间件
│   └── auth.go     # 会话验证
└── models/          # 数据模型
    ├── diary.go    # Diary 模型
    ├── goal.go     # Goal, GoalRecord 模型
    ├── json_store.go # JSONStoreItem 模型
    └── session.go  # SessionRevocation 模型
```

### 前端结构

```
web/
├── src/
│   ├── lib/
│   │   ├── api.ts           # API 调用封装
│   │   ├── stores.svelte.ts # 认证状态管理
│   │   ├── date.ts          # 日期工具函数
│   │   └── routes/          # 共享路由组件
│   ├── routes/              # 页面路由
│   │   ├── +page.svelte    # 登录页
│   │   ├── diary/           # 日记页面
│   │   ├── goals/           # 目标页面
│   │   ├── json/            # JSON 管理页面
│   │   └── stats/           # 统计页面
│   ├── app.css              # 全局样式
│   └── app.html             # HTML 模板
├── dist/                    # 构建产物（生成）
├── package.json
└── svelte.config.js
```

## API 调用封装

前端 API 调用统一在 `web/src/lib/api.ts` 中：

```typescript
import { authAPI, diaryAPI, goalsAPI, publicJSONAPI, adminJSONAPI } from '$lib/api';

// 认证
await authAPI.login({ username, password });
await authAPI.logout();
await authAPI.status();

// 日记
await diaryAPI.list({ page: 1, per_page: 20, search: '关键词' });
await diaryAPI.get(1);
await diaryAPI.create({ content: '新日记' });
await diaryAPI.stats();

// 目标
await goalsAPI.list();
await goalsAPI.create({ name: '新目标', annual_target: 100 });
await goalsAPI.dashboard({ range: 'year' });
await goalsAPI.upsertRecord(1, '2026-04-06', { quantity: 5 });

// JSON 存储（公开）
await publicJSONAPI.get('MyKey');
await publicJSONAPI.create('MyKey', '{"test":true}');

// JSON 存储（管理）
await adminJSONAPI.list({ page: 1 });
await adminJSONAPI.delete('MyKey');
```

## 生成 bcrypt 密码哈希

```bash
cat <<'EOF' >/tmp/hash.go
package main

import (
    "fmt"
    "golang.org/x/crypto/bcrypt"
)

func main() {
    hash, _ := bcrypt.GenerateFromPassword([]byte("your-password"), bcrypt.DefaultCost)
    fmt.Println(string(hash))
}
EOF
go run /tmp/hash.go
```

## 生成 TOTP 密钥

```bash
E4_TOTP_ISSUER="E4 Diary" E4_TOTP_ACCOUNT="admin" ./scripts/generate-totp.sh
```

## 注意事项

- 不要手改 `web/dist` 或 `pkg/dist`，这些是生成文件
- 前端变更后应重新构建再嵌入
- 后端接口字段保持 snake_case，避免前后端契约漂移
- JSON 字段名使用 snake_case：`create_date`、`is_logged_in`、`max_consecutive_days`

## 命令速查

| 命令 | 说明 |
|------|------|
| `./dev.sh` | 全栈开发 |
| `go run main.go` | 后端单独运行 |
| `cd web && npm run dev` | 前端单独运行 |
| `./build.sh` | 完整生产构建 |
| `go test ./...` | 运行所有测试 |
| `cd web && npm run check` | 前端类型检查 |
