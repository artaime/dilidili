# 儿童故事 MCP 与 Story Memory

## 状态

done

## 需求

- 背景：ESP32 语音设备需按 [`doc/story.md`](../../doc/story.md) 生成适龄儿童故事，支持复播（「再讲一遍昨晚的故事」）、打断后续讲，并在 Redis 中保留至少 7 天且按复听加权延期。
- 验收标准：
  - 用户说讲故事 → 调用 `create_child_story` 生成并落库
  - 用户说再讲一遍 / 昨晚的故事 → 原样复播同一正文
  - 打断后说接着讲 → 从断点 segment 续播
  - 无年龄且无记忆 → 语音追问，不硬编年龄
  - 7 天内未复听故事可被 lazy eviction 清理；高频复听保留更久

## 设计

- 影响模块：
  - `internal/domain/story/` — 规范摘要、Store、生成、参数解析、保留策略
  - `internal/app/server/chat/` — Local MCP `create_child_story`、StoryPlaybackTracker、LLM 路由
  - `config/config.yaml` — `story.*`、`local_mcp.create_child_story`
- API/配置变更：无对外 HTTP API；新增 Local MCP 工具与 Redis key `{prefix}:story:{deviceID}:*`
- Memobase 偏好摘要：`PreferenceSync` 接口预留，首期不实现
- **流式生成（P0+P1）**：`generate` 路径先 TTS 过渡语，再 LLM 流式按句入 TTS，全文生成后落 Redis；复播/续讲仍整篇或分段朗读

## 配置（流式）

| 键 | 默认 | 说明 |
|----|------|------|
| `story.stream_enabled` | `true` | 生成走流式路径 |
| `story.filler_enabled` | `true` | 生成前播放过渡语 |
| `story.filler_default` | 见 config | 普通故事过渡语 |
| `story.filler_bedtime` | 见 config | 睡前故事过渡语 |
| `story.followup_enabled` | `true` | 情节追问按需拉正文 |
| `story.followup_ttl_minutes` | `30` | 最近故事指针有效期 |
| `story.followup_max_runes` | `3000` | 追问本轮注入正文上限 |
| `story.followup_clarify_max_rounds` | `2` | 非经典未讲过澄清轮次 |

情节追问详见 [`story-followup-ondemand/FEATURE.md`](./story-followup-ondemand/FEATURE.md)。

开放/随便讲的题材轮换与人物名防复用详见 [`story-diversity/FEATURE.md`](./story-diversity/FEATURE.md)。


## 开发计划

- [x] 实现
- [x] `go test ./...`
- [x] CHANGELOG

## 测试

```bash
go test ./internal/domain/story/...
go test ./internal/app/server/chat/... -run Story
go test ./...
```

## 改动记录

| 日期 | 摘要 |
|------|------|
| 2026-06-30 | 首期 P0~P2：create_child_story、Redis Story Store、复播/续讲、保留与弱偏好 |
| 2026-06-30 | 流式生成：过渡语 + LLM 分句 TTS（`story.stream_enabled` / `filler_*`） |
| 2026-07-17 | 开放生成题材轮换 + 主角名种子 + 近期人物回避（见 story-diversity） |
