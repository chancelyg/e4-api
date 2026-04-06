# 日记模块

## 功能概述

日记模块提供日记的创建、读取、删除和统计功能。

### 主要特性

- 按日期自动递增创建日期（接续最新日记）
- 关键词搜索（支持分词和模糊匹配）
- 按时间查看日记（月度视图）
- 删除前二次确认，避免误删
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

自动使用“下一日”逻辑：若已有最新日记，新日记日期自动设为最新一篇的下一天。

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
| sort | string | - | 排序方向，`asc` 表示从远到近，`desc` 表示从近到远 |

**说明**：

- 当传入 `search` 时，后端执行关键词模糊搜索，不要求同时传日期范围
- 当传入 `start_date` 和 `end_date` 时，可以实现按月查看
- 当传入搜索关键词或指定日期范围时，可以额外传入 `sort` 控制排序方向
- 前端当前只保留两种筛选模式：关键词搜索，或按时间查看；两者不会叠加

**示例**：
```bash
curl "http://localhost:8080/api/diary?page=1&per_page=50&search=关键词" \
  -b 'e4_session=your-session-cookie'
```

```bash
curl "http://localhost:8080/api/diary?page=1&per_page=50&search=关键词&sort=desc" \
  -b 'e4_session=your-session-cookie'
```

```bash
curl "http://localhost:8080/api/diary?page=1&per_page=50&start_date=2026-04-01&end_date=2026-04-30&sort=desc" \
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

### 删除日记

`DELETE /api/diary/:id`

```bash
curl -X DELETE http://localhost:8080/api/diary/1 \
  -b 'e4_session=your-session-cookie'
```

**响应**：

- 成功时返回 `204 No Content`
- 日记不存在时返回 `404`
- 日记 ID 非法时返回 `400`

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

## 搜索与浏览规则

### 关键词搜索

1. 按空格分词
2. 每个词同时支持两类匹配：
3. 前缀后缀模糊：`%keyword%`
4. 逐字模糊：`%k%e%y%w%o%r%d%`
5. 搜索结果默认按日期从远到近排列，可切换为从近到远

示例：搜索“今天 完成”时，需要同时匹配“今天”和“完成”，并支持单字分散匹配。

### 按时间查看

- 日记页默认展示“本月日记”
- 月初如果本月还没有日记，列表为空是正常表现
- 顶部时间选择区支持前后一年、前后一个月的步进选择
- 调整时间后需要点击“确认查看”才会切换列表，避免误触导致频繁刷新
- 切换月份后，页面展示所选月份内的全部日记，并按日期倒序排列
- 切换到时间视图时会清空关键词搜索
- 在月度列表底部也可以直接切换到上个月或下个月
- 指定日期范围的列表默认按日期从远到近排列，可切换为从近到远

## 前端实现

前端路由：`web/src/routes/diary/+page.svelte`

主要功能：

- 快速写日记（自动日期接续）
- 默认展示本月日记
- 关键词搜索
- 搜索/指定日期结果的排序切换
- 年份/月度步进选择与确认查看
- 列表底部上月/下月导航
- 删除确认弹窗
- 移动端优化后的搜索区布局

状态管理：`web/src/lib/stores.svelte.ts`

## 相关文件

| 文件 | 说明 |
|------|------|
| `internal/handlers/diary.go` | 日记处理器实现 |
| `internal/handlers/diary_test.go` | 日记处理器测试 |
| `internal/models/diary.go` | 日记数据模型 |
| `web/src/routes/diary/+page.svelte` | 日记页面组件 |
| `web/src/lib/api.ts` | API 调用封装 |
