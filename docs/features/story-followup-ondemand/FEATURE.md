# 故事追问：意图判定 + 按需拉原文

## 状态

done

## 需求

- 背景：当前「最近故事」把最多约 1500 字正文摘要写入会话，并在 30 分钟内**每轮**注入主 LLM system prompt，闲聊也白耗 token、拖慢首 token。故事已 dual-write 到 Redis Story Store + MySQL `story_assets`，无需常驻全文。
- 目标：用户可对已讲故事做任意形式追问；**先判意图，再按需取原文**，非追问轮零正文注入。
- 验收标准：
  1. 非故事追问轮：主 LLM system prompt **不再**注入故事全文/长摘要。
  2. 会话仅保留轻量指针：`story_id` + `title`（+ 时间戳）；TTL 默认 30 分钟。
  3. 判定为故事追问时：本轮才加载正文，注入后走主 LLM 短答；不调用 `create_child_story` 复播。
  4. 点名追问：优先本机正文；经典未讲过 → 通行情节直答；非经典未讲过 → 澄清线索，多轮仍不定则礼貌收尾，禁止瞎编/乱引导开讲。
  5. 复播/续讲行为不变。
  6. 单测 + `go test` 相关包通过。

## 设计

```
指针常驻 → 意图判定(followup) → 命中才拉文(Redis→MySQL) → 本轮短答
```

- `ClientState`：`SetRecentStoryPointer` / `RecentStoryPointer`
- 意图：`action=followup` / `is_story_followup`（`BuildStoryIntentSystemPrompt`，注入近期对话；事实介绍类「……的故事」为 `action=none`）
- 路由：`handleStoryFollowup` / `handleFollowupClarifyTurn`（`story_followup.go`）；无名追问且无最近故事 → **放行主对话**（不再固定话术）
- 取文：`Store.Get` → `FindLatestByTheme` → `storypersist.GetAsset`
- 回答：路由内 `callLLMSyncText` + `InjectMessage`；无关键词召回快路径

详见 [`INTENT_ROUTER_CONTEXT.md`](../INTENT_ROUTER_CONTEXT.md)。

### 配置

| 键 | 默认 | 说明 |
|----|------|------|
| `story.followup_enabled` | `true` | 总开关 |
| `story.followup_ttl_minutes` | `30` | 指针有效期 |
| `story.followup_max_runes` | `3000` | 本轮注入正文上限 |
| `story.followup_clarify_max_rounds` | `2` | 非经典澄清轮次 |

## 开发计划

- [x] 实现
- [x] `go test`（story / chat / client）
- [x] CHANGELOG + DOC_SYNC

## 测试

```bash
go test ./internal/domain/story/... ./internal/app/server/chat/... ./internal/data/client/...
```

## 改动记录

| 日期 | 摘要 |
|------|------|
| 2026-07-15 | 设计定稿并实现：指针 + followup + 按需拉文/经典直答/澄清收尾 |
| 2026-07-28 | 分类带近期对话；无指针 followup 放行主对话；去掉关键词召回快路径 |
