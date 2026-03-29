# 部署参考

本目录提供适合单机公网部署的最小参考文件。

## 目录内容

- `e4-api.service`: systemd 服务单元示例
- `nginx.conf.example`: Nginx 反向代理示例

## 配置分层建议

- `config.yaml`: 提交到仓库的非敏感默认配置，适合本地开发和通用默认值
- `.env`: 不提交到仓库的部署配置，放账号、密码哈希、会话密钥、数据库路径等敏感项
- 显式环境变量: 优先级最高，适合 CI/CD 或容器平台注入

当前加载优先级:

1. 显式环境变量
2. `.env`
3. `config.yaml`
4. 内置默认值

## 推荐的公网部署方式

### 方式一：systemd + Nginx

适合常规 Linux 云主机。

1. 创建运行用户

```bash
sudo useradd --system --home /opt/e4-api --shell /usr/sbin/nologin e4
```

2. 创建目录并放置二进制

```bash
sudo mkdir -p /opt/e4-api/data
sudo chown -R e4:e4 /opt/e4-api
```

3. 在 `/opt/e4-api/.env` 写入部署配置

参考根目录 `.env.example`，至少要设置：

- `E4_SERVER_MODE=release`
- `E4_SERVER_HOST=127.0.0.1`
- `E4_AUTH_USERNAME`
- `E4_AUTH_PASSWORD`
- `E4_AUTH_SECRET`

4. 安装 systemd 服务

```bash
sudo cp deploy/e4-api.service /etc/systemd/system/e4-api.service
sudo systemctl daemon-reload
sudo systemctl enable --now e4-api
sudo systemctl status e4-api
```

5. 配置 Nginx 反代

```bash
sudo cp deploy/nginx.conf.example /etc/nginx/sites-available/e4-api
sudo ln -s /etc/nginx/sites-available/e4-api /etc/nginx/sites-enabled/e4-api
sudo nginx -t
sudo systemctl reload nginx
```

6. 开启 HTTPS

建议使用 Let's Encrypt 或你现有的证书方案。

### 方式二：纯二进制 + 进程管理

如果不用 systemd，也至少保证：

- 应用只监听 `127.0.0.1`
- 前面有 Nginx/Caddy 反向代理
- `.env` 权限为 `600`
- 二进制目录和 `data/` 目录只对部署用户可写
- 有日志轮转和自动重启机制

### 方式三：容器部署

如果使用 Docker/Podman，建议：

- 挂载 `data/` 持久化 SQLite
- 用环境变量或 secret 注入敏感项，不要 bake 进镜像
- 只暴露给反代层，避免直接公网映射应用端口
- 继续保持 `E4_SERVER_HOST=0.0.0.0` 仅在容器内确有需要时使用

## 公网部署安全检查清单

- 不要提交真实 `.env`
- 不要在 `config.yaml` 中写生产凭据
- 生产环境必须使用独立用户名、bcrypt 密码哈希、随机会话密钥
- 建议启用 TOTP 二步验证
- Nginx 只转发到 `127.0.0.1:8080`
- 服务器防火墙只开放 `80/443`，不要直接开放应用端口
- 定期备份 `data/app.db`
- 通过 `journalctl -u e4-api` 和 Nginx 日志观察异常登录

## 生成 bcrypt 密码哈希

可以用 Go 快速生成：

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
