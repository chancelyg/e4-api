# JSON 存储模块

## 功能概述

JSON 存储模块提供匿名的临时数据存取接口，支持按 key 的 GET/POST/PUT/DELETE 操作，带 TTL 过期管理。

### 主要特性

- **匿名访问**：无需认证即可使用公开接口
- **按 key 操作**：key 仅允许字母和数字
- **TTL 过期**：支持自定义过期时间，默认 30 天，最大 90 天
- **容量限制**：单条最大 512KB，总存储最大 128MB，最多 1000 条
- **频率限制**：读/写分开限流
- **管理接口**：登录后可分页查看所有 JSON 元信息

## 数据模型

### JSONStoreItem

| 字段 | 类型 | 说明 |
|------|------|------|
| id | uint | 主键 |
| key | string | 唯一标识（6-64 字符，仅字母数字） |
| content | string | JSON 内容 |
| size_bytes | int64 | 内容大小（字节） |
| expires_at | time | 过期时间 |
| created_at | time | 创建时间 |
| updated_at | time | 更新时间 |

## 公开接口

### 创建 JSON（POST）

`POST /api/json/:key`

若 key 已存在且未过期，返回 409 Conflict。

**请求**：
```bash
curl -X POST http://localhost:8080/api/json/MyKey123 \
  -H 'Content-Type: application/json' \
  --data-binary '{"hello":"world","value":42}'
```

**查询参数**：
- `ttl_days`：过期天数（可选，默认 30，最大 90）

**响应**（成功）：
```json
{
  "key": "MyKey123",
  "size_bytes": 26,
  "expires_at": "2026-05-06T10:30:00Z",
  "created_at": "2026-04-06T10:30:00Z",
  "updated_at": "2026-04-06T10:30:00Z"
}
```

**响应**（key 已存在）：
```json
{
  "error": "key 已存在"
}
```

### 读取 JSON（GET）

`GET /api/json/:key`

**请求**：
```bash
curl http://localhost:8080/api/json/MyKey123
```

**响应**：返回原始 JSON 数据（Content-Type: application/json）

**响应**（不存在或已过期）：
```json
{
  "error": "JSON 不存在或已过期"
}
```

### 创建或覆盖 JSON（PUT）

`PUT /api/json/:key`

与 POST 的区别：key 存在时会覆盖，并刷新过期时间。

**请求**：
```bash
curl -X PUT http://localhost:8080/api/json/MyKey123?ttl_days=45 \
  -H 'Content-Type: application/json' \
  --data-binary '{"updated":true}'
```

### 删除 JSON（DELETE）

`DELETE /api/json/:key`

```bash
curl -X DELETE http://localhost:8080/api/json/MyKey123
```

**响应**：`204 No Content`

## 管理接口（需认证）

### 列出所有 JSON

`GET /api/admin/json`

**查询参数**：

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| page | int | 1 | 页码 |
| per_page | int | 20 | 每页数量（最大 100） |
| search | string | - | 搜索关键词（支持 id、key、content） |
| sort | string | desc | 排序：asc/desc（按 updated_at） |

```bash
curl "http://localhost:8080/api/admin/json?page=1&per_page=20&sort=desc" \
  -b 'e4_session=your-session-cookie'
```

**响应**：
```json
{
  "items": [
    {
      "id": 1,
      "key": "MyKey123",
      "size_bytes": 26,
      "expires_at": "2026-05-06T10:30:00Z",
      "created_at": "2026-04-06T10:30:00Z",
      "updated_at": "2026-04-06T10:30:00Z",
      "is_expired": false
    }
  ],
  "total": 100,
  "total_size_bytes": 2600,
  "newest_key": "MyKey123",
  "sort_order": "desc"
}
```

### 读取 JSON 完整内容

`GET /api/admin/json/:key/content`

```bash
curl http://localhost:8080/api/admin/json/MyKey123/content \
  -b 'e4_session=your-session-cookie'
```

**响应**：
```json
{
  "key": "MyKey123",
  "content": "{\"hello\":\"world\",\"value\":42}",
  "size_bytes": 26,
  "expires_at": "2026-05-06T10:30:00Z",
  "created_at": "2026-04-06T10:30:00Z",
  "updated_at": "2026-04-06T10:30:00Z",
  "is_expired": false
}
```

### 删除 JSON

`DELETE /api/admin/json/:key`

```bash
curl -X DELETE http://localhost:8080/api/admin/json/MyKey123 \
  -b 'e4_session=your-session-cookie'
```

## 配置参数

在 `config.yaml` 中配置：

```yaml
json_store:
  max_size_bytes: 524288        # 单条最大 512KB
  default_ttl_days: 30           # 默认过期 30 天
  max_ttl_days: 90              # 最大过期 90 天
  min_key_length: 6              # key 最小长度
  max_key_length: 64             # key 最大长度
  max_items: 1000                # 最大条目数
  max_total_bytes: 134217728     # 总存储最大 128MB
  read_rate_limit: 120            # 读频率限制（次/窗口）
  write_rate_limit: 30            # 写频率限制（次/窗口）
  rate_limit_window_seconds: 60  # 频率限制窗口（秒）
```

## 频率限制

- **读操作**：默认 120 次/分钟
- **写操作**：默认 30 次/分钟
- 超出限制返回 `429 Too Many Requests`

## 使用场景

- 临时数据存储（如表单草稿）
- 跨会话数据共享
- 简单的配置存储
- 公开数据接口

## 前端实现

前端路由：`web/src/routes/json/+page.svelte`

管理界面提供：
- JSON 列表浏览
- 内容查看和复制
- 批量删除

## 相关文件

| 文件 | 说明 |
|------|------|
| `internal/handlers/json_store.go` | JSON 存储处理器实现 |
| `internal/models/json_store.go` | JSON 存储数据模型 |
| `web/src/routes/json/+page.svelte` | JSON 管理页面组件 |
| `web/src/lib/api.ts` | API 调用封装（publicJSONAPI、adminJSONAPI） |
