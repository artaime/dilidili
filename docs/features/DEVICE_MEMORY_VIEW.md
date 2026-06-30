# 设备长期记忆查看

## 概述

管理端设备管理支持按设备查看 Memobase **长期记忆**（Profile / Event / Context），并支持管理员清空该设备的 Memobase 用户数据。首版仅支持 Memobase Provider。

与「对话记录」的区别：

| 能力 | 对话记录 | 设备记忆 |
|------|----------|----------|
| 数据来源 | 本地 `chat_messages` / 家长留言 | Memobase API |
| 内容 | 会话原文时间线 | 画像、事件、注入 LLM 的上下文 |
| 前置条件 | 绑定智能体 + SN | 同上 + 智能体 `memory_mode=long` + 默认 Memory 为 memobase |

## 范围

- 管理端：`/admin/devices/:id/memory`
- API：`GET/DELETE /api/admin/devices/:id/memory`
- 共享算法：`pkg/memobaseuserid`（设备 SN → Memobase user_id）

## 前置条件

每个请求校验：

1. 设备存在且 `device_name`（SN）非空
2. 绑定智能体且 `memory_mode == long`
3. 默认启用的 Memory 配置 `provider == memobase`，且 `base_url` / `api_key` 非空

不满足时返回 400 与明确文案（如「设备未启用长记忆」），不调用 Memobase。

## 历史键兼容

修复 double-UUID 前，Memobase 实际 user_id 为「二次哈希」。页面优先读一次哈希；若为空则 fallback 读二次哈希，并在 UI 提示 `using_legacy`。

清空记忆时会同时尝试删除 primary 与 legacy 两个 user_id。

## API

### 管理端（Admin JWT）

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/admin/devices/:id/memory` | 聚合 Profile / Event / Context |
| DELETE | `/api/admin/devices/:id/memory` | 清空该设备 Memobase 长期记忆 |

GET 响应 `data` 字段示例：

```json
{
  "device_id": 1,
  "device_sn": "3Z73XX06PEV8FXV4G0NQD5R0FZ",
  "memory_mode": "long",
  "provider": "memobase",
  "memobase_user_id": "4cf315dc-...",
  "legacy_user_id": "bf0c5a5b-...",
  "using_legacy": true,
  "profiles": [],
  "events": [],
  "context": "..."
}
```

## 与 MCP memobase 的关系

- **智能体 MCP 服务**中的 memobase：控制 LLM 是否可用 MCP 工具主动存/搜记忆
- **本页面**：读取 Memory Provider 后台自动沉淀的 Memobase 数据
- 两者独立；未勾选 MCP memobase 不影响本页查看

## 验收

1. long 模式 + memobase 配置下，会话 Flush 后管理端可见 Profile/Event
2. 多设备共绑同一智能体，各 SN 独立记忆
3. 清空后 Profile/Event/Context 为空
4. short/nomemo 设备进入页面有明确提示，不 500
