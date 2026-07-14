# LLM 能力地面（防乱答/虚构操作）

## 状态

done

## 需求

- 背景：主对话 LLM 有时虚构未发生的操作（如「已帮你关灯」）、或声称设备不具备的能力，仅靠人设 prompt 约束不足。
- 验收标准：
  1. 每轮 LLM 请求的 system prompt 注入与本轮实际 tools 同步的能力白名单与禁令。
  2. 无 tool call 时，若回复含「已完成设备操作」类完成态话术，发往 TTS / 落历史前改写为「做不到」短句。
  3. 意图路由 `general` 路径同样禁止声称已执行操作。
  4. 单测覆盖政策文案与改写逻辑；`go test ./...` 通过。

## 设计

- 影响模块：`internal/domain/llm/common`（地面策略）、`internal/app/server/chat/llm.go`（注入与流式护栏）、`internal/domain/chat/intent`（分类器禁令）
- API/配置变更：无；不新增配置项（默认开启）

## 开发计划

- [x] 实现
- [x] `go test ./...`
- [x] CHANGELOG

## 测试

- `TestBuildCapabilityGroundingPolicy*`
- `TestMaybeRewriteUngroundedActionClaim*`
- 既有 `TestGetMessages*` 仍通过，且 system 含能力地面段

## 改动记录

| 日期 | 摘要 |
|------|------|
| 2026-07-14 | 初版：tool 白名单注入 + 无工具完成态改写 + 意图分类器禁令 |
