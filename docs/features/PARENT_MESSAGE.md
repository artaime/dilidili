# 家长留言小程序

## 概述

为家长提供微信小程序，支持微信登录、BLE/手动 SN 绑定儿童设备、发送语音或文字留言。儿童设备联网并完成 hello+UDP 后，主服务可自动询问是否收听；儿童也可主动问「有留言吗」或说「播放留言」。语音留言播放家长原声，文字留言 TTS 播报。

## 范围

- 小程序端：`manager/miniprogram/`
- 管理 API：`manager/backend/` `/api/mp/*`
- 主服务留言投递：`internal/app/server/chat/parent_message*.go`
- 意图路由：`internal/domain/chat/intent/schemas.go`（**data 结构统一维护入口**）
- 设备留言档案：`internal/domain/chat/devicestate/`

## 用户流程

1. 管理员在控制台预录入设备 SN（`device_name`）
2. 家长微信登录、绑定设备、发送语音/文字留言
3. 设备 **hello + UDP 就绪** 后，主服务拉取 pending 并按规则编排；或儿童主动触发意图路由

## 意图路由（ASR 后、主 LLM 前）

| 意图 | 触发示例 | 行为 |
|------|----------|------|
| `msg_inquiry` | 「有留言吗」「爸爸留言了吗」 | 查档案 + pending，播报摘要，**不自动播放** |
| `msg_play` | 「播放留言」「继续播放」 | 有待播则播待播；无待播则播刚播过的那条 |
| `msg_play` | 「再播一遍」 | 重播上一条（不改 DB 状态） |
| `msg_play` | 「播放最近一条留言」 | `action=latest`，**刚播过的那条**；从未播过则回退创建时间最新 |
| `msg_play` | 「播放妈妈最近的留言」 | `action=latest` + `family_role`，该家长按 `created_at` 最新一条 |
| `msg_play` | 「播放妈妈昨天早上的留言」「播放爸爸下午的留言」「播放下午的留言」 | `action=select` + `family_role`/`start`/`end`；**含已播留言**，按 `created_at` 在库内检索 |
| `general` / `device` | 闲聊、设备能力 | **不截获**，交主 LLM（带 short_context） |

严格 JSON 示例：

```json
{"intent":"msg_inquiry","confidence":"0.95","needs_dialogue":false,"data":{"action":"list"}}
{"intent":"msg_play","confidence":"0.92","needs_dialogue":false,"data":{"action":"latest"}}
{"intent":"msg_play","confidence":"0.92","needs_dialogue":false,"data":{"action":"latest","family_role":"妈妈"}}
{"intent":"msg_play","confidence":"0.90","needs_dialogue":false,"data":{"action":"select","family_role":"妈妈","start":"2026-06-14T05:00:00","end":"2026-06-14T11:00:00"}}
{"intent":"general","confidence":"0.88","needs_dialogue":true,"data":{}}
```

分类器会注入近期 Dialogue；`needs_dialogue=true` 或 `general`/`device` 一律放行主对话。无关键词 fallback。详见 [`INTENT_ROUTER_CONTEXT.md`](./INTENT_ROUTER_CONTEXT.md)。

配置：`config.yaml` → `chat.intent_router`、`chat.device_message_profile`。

## 设备留言档案（每设备一份）

字段：`HasNewMessages`、`PendingCount`、`AllCaughtUp`、`PlayedHistory`、`LastPlayedMessageID`。

- 同步：拉取 pending + played API
- 播放完成：append 已播历史
- 重播：读 `PlayedHistory`，不修改 manager `status`

## 设备端播放规则

设 pending 列表按时间升序为 `M[0..n-1]`：

1. **首条或与前一条间隔 >3 小时**：TTS 自然询问是否播放（不引导口令）
2. **确认（仅主动询问流程）**：儿童回复经 **LLM 判断** `play` / `skip` / `unknown`
   - 同意 → 播放 → `played`
   - 拒绝 → 友好结束，**状态保持** `pending`/`notified`
   - 意图不清或与留言无关：LLM 自然过渡聊天（不指定口令），**继续等待**
3. 同批次间隔 ≤3 小时：过渡语后直接播放
4. 用户说「播放留言」等：`msg_play` 意图路由，跳过询问直接播

### 固件协议（AI 玩具 / pangdou-toy）

见 [AI玩具协议.md](../../AI玩具协议.md)。默认 `chat.speak_request_enabled: false`，主动播报走 **hello → session → UDP → tts**。

开机自动通知仅在 **hello + UDP 绑定** 后触发：有待播留言时 TTS 主动询问是否收听（`skip_ask=false`）。

使用中通过 `chat.parent_message.poll_interval_sec`（默认 30s）轮询；检测到**新留言 ID**（相对上次快照）且 hello+UDP 就绪时，再次主动询问。用户拒绝后同一条留言不会重复询问，直至家长发送新留言。**hello 重连不会清空已询问快照**，避免重复打扰。

## API

### 内部（主服务）

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/internal/devices/:device_name/parent-messages/pending` | 待播（`pending` + `notified`） |
| GET | `/api/internal/devices/:device_name/parent-messages/search` | 按 `start`/`end`/`limit` 检索可播留言（含 `played`） |
| GET | `/api/internal/devices/:device_name/parent-messages/played` | 已播列表 |
| GET | `/api/internal/parent-messages/:id` | 单条留言详情 |
| GET | `/api/internal/parent-messages/:id/audio` | 语音流 |
| PATCH | `/api/internal/parent-messages/:id/status` | 更新状态 |

## 状态机

```
pending → notified → played
                  ↘ expired（内容不可播）
```

**已删除 `skipped`**：儿童拒绝或播放失败时留言保持待播，可下次再查/再播。历史 `skipped` 数据在 manager 启动时迁移为 `pending`。

## 对话记录时间线

设备端播放完成（`status=played`）的家长留言，会按 `played_at` 与 AI 聊天记录合并展示在小程序/管理端「对话记录」页。小程序/管理端回放留言**不修改**留言状态。详见 [DEVICE_CONVERSATION_RECORDS.md](DEVICE_CONVERSATION_RECORDS.md)。

## 依赖

- 主服务与 Manager 内网可达
- 固件需主动发 hello 并建立 UDP 后方能下行 TTS
- 留言语音默认落盘：`manager/backend/data/parent_messages/audio`（配置 `parent_message.audio_base_path`；与聊天录音同属 `data/`）
