# 小程序设备解绑出厂重置

## 状态

done

## 需求

小程序解绑设备时：

1. 删除该设备 SN 相关的全部业务数据（记忆、故事、对话、留言及音频）
2. 将 `devices` 行恢复为出厂登记状态（保留 SN、激活码、出厂智能体）
3. 通知主服务踢线并清理内存态

## 出厂登记状态

| 字段 | 解绑后 |
|------|--------|
| `user_id` | `0` |
| `activated` | `false` |
| `nick_name` | `""` |
| `role_id` | `NULL` |
| `last_active_at` | `NULL` |
| `agent_id` | **保留** |
| `device_name` / `device_code` / `challenge` / `pre_secret_key` | **保留** |

## 清理范围

| 存储 | 范围 | 说明 |
|------|------|------|
| Memobase | SN → primary + legacy user_id | 未配置 memobase 时 skip |
| Redis | `{prefix}:story:{sn}:*`、`llm:{sn}`、`llm:system:{sn}`、`userconfig:{sn}` | 未配置 Redis 时 skip |
| MySQL | `chat_messages`（`device_id`=SN）、`parent_messages`（`device_id`=设备 DB id）、`device_members`、`device_invites` | 硬删 + 音频文件；家庭成员与邀请一并清空 |
| 主服务内存 | ChatManager、OpenClaw offline、MCP session | 经 WS `POST /api/device/reset` |

**不删**：声纹组（属 user）、智能体（出厂关联）。

## 流程

1. 校验设备归属
2. **best-effort** 经 Manager WebSocket 广播 `/api/device/reset` 至主服务
3. Memobase → Redis → DB/文件（任一步失败则**不**改 `devices.user_id`）
4. 重置 `devices` 出厂字段

## API

| 方法 | 路径 | 说明 |
|------|------|------|
| DELETE | `/api/mp/devices/:id` | 小程序解绑（JWT，仅设备所属用户） |
| POST | `/api/admin/devices/:id/factory-reset` | 管理端出厂重置（admin 鉴权，不校验 user 归属） |
| DELETE | `/api/admin/devices/:id` | 管理端删除设备（先出厂重置清理业务数据，再删 `devices` 行） |

## 管理端与小程序解绑

| 操作 | 行为 |
|------|------|
| 小程序解绑 | 同出厂重置：清数据 + 保留出厂登记字段 |
| 管理端「出厂重置」 | 复用同一 `DeviceResetService`，不校验设备归属 |
| 管理端编辑设备 | **禁止**将 `user_id` 从非 0 改为 0（须走出厂重置） |
| 管理端编辑设备 | `user_id=0` 时不可设 `activated=true` |
| 管理端删除设备 | 先 `ResetToFactoryByAdmin`，再删除 `devices` 行 |
| 管理端「清空 Memobase 长期记忆」 | 仅清 Memobase，**不含**对话/Redis/故事；全量请用出厂重置 |

## 相关文件

- `manager/backend/services/device_reset/` — 编排服务
- `manager/backend/controllers/mp_device.go` — 小程序解绑入口
- `manager/backend/controllers/admin.go` — 管理端出厂重置 / 删除
- `manager/backend/controllers/websocket.go` — `NotifyDeviceReset`
- `internal/app/server/app.go` — `HandleDeviceReset`
- `internal/domain/story/store.go` — `DeleteAllForDevice`

## 注意事项

- manager `redis.key_prefix` 须与主服务 `config.yaml` 中 `redis.key_prefix` 一致
- 解绑后原用户**无法**再查看该设备对话（硬删，非软删）
- mem0/memOS 长期记忆首版未清，后续按需扩展
