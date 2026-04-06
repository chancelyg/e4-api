# 日记模块

## 功能概述

日记模块提供日记的创建、读取、更新、删除和统计功能。

### 主要特性

- 按日期自动递增创建日期（接续最新日记）
- 关键词搜索（支持分词和模糊匹配）
- 月份筛选
- 分页浏览
- 写日记统计：总篇数、最长连续记录、时间跨度、记录频率

## 数据模型

### Diary

| 字段 | 类型 | 说明 |
|------|------|------|
| id | uint | 主键 |
| content | string | 日记内容（text） |
| create_date | string | 创建日期（YYYY-MM-DD） |
| created_at | time | 创建时间 |
| updated_at | time | 更新时间 |

## API 接口

### 创建日记

`POST /api/diary`

自动使用"下一日"逻辑：若今天已有日记，新日记日期自动设为明天。

**请求**：
```bash
curl -X POST http://localhost:8080/api/diary \
  -H 'Content-Type: application/json' \
  -b 'e4_session=your-session-cookie' \
  -d '{"content":"今天完成了很多事情..."}'
```

**响应**：
```json
{
  "id": 1,
  "content": "今天完成了很多事情...",
  "create_date": "2026-04-06",
  "created_at": "2026-04-06T10:30:00Z",
  "updated_at": "2026-04-06T10:30:00Z"
}
```

### 获取日记列表

`GET /api/diary`

**查询参数**：

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| page | int | 1 | 页码 |
| per_page | int | 20 | 每页数量（最大 100） |
| search | string | - | 搜索关键词 |
| start_date | string | - | 开始日期（YYYY-MM-DD） |
| end_date | string | - | 结束日期（YYYY-MM-DD） |

**示例**：
```bash
curl "http://localhost:8080/api/diary?page=1&per_page=20&search=关键词&start_date=2026-01-01&end_date=2026-12-31" \
  -b 'e4_session=your-session-cookie'
```

**响应**：
```json
{
  "diaries": [
    {
      "id": 1,
      "content": "今天完成了很多事情...",
      "create_date": "2026-04-06",
      "created_at": "2026-04-06T10:30:00Z",
      "updated_at": "2026-04-06T10:30:00Z"
    }
  ],
  "total": 100
}
```

### 获取单篇日记

`GET /api/diary/:id`

```bash
curl http://localhost:8080/api/diary/1 \
  -b 'e4_session=your-session-cookie'
```

**响应**：
```json
{
  "id": 1,
  "content": "今天完成了很多事情...",
  "create_date": "2026-04-06",
  "created_at": "2026-04-06T10:30:00Z",
  "updated_at": "2026-04-06T10:30:00Z"
}
```

### 更新日记

`PUT /api/diary/:id`

**请求**：
```bash
curl -X PUT http://localhost:8080/api/diary/1 \
  -H 'Content-Type: application/json' \
  -b 'e4_session=your-session-cookie' \
  -d '{"content":"更新后的日记内容..."}'
```

### 删除日记

`DELETE /api/diary/:id`

```bash
curl -X DELETE http://localhost:8080/api/diary/1 \
  -b 'e4_session=your-session-cookie'
```

### 获取统计信息

`GET /api/diary/stats`

```bash
curl http://localhost:8080/api/diary/stats \
  -b 'e4_session=your-session-cookie'
```

**响应**：
```json
{
  "total_count": 365,
  "max_consecutive_days": 30,
  "start_date": "2025-01-01",
  "end_date": "2026-04-06",
  "time_span_days": 461
}
```

**统计字段说明**：

| 字段 | 说明 |
|------|------|
| total_count | 日记总篇数 |
| max_consecutive_days | 最长连续记录天数 |
| start_date | 第一篇日记日期 |
| end_date | 最后一篇日记日期 |
| time_span_days | 时间跨度（天） |

## 搜索功能

### 搜索算法

1. **分词**：按空格分割关键词
2. **模糊匹配**：每个词同时支持：
   - 前缀后缀模糊：`%keyword%`
   - 逐字模糊：`%k%w%o%r%d%`

### 示例

搜索"今天 完成"：
- 匹配包含"今天"和"完成"的日记
- 同时支持单字分散匹配

## 前端实现

前端路由：`web/src/routes/diary/+page.svelte`

主要功能：
- 快速写日记（自动日期接续）
- 月份回看导航
- 关键词搜索
- 分页浏览

状态管理：`web/src/lib/stores.svelte.ts`

## 相关文件

| 文件 | 说明 |
|------|------|
| `internal/handlers/diary.go` | 日记处理器实现 |
| `internal/models/diary.go` | 日记数据模型 |
| `web/src/routes/diary/+page.svelte` | 日记页面组件 |
| `web/src/lib/api.ts` | API 调用封装 |
