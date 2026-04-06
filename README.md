# E4 API

一个以个人日记为核心的轻量应用，后端使用 Go + Echo，前端使用 Svelte 5。
当前形态是单用户日记系统，同时已经为后续对外 API 扩展预留了清晰的服务边界。

## 项目概览

- Go 后端提供 JSON API，并在生产构建时嵌入前端静态资源
- Svelte 前端负责登录、日记列表、详情、编辑、统计等交互页面
- 数据默认存储在 SQLite，适合单机部署与快速迭代
- 当前认证基于签名 Session Cookie，登出吊销状态持久化到 SQLite，支持可选 TOTP 二步验证
- 后续可以逐步扩展为独立 API 服务或增加外部集成模块

## 当前功能

- 用户登录、登出、登录状态查询
- 可选二步验证登录流程
- 日记创建、编辑、删除、详情查看
- 日记分页、关键词搜索、按月份筛选
- 日记统计：总篇数、最长连续记录、时间跨度、记录频率
- 匿名 JSON 临时存取接口，支持按 key 的 GET/POST/PUT/DELETE
- 登录后台可分页查看 JSON 元信息，支持复制内容与删除
- 单二进制部署：前端资源嵌入 Go 可执行文件

## 技术栈

### 后端

- Go 1.25+
- Echo
- GORM
- SQLite
- Viper
- bcrypt
- TOTP

### 前端

- Svelte 5
- TypeScript
- Vite
- 纯 CSS

## 目录结构

```text
.
├── main.go                    # 应用入口，路由、CORS、静态资源回退
├── internal/
│   ├── config/                # 配置加载与默认值
│   ├── db/                    # 数据库初始化
│   ├── handlers/              # HTTP 处理器
│   ├── middleware/            # 认证中间件
│   └── models/                # 数据模型
├── pkg/
│   ├── embed.go               # 嵌入 pkg/dist
│   └── dist/                  # 生产前端资源（生成文件）
├── web/
│   ├── src/                   # Svelte 源码
│   ├── dist/                  # 前端构建结果（生成文件）
│   └── .svelte-kit/           # SvelteKit 生成文件
├── data/                      # 运行期 SQLite 数据
├── deploy/                    # systemd / Nginx / 部署参考
├── build.sh                   # 生产构建脚本
├── dev.sh                     # 本地开发脚本
├── config.yaml                # 本地开发配置
└── .env.example               # 环境变量示例
```

## 快速开始

### 1. 准备环境

需要先安装：

- Go 1.25+
- Node.js / npm（需能运行 SvelteKit 2 / Vite 7）

### 2. 启动开发环境

```bash
./dev.sh
```

或者分别启动：

```bash
# 终端 1
cd web
npm install
npm run dev

# 终端 2
go run main.go
```

默认访问地址：

- 前端开发服务：`http://localhost:5173`
- 后端服务：`http://localhost:8080`

## 构建与运行

### 生产构建

```bash
./build.sh
```

或使用 GoReleaser：

```bash
goreleaser release --snapshot --clean
```

构建流程：

1. 构建 `web/dist`
2. 将前端产物复制到 `pkg/dist`
3. 编译 Go 二进制 `e4-api`

### 运行

```bash
./e4-api
```

查看部署时可用参数与环境变量：

```bash
./e4-api --help
```

也可以通过环境变量覆盖端口：

```bash
E4_SERVER_PORT=3000 ./e4-api
```

### 使用 GoReleaser 打包

仓库根目录已经提供 `.goreleaser.yaml`，适合单机部署场景下生成多平台压缩包。

```bash
goreleaser release --snapshot --clean
```

默认行为：

- 先执行 `scripts/prepare-dist.sh` 构建前端并刷新 `pkg/dist`
- 再编译 `linux` / `darwin` / `windows` / `freebsd` 的常见架构二进制
- 输出校验文件 `checksums.txt`
- 自动创建 GitHub Release 并上传构建产物

常见发布平台包括：

- Linux: `amd64`, `arm64`, `386`, `armv6`, `armv7`
- macOS: `amd64`, `arm64`
- Windows: `amd64`, `arm64`, `386`
- FreeBSD: `amd64`, `arm64`, `386`

如果只想本地验证配置，可以继续使用 snapshot 模式；正式发版时再基于 git tag 执行 `goreleaser release`。

正式发布流程：

1. 创建并推送版本 tag，例如 `v0.1.1`
2. GitHub Actions 自动触发 GoReleaser
3. 产物上传到对应 GitHub Release 页面

如果你在本机手动发布：

```bash
goreleaser release --clean
```

## 配置说明

项目支持通过 `config.yaml`、`.env` 和环境变量配置。

优先级如下：

1. 显式环境变量
2. `.env`
3. `config.yaml`
4. 内置默认值

### 当前推荐分工

- `config.yaml`：数据库、管理员用户名、bcrypt 密码哈希、TOTP 配置、限流参数、站点标题
- `.env`：服务监听参数和会话签名密钥
- 显式环境变量：可以覆盖所有配置项，适合 systemd、Docker、CI/CD 注入

### `config.yaml` 配置示例

```yaml
server:
  host: 0.0.0.0
  port: 8080
  mode: development

database:
  dsn: ./data/app.db

auth:
  username: admin
  # bcrypt hash for the password "admin"
  password: "$2a$10$5OUxfHLfhWa1sYDlpuarQevoiPznWTmM1OZjLS.vtlbj7zsW6gMvG"
  totp_secret: ""
  rate_limit: 5
  lockout_minutes: 15

site:
  title: E4 Diary

json_store:
  max_size_bytes: 524288
  default_ttl_days: 30
  max_ttl_days: 90
  min_key_length: 6
  max_key_length: 64
  max_items: 1000
  max_total_bytes: 134217728
  read_rate_limit: 120
  write_rate_limit: 30
  rate_limit_window_seconds: 60
```

### 配置来源矩阵

| 配置项 | 显式环境变量 | `.env` | `config.yaml` 写法 | 默认值 |
|------|------|------|------|------|
| `server.host` | `E4_SERVER_HOST` | 支持 | `server.host` | `127.0.0.1` |
| `server.port` | `E4_SERVER_PORT` | 支持 | `server.port` | `8080` |
| `server.mode` | `E4_SERVER_MODE` | 支持 | `server.mode` | `development` |
| `database.dsn` | `E4_DATABASE_DSN` | 不支持 | `database.dsn` | `./data/app.db` |
| `auth.username` | `E4_AUTH_USERNAME` | 不支持 | `auth.username` | `admin` |
| `auth.password` | `E4_AUTH_PASSWORD` | 不支持 | `auth.password` | 内置默认 admin 的 bcrypt hash |
| `auth.secret` | `E4_AUTH_SECRET` | 支持 | `auth.secret` | `your-secret-key-change-in-production` |
| `auth.totp_secret` | `E4_AUTH_TOTP_SECRET` | 不支持 | `auth.totp_secret` | `""` |
| `auth.rate_limit` | `E4_AUTH_RATE_LIMIT` | 不支持 | `auth.rate_limit` | `5` |
| `auth.lockout_minutes` | `E4_AUTH_LOCKOUT_MINUTES` | 不支持 | `auth.lockout_minutes` | `15` |
| `site.title` | `E4_SITE_TITLE` | 不支持 | `site.title` | `E4 Diary` |
| `json_store.max_size_bytes` | `E4_JSON_STORE_MAX_SIZE_BYTES` | 不支持 | `json_store.max_size_bytes` | `524288` |
| `json_store.default_ttl_days` | `E4_JSON_STORE_DEFAULT_TTL_DAYS` | 不支持 | `json_store.default_ttl_days` | `30` |
| `json_store.max_ttl_days` | `E4_JSON_STORE_MAX_TTL_DAYS` | 不支持 | `json_store.max_ttl_days` | `90` |
| `json_store.min_key_length` | `E4_JSON_STORE_MIN_KEY_LENGTH` | 不支持 | `json_store.min_key_length` | `6` |
| `json_store.max_key_length` | `E4_JSON_STORE_MAX_KEY_LENGTH` | 不支持 | `json_store.max_key_length` | `64` |
| `json_store.max_items` | `E4_JSON_STORE_MAX_ITEMS` | 不支持 | `json_store.max_items` | `1000` |
| `json_store.max_total_bytes` | `E4_JSON_STORE_MAX_TOTAL_BYTES` | 不支持 | `json_store.max_total_bytes` | `134217728` |
| `json_store.read_rate_limit` | `E4_JSON_STORE_READ_RATE_LIMIT` | 不支持 | `json_store.read_rate_limit` | `120` |
| `json_store.write_rate_limit` | `E4_JSON_STORE_WRITE_RATE_LIMIT` | 不支持 | `json_store.write_rate_limit` | `30` |
| `json_store.rate_limit_window_seconds` | `E4_JSON_STORE_RATE_LIMIT_WINDOW_SECONDS` | 不支持 | `json_store.rate_limit_window_seconds` | `60` |

说明：

- `.env` 只会自动加载 `server.host`、`server.port`、`server.mode`、`auth.secret`
- 其余项虽然支持“显式环境变量覆盖”，但不会从 `.env` 自动吸收
- `config.yaml` 中的键路径使用表格里的 `server.host` / `auth.password` 这种层级写法

### `.env` 示例

参考 `.env.example`：

- `E4_SERVER_HOST`
- `E4_SERVER_PORT`
- `E4_SERVER_MODE`
- `E4_AUTH_SECRET`

## 安全说明

当前仓库保留了开发默认配置，便于本地快速启动，但生产环境必须覆盖：

- `auth.username`
- `auth.password`
- `auth.secret`

同时建议部署时保持：

- `server.host=127.0.0.1`，仅让 Nginx/Caddy 在本机反代；如容器内部监听可改为 `0.0.0.0`
- `server.mode=release`
- HTTPS 终止在反向代理层

项目现在会在 `release` 模式下拒绝使用默认管理员凭据和默认 secret。

详细部署参考见 `deploy/README.md`。

## TOTP 工具

仓库提供了 `scripts/generate-totp.sh`，用于生成：

- Base32 TOTP 密钥
- `otpauth://` URI 字符串
- 文本二维码
- SVG 二维码

示例：

```bash
E4_TOTP_ISSUER="E4 Diary" E4_TOTP_ACCOUNT="admin" ./scripts/generate-totp.sh
```

可选输出目录参数：

```bash
./scripts/generate-totp.sh ./out/my-totp
```

脚本优先使用 `qrencode`，如果系统未安装，则回退到 `npx qrcode`。

## 会话持久化说明

- 登录态本身保存在签名 Cookie 中，因此刷新页面和服务重启后仍可继续使用
- 登出后的吊销记录会写入 SQLite 的 `session_revocations` 表
- 这意味着单机部署下，即使进程重启，已登出的会话也不会重新生效
- 过期吊销记录会在后续鉴权或登出时自动清理

## API 概览

### 认证接口

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/auth/login` | 单步登录；如开启 2FA，返回挑战令牌 |
| POST | `/api/auth/login-step1` | 密码校验，返回 2FA 挑战令牌 |
| POST | `/api/auth/login-step2` | 提交验证码与挑战令牌完成登录 |
| POST | `/api/auth/logout` | 登出 |
| GET | `/api/auth/status` | 登录状态 |

### 日记接口

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/diary` | 获取日记列表 |
| POST | `/api/diary` | 创建日记 |
| GET | `/api/diary/:id` | 获取单篇日记 |
| PUT | `/api/diary/:id` | 更新日记 |
| DELETE | `/api/diary/:id` | 删除日记 |
| GET | `/api/diary/stats` | 获取统计信息 |

### 日记列表查询参数

- `page`: 页码，默认 `1`
- `per_page`: 每页数量，默认 `20`
- `search`: 搜索关键词，支持空格分词组合搜索
- `start_date`: 开始日期，格式 `YYYY-MM-DD`
- `end_date`: 结束日期，格式 `YYYY-MM-DD`

### 其他接口

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/ip` | 返回客户端 IP 与 User-Agent |

### 匿名 JSON 接口

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/json/:key` | 创建 JSON，已存在则返回冲突 |
| GET | `/api/json/:key` | 读取原始 JSON |
| PUT | `/api/json/:key` | 创建或覆盖 JSON，并刷新过期时间 |
| DELETE | `/api/json/:key` | 删除 JSON |

规则：

- `key` 仅允许字母和数字，长度默认 `6-64`
- 请求体直接提交原始 JSON，不包额外字段
- 单条 JSON 默认最大 `512 KiB`
- `ttl_days` 可选，不传默认 `30`，最大默认 `90`
- 公开接口不提供列表能力

示例：

```bash
curl -X POST \
  -H 'Content-Type: application/json' \
  --data-binary '{"hello":"world"}' \
  'http://localhost:8080/api/json/Abc123'

curl 'http://localhost:8080/api/json/Abc123'

curl -X PUT \
  -H 'Content-Type: application/json' \
  --data-binary @data.json \
  'http://localhost:8080/api/json/Abc123?ttl_days=45'

curl -X DELETE 'http://localhost:8080/api/json/Abc123'
```

## 测试与检查

### 后端

```bash
go test ./...
```

### 前端

```bash
cd web
npm run check
```

### 构建验证

```bash
./build.sh
```

## API 扩展模块规划

项目后续会增加对外 API 扩展模块，建议按下面的方向演进。

### 建议的扩展结构

```text
internal/
├── handlers/
│   ├── auth.go
│   ├── diary.go
│   └── api/                  # 对外 API handler
├── service/                  # 业务服务层
├── repository/               # 数据访问层
└── models/
```

### 推荐演进路径

1. 把日记业务从 handler 中抽到 service 层，降低 HTTP 与业务耦合
2. 为未来多用户能力引入真实 `User` 模型与 `user_id` 归属字段
3. 将当前 session-only 认证与对外 API 认证分离
4. 为外部 API 单独规划版本前缀，例如 `/api/v1/...`
5. 为扩展模块补充更稳定的响应结构、错误码与接口文档
6. 如果未来需要第三方调用，考虑引入 token 或 API key 方案，而不是直接复用浏览器 session

### 当前已完成的铺垫

- 路由已经统一挂在 `/api`
- `internal/handlers` 已按职责拆分
- 前后端通信已通过 `web/src/lib/api.ts` 集中管理
- 认证、日记、通用能力已经具备独立拆分基础

## 已知限制

- 当前仍是单用户模型
- 会话保存在签名 Cookie 中，重启后仍可保持登录，登出吊销记录也会持久化到 SQLite
- SQLite 适合轻量单机场景，不适合高并发多实例部署
- 统计逻辑仍偏向当前规模，未来需要进一步下沉到数据库层优化

## 仓库初始化说明

当前仓库已初始化 Git，可以直接开始提交与分支管理。
建议首次提交前先确认本地环境已安装 Go / Node，并确保生成文件未误提交。

## 开发建议

- 不要手改 `web/dist`
- 不要手改 `pkg/dist`
- 前端变更后应重新构建再嵌入
- 后端接口字段保持 snake_case，避免前后端契约漂移

## License

MIT
