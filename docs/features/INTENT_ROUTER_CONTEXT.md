# 意图路由短上下文 + 事实/讲故事消歧

## 状态

done

## 需求

- 背景：ASR 后故事旁路 / 意图路由在主 LLM 前截获，分类只看当前句；`general` 单句编回复、故事 followup 固定话术、以及「介绍……的故事」被当成儿童编故事，导致指代/回忆/事实介绍答错。
- 目标：分类带近期 Dialogue；意图判定不做关键词/正则；闲聊交主对话；通用消歧「事实介绍 vs 儿童讲故事」。
- 验收标准：
  1. 「介绍建筑…介绍第二个」由主对话答，不被留言能力话术截走。
  2. 「我刚刚问的问题是什么」不走「刚才好像还没讲故事」固定话术。
  3. 列举真实事物后说「介绍第 N 个 / ……的故事/历史」→ 事实介绍，不启动 `create_child_story`。
  4. 明确「有留言吗 / 播放留言 / 讲个故事 / 女娲补天」仍可旁路直达。
  5. LLM 分类失败时不关键词兜底，直接放行主对话。
  6. 意图分类路径无关键词/`Contains` 快路径、无正则意图判定（JSON 提取解析除外）。

## 设计

```
ASR → 故事意图(近期对话) → 可执行故事动作才截
    → 意图路由(近期对话) → 仅 msg_inquiry / msg_play 截
    → 主 LLM + short_context
```

- 分类 user prompt 注入近期 Dialogue（条数上限与 short_context 对齐，分类侧再封顶）
- `needs_dialogue` / `general` / `device` → 放行主对话
- 故事：事实介绍类「……的故事」→ `action=none`；followup 无故事可答 → 放行
- 主 LLM `buildStoryRoutingPolicy` 双层禁止对事实介绍调 `create_child_story`
- 删除 `FallbackClassify`、`isStoryRecallQuestion`

## 开发计划

- [x] 实现分类上下文 + prompt/schema
- [x] 删除关键词分类；路由放行策略；故事消歧
- [x] 单测 + DOC_SYNC + CHANGELOG
- [x] `go test` 相关包

## 测试

```bash
go test ./internal/domain/chat/intent/... ./internal/domain/story/... ./internal/app/server/chat/...
```

## 改动记录

| 日期 | 摘要 |
|------|------|
| 2026-07-28 | 立项并实现 |
