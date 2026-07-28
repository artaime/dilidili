# 讲故事前标题过渡语

## 状态

done

## 需求

- 背景：复播 / 共享资产直读等路径会直接进入正文，缺少礼貌开场；用户希望讲正文前先有一句友好过渡语，告知故事标题等。
- 验收标准：
  1. `ready` / `replay`（含共享复用）朗读前先播报含标题的过渡语（如「好呀，接下来给你讲龟兔赛跑的故事。」）。
  2. `resume` 续讲：若正文已含「上次讲到…」衔接语则不再叠「继续讲标题」；否则补一句标题引导。
  3. 流式开放生成（无明确主题）：**只播一句**礼貌过渡语——等 meta 标题后播「好呀，接下来给你讲…」；不再先播通用「讲一个故事」再播「这篇故事叫」。
  4. 过渡语不计入故事正文播放进度；`story.filler_enabled=false` 时不播过渡语。
  5. 模型若再输出开场白，流式路径会跳过与系统过渡语重复的句子。

## 设计

- 影响模块：
  - `internal/domain/story` — `BuildNarrationIntro` / `BuildResumeTitleLead` / `BuildMetaTitleAnnounce` / `StripRedundantStoryOpening`
  - `internal/app/server/chat` — `narrateChildStory`、`deliverChildStoryFromTool`、`runStoryStream`
- API/配置变更：无；沿用 `story.filler_enabled`

## 开发计划

- [x] FEATURE
- [x] 实现 + 单测
- [x] `go test`（story / chat 相关）
- [x] CHANGELOG / DOC_SYNC

## 测试

```bash
go test ./internal/domain/story/... -run 'NarrationIntro|Speakable|MetaTitle|HasNarration|StripRedundant'
go test ./internal/app/server/chat/... -run PrependStory
```

## 改动记录

| 日期 | 摘要 |
|------|------|
| 2026-07-28 | 讲前标题过渡语：复播/续讲/开放流式 meta |
| 2026-07-28 | 修复开场过渡语双播：开放流式改等 meta 单句开场；续讲不叠衔接语；跳过模型重复开场白 |
