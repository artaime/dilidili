# DeepSeek Thinking 默认关闭

## 状态

done

## 需求

- 背景：DeepSeek Chat Completions（含 V4）在未传 `thinking` 时默认 `type=enabled`，语音链路首包变慢、耗 token 升高。现有 `thinking.mode=default`/未配置不会注入关闭参数。
- 验收标准：
  - `provider=deepseek` 且未配置 `thinking`（或 `mode=default`）时，请求体注入 `thinking: {type: disabled}`
  - 配置 `thinking.mode: enabled` 可显式开启
  - 管理端 DeepSeek 表单默认「关闭」，可选手动开启

## 设计

- 影响模块：`internal/domain/llm/eino_llm/thinking.go`、管理端 `llmCatalog.js` / `LLMConfigForm.vue`、config 样例
- API/配置变更：LLM config 已有字段 `thinking.mode`（`enabled`|`disabled`）；DeepSeek 未配置时运行时默认按 `disabled` 注入

## 开发计划

- [x] 实现
- [x] `go test ./...`
- [x] CHANGELOG

## 测试

- 单测：`applyProviderThinkingDefaults` / `injectThinkingPayload` 对 deepseek 未配置时注入 disabled

## 改动记录

| 日期 | 摘要 |
|------|------|
| 2026-07-17 | DeepSeek thinking 默认关闭并保留配置开关 |
