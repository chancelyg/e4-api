# 目标模块

## 功能概述

目标模块提供目标管理和每日打卡功能，支持年度目标、周目标和数量型目标。

### 主要特性

- **目标类型**：checkbox（完成/未完成）、quantity（数量型）
- **软删除**：删除目标时保留历史打卡记录
- **重新启用**：可恢复已删除的目标
- **年度目标**：设置年度完成次数/数量目标
- **周目标**：设置每周完成次数目标
- **打卡记录**：支持按日期补签（仅限今年且不晚于今天）
- **年度统计**：查看往年打卡数据

## 数据模型

### Goal

| 字段 | 类型 | 说明 |
|------|------|------|
| id | uint | 主键 |
| name | string | 目标名称 |
| description | string | 备注说明 |
| goal_type | string | 类型：`checkbox` / `quantity` |
| unit | string | 单位（如 km、页、次） |
| annual_target | *float64 | 年度目标 |
| weekly_target | *int | 每周目标次数 |
| is_active | bool | 是否启用 |
| sort_order | int | 排序顺序 |
| created_at | time | 创建时间 |
| updated_at | time | 更新时间 |

### GoalRecord

| 字段 | 类型 | 说明 |
|------|------|------|
| id | uint | 主键 |
| goal_id | uint | 关联目标 ID |
| record_date | string | 记录日期（YYYY-MM-DD） |
| is_completed | bool | 是否完成 |
| quantity | *float64 | 数量（quantity 型目标） |
| created_at | time | 创建时间 |
| updated_at | time | 更新时间 |

## API 接口

### 获取目标列表

`GET /api/goals`

返回所有启用状态的目标列表。

```bash
curl http://localhost:8080/api/goals \
  -b 'e4_session=your-session-cookie'
```

**响应**：
```json
{
  "goals": [
    {
      "id": 1,
      "name": "跑步",
      "description": "晨跑或夜跑",
      "goal_type": "quantity",
      "unit": "km",
      "annual_target": 500,
      "weekly_target": 3,
      "is_active": true,
      "sort_order": 1
    }
  ]
}
```

### 创建目标

`POST /api/goals`

**请求体**：
```json
{
  "name": "读书",
  "description": "每天阅读",
  "unit": "页",
  "annual_target": 1000,
  "weekly_target": 5
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| name | string | 是 | 目标名称 |
| description | string | 否 | 备注说明 |
| unit | string | 否 | 单位，填写后打卡需输入数量 |
| annual_target | float | 否 | 年度目标 |
| weekly_target | int | 否 | 每周目标次数 |
| reactivate_id | uint | 否 | 重新启用已删除目标的 ID |

**响应**：
```json
{
  "id": 2,
  "name": "读书",
  "description": "每天阅读",
  "goal_type": "checkbox",
  "unit": "页",
  "annual_target": 1000,
  "weekly_target": 5,
  "is_active": true,
  "sort_order": 2
}
```

### 更新目标

`PUT /api/goals/:id`

支持部分更新，只传需要修改的字段。

### 删除目标（软删除）

`DELETE /api/goals/:id`

软删除：设置 `is_active = false`，历史打卡记录保留。

```bash
curl -X DELETE http://localhost:8080/api/goals/1 \
  -b 'e4_session=your-session-cookie'
```

### 目标面板

`GET /api/goals/dashboard`

获取目标面板的完整数据，包括打卡状态、进度、日历等。

**查询参数**：

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| date | string | 今天 | 锚定日期（YYYY-MM-DD） |
| range | string | year | 统计范围：week/month/quarter/year/all |
| checkin_date | string | 昨天 | 打卡日期（YYYY-MM-DD） |
| month | string | 当前月 | 日历月份（YYYY-MM） |

```bash
curl "http://localhost:8080/api/goals/dashboard?range=year&date=2026-04-06" \
  -b 'e4_session=your-session-cookie'
```

**响应**：
```json
{
  "anchor_date": "2026-04-06",
  "range": "year",
  "range_start_date": "2026-01-01",
  "range_end_date": "2026-12-31",
  "checkin_date": "2026-04-05",
  "today_completed_count": 2,
  "annual_checkin_total": 156,
  "calendar_month": "2026-04",
  "goals": [
    {
      "id": 1,
      "name": "跑步",
      "annual_target": 500,
      "annual_quantity_total": 320,
      "annual_progress_percent": 64,
      "current_week_completed_count": 2,
      "current_week_progress_percent": 66.7,
      "checkin_record": {
        "record_date": "2026-04-05",
        "is_completed": true,
        "quantity": 5.5
      }
    }
  ],
  "inactive_goals": [],
  "calendar_days": [...],
  "day_details": [...],
  "week_details": [...],
  "month_details": [...]
}
```

### 年度统计

`GET /api/goals/year-summary`

```bash
curl "http://localhost:8080/api/goals/year-summary?year=2025" \
  -b 'e4_session=your-session-cookie'
```

**响应**：
```json
{
  "year": 2025,
  "has_records": true,
  "recorded_goal_count": 5,
  "total_checkins": 342,
  "recorded_days": 156,
  "start_date": "2025-01-01",
  "end_date": "2025-12-31"
}
```

### 打卡（创建/更新）

`PUT /api/goals/:id/records/:date`

**请求体**（checkbox 型）：
```json
{}
```

**请求体**（quantity 型）：
```json
{
  "quantity": 5.5
}
```

### 删除打卡记录

`DELETE /api/goals/:id/records/:date`

```bash
curl -X DELETE "http://localhost:8080/api/goals/1/records/2026-04-05" \
  -b 'e4_session=your-session-cookie'
```

## 打卡规则

1. **日期限制**：只能填写今年且不晚于今天的打卡记录
2. **默认打卡日期**：昨天（若昨天是今年）
3. **数量型目标**：必须填写大于 0 的数量
4. **完成状态**：checkbox 型目标有记录即完成；quantity 型目标需同时有数量

## 前端实现

前端路由：`web/src/routes/goals/+page.svelte`

主要功能：
- 目标创建、编辑、软删除
- 每日打卡（支持补签）
- 目标进度可视化
- 周/月统计详情
- 往年年度统计回看

## 相关文件

| 文件 | 说明 |
|------|------|
| `internal/handlers/goals.go` | 目标处理器实现 |
| `internal/models/goal.go` | 目标数据模型 |
| `web/src/routes/goals/+page.svelte` | 目标页面组件 |
| `web/src/lib/api.ts` | API 调用封装 |
