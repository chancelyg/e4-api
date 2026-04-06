# 认证模块

## 功能概述

认证模块提供用户身份验证、会话管理和可选的 TOTP 二步验证功能。

### 主要特性

- 用户名/密码认证（bcrypt 哈希验证）
- 可选 TOTP 二步验证
- 基于签名 Cookie 的会话管理
- 会话吊销列表持久化到 SQLite
- 登录频率限制（防止暴力破解）
- 账户锁定机制

## 登录流程

### 单步登录（`POST /api/auth/login`）

适合未启用 TOTP 或希望简化流程的场景：

```bash
curl -X POST http://localhost:8080/api/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"admin","password":"your-password"}'
```

**响应示例**：

未启用 TOTP 时直接登录成功：
```json
{
  "success": true,
  "username": "admin"
}
```

启用 TOTP 后返回挑战令牌：
```json
{
  "needs_2fa": true,
  "challenge_token": "a1b2c3d4e5f6..."
}
```

### 分步登录

分步登录提供更精细的流程控制，适合自定义前端 UI。

#### 第一步：密码验证（`POST /api/auth/login-step1`）

```bash
curl -X POST http://localhost:8080/api/auth/login-step1 \
  -H 'Content-Type: application/json' \
  -d '{"username":"admin","password":"your-password"}'
```

响应格式同单步登录。

#### 第二步：TOTP 验证（`POST /api/auth/login-step2`）

```bash
curl -X POST http://localhost:8080/api/auth/login-step2 \
  -H 'Content-Type: application/json' \
  -d '{"code":"123456","challenge_token":"a1b2c3d4e5f6..."}'
```

**响应**：
```json
{
  "success": true,
  "username": "admin"
}
```

## 会话管理

### 会话结构

会话 Cookie 名称：`e4_session`

会话 payload 包含：
- `session_id`：会话唯一标识
- `username`：用户名
- `expires_at`：Unix 时间戳

会话通过 HMAC-SHA256 签名防篡改。

### 会话持久化

- 会话本身存储在签名 Cookie 中，刷新页面或重启服务后依然有效
- 登出时将 session_id 写入 SQLite 的 `session_revocations` 表
- 已登出的会话即使 Cookie 未过期也无法使用
- 过期的吊销记录在后续鉴权时自动清理

### 会话有效期

默认会话有效期为 **7 天**（`7 * 24 * 60 * 60` 秒）。

## 登出

```bash
curl -X POST http://localhost:8080/api/auth/logout \
  -b 'e4_session=your-session-cookie'
```

**响应**：
```json
{
  "success": true
}
```

## 登录状态

```bash
curl http://localhost:8080/api/auth/status \
  -b 'e4_session=your-session-cookie'
```

**响应**：

已登录：
```json
{
  "is_logged_in": true,
  "username": "admin"
}
```

未登录：
```json
{
  "is_logged_in": false
}
```

## TOTP 二步验证

### 配置

在 `config.yaml` 中设置 `auth.totp_secret`：
```yaml
auth:
  totp_secret: "JBSWY3DPEHPK3PXP"
```

### 生成 TOTP 密钥

使用项目提供的脚本生成：
```bash
E4_TOTP_ISSUER="E4 Diary" E4_TOTP_ACCOUNT="admin" ./scripts/generate-totp.sh
```

输出包括：
- Base32 编码的密钥
- `otpauth://` URI
- 文本二维码和 SVG 二维码

### 验证器配置

将生成的 `otpauth://` URI 或二维码导入验证器（如 Google Authenticator、Authy）。

## 安全机制

### 登录频率限制

- 窗口期：15 分钟
- 允许尝试次数：默认 5 次（可配置）
- 超限后锁定：默认 15 分钟（可配置）

### Release 模式限制

当 `server.mode=release` 时：
- 必须设置非空的管理员用户名
- 禁止使用默认的 bcrypt 密码哈希
- 禁止使用默认的会话签名密钥

## 相关文件

| 文件 | 说明 |
|------|------|
| `internal/handlers/auth.go` | 认证处理器实现 |
| `internal/middleware/auth.go` | 会话验证中间件 |
| `internal/models/session.go` | 会话吊销模型 |
| `web/src/lib/stores.svelte.ts` | 前端认证状态管理 |
