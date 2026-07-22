# LLM 能力地面（防乱答/虚构操作）

## 状态

done

## 需求

- 背景：主对话 LLM 有时虚构未发生的操作（如「已帮你关灯」）、或主动推销未接入能力（如「还能帮你查天气、定闹钟」），仅靠人设 prompt 约束不足。
- 验收标准：
  1. 每轮 LLM 请求的 system prompt 注入与本轮实际 tools 同步的能力白名单与禁令（含「禁止推销工具清单外能力」）。
  2. 无 tool call 时，若回复含「已完成设备操作」类完成态话术，发往 TTS / 落历史前改写为「做不到」短句。
  3. 无对应工具却主动推销查天气/定闹钟等能力时，改写为仅陪伴向短句；意图路由 `general` 若含此类推销则改交主对话。
  4. 意图路由 `general` 路径同样禁止声称已执行操作、禁止推销未接入能力。
  5. 单测覆盖政策文案与改写逻辑；`go test ./...` 通过。

## 设计

- 影响模块：`internal/domain/llm/common`（地面策略）、`internal/app/server/chat/llm.go`（注入与流式护栏）、`internal/domain/chat/intent`（分类器禁令）、`intent_router.go`（general 回退）
- API/配置变更：无；不新增配置项（默认开启）

## 开发计划

- [x] 实现
- [x] `go test ./...`
- [x] CHANGELOG

## 测试

- `TestBuildCapabilityGroundingPolicy*`
- `TestMaybeRewriteUngroundedActionClaim*`
- `TestLooksLikeUngroundedCapabilityOffer*`
- `TestMaybeRewriteUngroundedCapabilityOffer*`
- 既有 `TestGetMessages*` 仍通过，且 system 含能力地面段

## 改动记录

| 日期 | 摘要 |
|------|------|
| 2026-07-17 | 补强：禁止推销未接入能力；general 虚构推销改交主对话；落历史前改写 |
| 2026-07-14 | 初版：tool 白名单注入 + 无工具完成态改写 + 意图分类器禁令 |
