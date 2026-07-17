# 讲故事多样性与人物名防复用

## 状态

done

## 需求

- 背景：用户说「讲个故事 / 随便讲」时，模型常产出题材雷同、主角名刻板（如小明/小兔子）的故事，且容易复用近期讲过的人物名，听多了易腻。
- 验收标准：
  - 开放/随便生成时，系统从题材池随机指定一类（童话/历史/神话/寓言/冒险/侦探/科幻/生活），并尽量避开本设备近期已用题材
  - 同时给出新鲜主角名种子，并注入近期人物名回避名单
  - 用户点名经典正篇（canonical）不受影响；用户已指定明确主题时保留主题，仍做人物名回避
  - 生成 meta 可携带 `characters`，供后续回避

## 设计

- 影响模块：
  - `internal/domain/story/diversity.go` — 题材池、主角名池、近期回避收集、种子选取
  - `internal/domain/story/reference.go` — system/user prompt 注入多样性约束
  - `internal/domain/story/service.go` — `PlanGenerate` / `handleGenerate` 组装种子
  - `internal/domain/story/meta.go` — 解析并落库 `characters`
- API/配置变更：无新配置键；行为内置

## 开发计划

- [x] 实现 diversity 种子与 prompt
- [x] 单测
- [x] `go test ./internal/domain/story/...`
- [x] CHANGELOG / DOC_SYNC

## 测试

```bash
go test ./internal/domain/story/...
go test ./internal/app/server/chat/... -run Story
```

## 改动记录

| 日期 | 摘要 |
|------|------|
| 2026-07-17 | 开放生成强制题材轮换 + 主角名种子 + 近期人物回避；meta 增加 characters |
