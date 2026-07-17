# 短时多轮衔接（Short Context Continuity）

## 状态

done

## 需求

- 背景：设备重连新建 session 后，Manager 按当前 `session_id` 加载历史常为空，短时多轮衔接断裂；仅 `device+agent` 隔离在换绑后会串前任机主对话。
- 验收标准：
  1. 新 session / 重连后，LLM 能带上当前 `user_id+device_id+agent_id` 下最近 N 条原文（默认进 prompt 10、加载 20）。
  2. 隔日续聊：昨日最后几轮在未挤出窗口、未出厂清空时可被加载；不承诺回忆昨日全部。
  3. Redis 近期窗口与 `config_provider` 解耦，优先读缓存、空则回落 DB；出厂重置清理 shortctx。
  4. goodbye 保留窗口内尽量复用内存 Dialogue；在线换绑 `user_id`/`agent_id` 变化清空 Dialogue。
  5. `user_id` 或 `agent_id` 无效时跳过短上下文，不退化为仅按 device 加载。

## 设计

- 隔离键：`user_id` + `device_id` + `agent_id`
- 影响模块：Manager configs / chat_history、主服务 UConfig/ClientState、session 历史加载、shortctx Redis、message_handle、hello/retention、device_reset purge
- 配置：`chat.short_context.*`
- 不改 long Search；保留「最近刚讲过的故事」旁路

## 开发计划

- [x] Phase 0：configs 下发 `user_id`
- [x] Phase 1：跨 session 三维 DB 加载 + 同步 init + 可配窗口
- [x] Phase 2：Redis shortctx + 出厂 purge
- [x] Phase 3：fresh hello 复用 + 换绑清 Dialogue
- [x] `go test`（核心包）+ CHANGELOG + DOC_SYNC

## 测试

- `TestHasValidShortContextIdentity` / `TestShortContextDefaults`
- `TestDeviceKeyPattern` / `TestStoreKey`
- `TestApplyOwnerIdentityClearsDialogueOnChange`
- `go test ./internal/app/server/chat/ ./internal/data/client/ ./internal/domain/memory/shortctx/`
- `go test`（manager）`./services/device_reset/ ./controllers/`

## 改动记录

| 日期 | 摘要 |
|------|------|
| 2026-07-15 | 立项并落地 Phase 0–3 |
