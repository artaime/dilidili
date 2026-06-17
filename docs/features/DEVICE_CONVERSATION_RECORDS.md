# 设备对话记录

## 概述

小程序与管理端设备管理支持按设备查看统一对话时间线：AI 聊天记录（`chat_messages`）与已在**固件端播放**的家长留言（`parent_messages.status=played`）按时间合并展示。支持游标双向分页、按日期锚点搜索、纯播放（不写库）。

## 范围

- 小程序：`pages/device-records/`
- 管理端：`/admin/devices/:id/conversation-records`
- API：`/api/mp/devices/:id/conversation-records`、`/api/admin/devices/:id/conversation-records`
- 数据：复用 `chat_messages` 写入链路与 `ChatHistoryController` 音频存储

## 时间线规则

| 类型 | 来源 | 排序时间 | 展示 |
|------|------|----------|------|
| `chat` | `chat_messages` | `created_at` | user 右气泡 / assistant 左气泡 |
| `parent_message` | `parent_messages`（仅 `played`） | `played_at` | 居中「家长留言」气泡 |

- 仅展示 `role` 为 `user` / `assistant` 的聊天消息
- 家长留言须设备端播放完成后（`status=played` 且 `played_at` 非空）才进入时间线
- 小程序/管理端播放音频**不修改**任何数据库状态

## 分页

| 场景 | 参数 | 行为 |
|------|------|------|
| 首次进入 | `limit=20` | 最新 20 条，界面自上而下时间正序 |
| 加载更早 | `before_sort_time` + `before_type` + `before_id` |  prepend |
| 加载更新 | `after_sort_time` + `after_type` + `after_id` | append |
| 按日期 | `date=YYYY-MM-DD` | 该日 00:00 起前 20 条（正序） |

## API

### 小程序（JWT + 设备归属）

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/mp/devices/:id/conversation-records` | 合并列表 |
| GET | `/api/mp/conversation-records/chat/:id/audio` | 聊天音频 |
| GET | `/api/mp/messages/:id/audio` | 家长留言音频（已有） |

### 管理端（AdminAuth）

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/admin/devices/:id/conversation-records` | 合并列表 |
| GET | `/api/admin/conversation-records/chat/:id/audio` | 聊天音频 |
| GET | `/api/admin/conversation-records/parent/:id/audio` | 家长留言音频 |

## 依赖

- 设备须绑定智能体且 `device_name`（SN）非空，方可查询 `chat_messages`
- 主服务经 `/api/internal/history/messages` 写入聊天记录（不变）
