# 故事播报去重 / 进度 / 对话短卡片修复

## 状态

done

## 需求

- 背景：连说两次「讲一个故事」会串播两篇；未完成生成的故事被共享；管理端进度停留在开头且完播后显示「打断」；对话记录仍露出全文。
- 验收标准：
  1. 空主题与播报中的 generate 请求去重，不再串播下一篇。
  2. 仅 `generation_complete=true` 且有正文的故事可入共享池。
  3. 播报中周期性写入进度；正常播完标记 `completed`；管理端进度只显示百分比（无「y/x 段」）。
  4. 管理端对话记录与小程序一致：展示「播放故事：标题」，点击再拉全文。

## 设计

- 影响模块：
  - `internal/app/server/chat` — 开流去重、播报中拒重、历史短卡片、进度同步
  - `internal/domain/story` — 共享入池门槛
  - `manager/backend/services/story_persist` — shareable 兜底
  - `manager/frontend` — DeviceStories 进度文案、ConversationTimeline 故事卡片

## 开发计划

- [x] FEATURE
- [x] 实现
- [x] `go test`（chat/story/manager 相关）/ CHANGELOG / DOC_SYNC

## 改动记录

| 日期 | 摘要 |
|------|------|
| 2026-07-14 | 去重串播、不完整不共享、进度/完播态、管理端对话短卡片 |
