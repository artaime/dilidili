# Bug 定位指南

修 bug 时先读本页。智能体：L1 任务必读。

## 调用链

```
ESP32 设备
  → Transport（WebSocket / MQTT-UDP）
    → internal/app/server/chat/ChatManager
      → VAD → ASR → LLM → TTS（internal/domain/*）
      → MCP / OpenClaw（internal/domain/mcp、openclaw）
  → 管理台 Gin API
    → manager/backend/router → controllers → database
    → 内部接口回调主服务（/api/internal/*）
```

## 按现象查表

| 现象 | 优先文件 |
|------|----------|
| 启动/配置失败 | `cmd/server/main.go`、`config/config.yaml`、`internal/domain/config/` |
| 配置拉取/热更新异常 | `internal/domain/config/manager/`、`manager/backend/controllers/admin.go` |
| WebSocket 连接/断连 | `internal/app/server/websocket/websocket_server.go`、`websocket_conn.go` |
| OTA 升级失败 | `internal/app/server/websocket/ota.go`、`manager/backend/controllers/` |
| MQTT-UDP 断连/播报异常 | `internal/app/server/mqtt_udp/`、`doc/mqtt_udp.md` |
| 语音识别无结果/延迟高 | `internal/app/server/chat/asr.go`、`internal/domain/asr/` |
| LLM 回复异常/超时 | `internal/app/server/chat/llm.go`、`internal/domain/llm/` |
| ASR 正确但无回复 / `DoLLmRequest context canceled` / DeepSeek `context canceled` | `session.go`：chat turn 勿绑 `AfterAsrSessionCtx`；`realtime_mode=4` 空闲态勿 Stop；auto 下对话中勿 `TryRecoverStuckVoiceCapture` |
| auto 第二轮无回复 / 空拾音后 goodbye / `EmptyAudio` → `ChatSession.Close` | `isBenignAsrDisconnectError` 应始终保持会话（勿要求 hasActiveChatTurn）；另查 `TryRecoverStuckVoiceCapture` / soft listen stop |
| 唤醒后说话无识别、仅刷收包日志 / soft `VoiceStop` 卡住 | `HandleListenStop` soft stop、`TryRecoverStuckVoiceCapture` 清 soft stop；欢迎语期间 listen start 暂存见 `stashPendingListenStart` |
| TTS 播报中被 detect/listen start 误打断 | `HandleListenDetect` 助手输出门控；`shouldDeferListenStartDuringOutput` |
| 未满 `max_idle_duration` 就 goodbye | 上行须 `NoteUplinkActivity` 重置空闲；勿在 VoiceStop 跳过音频时仍累计 idle |
| 固件调音量后先拒再说成功/说「做不到」 | `llm.go` 能力地面改写 + `toolsSucceededInTurn`；`mcp/device_firmware_tools.go` |
| 问电量/调音量被闲聊截走、不调 MCP | `intent` 路由：`device` 意图应 fallthrough；查 `intent_router.go` / `prompt.go` |
| TTS 无声音/音色错误 | `internal/app/server/chat/tts.go`、`internal/domain/tts/` |
| VAD 误切/漏检 | `internal/domain/vad/` |
| MCP 工具调用失败 | `internal/domain/mcp/`、`internal/app/server/chat/mcp.go` |
| OpenClaw 会话/路由异常 | `internal/domain/openclaw/`、`internal/app/server/websocket/openclaw.go` |
| 知识库/RAG 召回异常 | `internal/domain/rag/`、`manager/backend/controllers/knowledge.go` |
| 管理台 API 401/403 | `manager/backend/middleware/auth.go` |
| 管理台 API 4xx 参数错误 | `manager/backend/controllers/`、`manager/backend/router/router.go` |
| 管理台 API 5xx / DB 错误 | `manager/backend/database/`、`manager/backend/controllers/` |
| 控制台 UI 异常 | `manager/frontend/src/views/` |

## 按错误文案 grep

```bash
rg "关键词" internal/ manager/backend/
```

| 关键词 | 位置 |
|--------|------|
| `config_provider` | `internal/domain/config/` |
| `ChatManager` | `internal/app/server/chat/` |
| `mcp-endpoint` | `manager/backend/controllers/`、`internal/domain/mcp/` |
| `openclaw` | `internal/domain/openclaw/`、`internal/app/server/websocket/openclaw.go` |
| `activation` | `manager/backend/controllers/device_activation` |

## L1 流程

BUG_TRIAGE 定位 → fix → `go test ./...` → CHANGELOG Fixed

复杂 bug 可选：`docs/features/fix-*/BUGFIX.md`（用 BUGFIX 模板）

## 维护

新错误模式修复后补充上表。
