# 配置说明

## 配置方式

项目支持多种配置方式，按优先级从高到低：

1. **显式环境变量**
2. **`.env` 文件**（仅限部分配置）
3. **`config.yaml` 文件**
4. **内置默认值**

## 配置来源矩阵

| 配置项 | 环境变量 | .env | config.yaml | 默认值 |
|--------|----------|------|-------------|--------|
| `server.host` | `E4_SERVER_HOST` | 支持 | `server.host` | `127.0.0.1` |
| `server.port` | `E4_SERVER_PORT` | 支持 | `server.port` | `8080` |
| `server.mode` | `E4_SERVER_MODE` | 支持 | `server.mode` | `development` |
| `database.dsn` | `E4_DATABASE_DSN` | 不支持 | `database.dsn` | `./data/app.db` |
| `auth.username` | `E4_AUTH_USERNAME` | 不支持 | `auth.username` | `admin` |
| `auth.password` | `E4_AUTH_PASSWORD` | 不支持 | `auth.password` | 内置默认 hash |
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

## `.env` 文件

`.env` 文件仅支持以下配置项：
- `E4_SERVER_HOST`
- `E4_SERVER_PORT`
- `E4_SERVER_MODE`
- `E4_AUTH_SECRET`

其他配置项不会从 `.env` 自动读取，必须使用显式环境变量或 `config.yaml`。

## config.yaml 示例

```yaml
server:
  host: 0.0.0.0
  port: 8080
  mode: development

database:
  dsn: ./data/app.db

auth:
  username: admin
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

## 配置说明

### server

| 配置项 | 类型 | 说明 |
|--------|------|------|
| host | string | 监听地址。开发环境默认 `127.0.0.1`，生产环境建议保持 `127.0.0.1` 由反代处理 |
| port | int | 监听端口，默认 `8080` |
| mode | string | 运行模式：`development` 或 `release`。release 模式会强制检查安全配置 |

### database

| 配置项 | 类型 | 说明 |
|--------|------|------|
| dsn | string | SQLite 数据库路径，默认 `./data/app.db` |

### auth

| 配置项 | 类型 | 说明 |
|--------|------|------|
| username | string | 管理员用户名 |
| password | string | bcrypt 密码哈希（不是明文密码） |
| secret | string | 会话签名密钥，用于 HMAC 签名 |
| totp_secret | string | TOTP 二步验证密钥，Base32 编码，为空表示不启用 |
| rate_limit | int | 15 分钟内的登录尝试次数限制，默认 5 |
| lockout_minutes | int | 超限后锁定时间，默认 15 分钟 |

### site

| 配置项 | 类型 | 说明 |
|--------|------|------|
| title | string | 网站标题，显示在页面顶部 |

### json_store

| 配置项 | 类型 | 说明 |
|--------|------|------|
| max_size_bytes | int64 | 单条 JSON 最大字节数，默认 512KB |
| default_ttl_days | int | 默认过期天数，默认 30 |
| max_ttl_days | int | 最大过期天数，默认 90 |
| min_key_length | int | key 最小长度，默认 6 |
| max_key_length | int | key 最大长度，默认 64 |
| max_items | int64 | 最大条目数，默认 1000 |
| max_total_bytes | int64 | 总存储最大字节数，默认 128MB |
| read_rate_limit | int | 读操作频率限制（次/窗口），默认 120 |
| write_rate_limit | int | 写操作频率限制（次/窗口），默认 30 |
| rate_limit_window_seconds | int | 频率限制窗口秒数，默认 60 |

## bcrypt 密码哈希

密码必须使用 bcrypt 哈希，不能直接存储明文。

生成哈希的 Go 示例：
```go
package main

import (
    "fmt"
    "golang.org/x/crypto/bcrypt"
)

func main() {
    hash, _ := bcrypt.GenerateFromPassword([]byte("your-password"), bcrypt.DefaultCost)
    fmt.Println(string(hash))
}
```

默认密码 `admin` 的哈希值：
```
$2a$10$4ZPgUj01QYUd/4feVvRWKebBpHeWiHJQyJABYlTcycO6LiguI.Du2
```

**注意**：生产环境必须修改为独立的强密码哈希。

## Release 模式限制

当 `server.mode=release` 时，以下配置必须修改为非默认值：

1. `auth.username` - 必须设置为非空
2. `auth.password` - 禁止使用内置默认哈希
3. `auth.secret` - 禁止使用 `your-secret-key-change-in-production`

否则应用启动失败。

## 相关文件

| 文件 | 说明 |
|------|------|
| `internal/config/config.go` | 配置加载实现 |
| `.env.example` | 环境变量示例 |
| `config.yaml` | 本地开发配置 |
