# 部署指南

## 部署方式概览

| 方式 | 适用场景 | 特点 |
|------|----------|------|
| systemd + Nginx | 常规 Linux 云主机 | 稳定、成熟 |
| 纯二进制 + 进程管理 | 轻量部署 | 简单灵活 |
| Docker/Podman | 容器化环境 | 隔离性好 |

## 方式一：systemd + Nginx

适合常规 Linux 云主机。

### 1. 创建运行用户

```bash
sudo useradd --system --home /opt/e4-api --shell /usr/sbin/nologin e4
```

### 2. 创建目录并放置二进制

```bash
sudo mkdir -p /opt/e4-api/data
sudo chown -R e4:e4 /opt/e4-api
```

### 3. 配置 .env

在 `/opt/e4-api/.env` 中写入：

```bash
E4_SERVER_MODE=release
E4_SERVER_HOST=127.0.0.1
E4_AUTH_SECRET=your-random-secret-here
```

### 4. 配置 config.yaml

在 `/opt/e4-api/config.yaml` 中写入：

```yaml
database:
  dsn: ./data/app.db

auth:
  username: admin
  password: "<your-bcrypt-hash>"
  totp_secret: ""
  rate_limit: 5
  lockout_minutes: 15

site:
  title: E4 Diary
```

### 5. 安装 systemd 服务

```bash
sudo cp deploy/e4-api.service /etc/systemd/system/e4-api.service
sudo systemctl daemon-reload
sudo systemctl enable --now e4-api
sudo systemctl status e4-api
```

### 6. 配置 Nginx 反向代理

```bash
sudo cp deploy/nginx.conf.example /etc/nginx/sites-available/e4-api
sudo ln -s /etc/nginx/sites-available/e4-api /etc/nginx/sites-enabled/e4-api
sudo nginx -t
sudo systemctl reload nginx
```

### 7. 开启 HTTPS

建议使用 Let's Encrypt 或现有证书方案。

## 方式二：纯二进制 + 进程管理

如果不用 systemd：

- 应用只监听 `127.0.0.1`
- 前面有 Nginx/Caddy 反向代理
- `.env` 权限为 `600`
- 二进制目录和 `data/` 目录只对部署用户可写
- 有日志轮转和自动重启机制

## 方式三：容器部署

Docker/Podman 部署建议：

- 挂载 `data/` 持久化 SQLite
- 用环境变量或 secret 注入敏感项
- 只暴露给反代层，避免直接公网映射应用端口
- 继续保持 `E4_SERVER_HOST=0.0.0.0` 仅在容器内确有需要时使用

### 示例 Docker Compose

```yaml
version: '3.8'
services:
  e4-api:
    image: e4-api:latest
    restart: unless-stopped
    ports:
      - "127.0.0.1:8080:8080"
    volumes:
      - ./data:/app/data
    env_file:
      - .env
    environment:
      - E4_SERVER_MODE=release
      - E4_SERVER_HOST=0.0.0.0
```

## 安全检查清单

- [ ] 不要提交真实 `.env`
- [ ] `config.yaml` 中的 bcrypt 密码哈希不要提交回仓库
- [ ] 生产环境使用独立用户名、强密码哈希、随机会话密钥
- [ ] 建议启用 TOTP 二步验证
- [ ] Nginx 只转发到 `127.0.0.1:8080`
- [ ] 服务器防火墙只开放 `80/443`，不直接开放应用端口
- [ ] 定期备份 `data/app.db`
- [ ] 通过 `journalctl -u e4-api` 和 Nginx 日志观察异常登录

## 生成 bcrypt 密码哈希

```bash
cat <<'EOF' >/tmp/hash.go
package main

import (
    "fmt"
    "golang.org/x/crypto/bcrypt"
)

func main() {
    hash, _ := bcrypt.GenerateFromPassword([]byte("your-strong-password"), bcrypt.DefaultCost)
    fmt.Println(string(hash))
}
EOF
go run /tmp/hash.go
rm /tmp/hash.go
```

## TOTP 二步验证

### 生成 TOTP 密钥

```bash
E4_TOTP_ISSUER="E4 Diary" E4_TOTP_ACCOUNT="admin" ./scripts/generate-totp.sh
```

输出文件：
- `out/totp/base32.txt` - Base32 密钥
- `out/totp/otpauth-uri.txt` - otpauth URI
- `out/totp/otpauth-uri.txt.qrcode.txt` - 文本二维码
- `out/totp/otpauth-uri.svg` - SVG 二维码

### 配置 TOTP

将 Base32 密钥填入 `config.yaml`：

```yaml
auth:
  totp_secret: "JBSWY3DPEHPK3PXP"
```

### 配置验证器

使用 Google Authenticator、Authy 等扫描 SVG 二维码或手动输入 Base32 密钥。

## 备份与恢复

### 备份

```bash
cp data/app.db data/app.db.backup.$(date +%Y%m%d)
```

### 恢复

```bash
cp data/app.db.backup.20260406 data/app.db
```

## 日志管理

systemd 日志：
```bash
journalctl -u e4-api -f
```

## 相关文件

| 文件 | 说明 |
|------|------|
| `deploy/e4-api.service` | systemd 服务单元 |
| `deploy/nginx.conf.example` | Nginx 反向代理配置 |
| `scripts/generate-totp.sh` | TOTP 密钥生成脚本 |
