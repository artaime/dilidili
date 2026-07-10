# 主服务运行时监控（自研看板）

管理端与主服务分机部署时，管理员可在控制台实时查看各主服务节点的 CPU/内存/磁盘/带宽、连接在线数、会话活跃数、资源池占用与 WS 连通状态。

## 指标口径

| 指标 | 字段 | 说明 |
|------|------|------|
| 节点存活 | `status` | `online`：30s 内有上报；`offline`：超时 |
| WS 连通 | `ws_connected` | 管理端 WS `clientsMap` 中 node_id 对应客户端在线 |
| 连接在线数 | `app.chat_manager_count` | 主服务 `ChatManager` 数量 |
| 会话活跃数 | `app.active_session_count` | 有 `ChatSession` 且状态为 listening/llmStart/ttsStart |
| 业务活跃设备 | `summary.devices_active_5m` | DB `last_active_at` 5 分钟内（全局，非单节点） |

## 主服务配置

```yaml
server:
  node_id: "main-01"      # 多节点唯一；可用环境变量 DILI_NODE_ID 覆盖
  node_name: "主服务-01"   # 看板展示名

runtime_report:
  enabled: true
  interval: 5s
  disk_path: "/"
```

- WS 连接 UUID 与 `node_id` 一致，便于管理端关联
- 上报接口：`POST /api/internal/runtime/report`（`InternalServiceAuth`）
- 兼容旧接口：同时双写 `POST /api/internal/pool/stats`

## 管理端 API（Admin）

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/admin/runtime/nodes` | 所有节点最新快照 |
| GET | `/api/admin/runtime/nodes/:node_id` | 单节点详情 |
| GET | `/api/admin/runtime/summary` | 集群汇总 + 业务活跃设备数 |
| GET | `/api/admin/runtime/stream-token` | 签发 SSE 短期 token（5min） |
| GET | `/api/admin/runtime/stream?token=` | SSE 推送（5s） |
| GET | `/api/admin/runtime/ws-clients` | WS 客户端调试列表 |

## 前端入口

- 菜单：**服务监控** → `/admin/server-monitor`
- Dashboard 管理员区展示：在线节点、连接在线、会话活跃、设备活跃(5min)

## Phase 2 预留

- 节点 CRUD（DB）、历史趋势、阈值告警
- 废弃独立「资源池统计」页
