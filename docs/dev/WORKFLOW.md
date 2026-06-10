# 开发流程

> 日常用法见 [HOW_TO_USE.md](HOW_TO_USE.md)。本文供 L2/L3 细节。

## 流程

读 PROJECT_MAP → 定 L0–L3 → 实现 → test → CHANGELOG →（L2+）DOC_SYNC

## 非平凡改动

新功能、行为变更、配置结构变更 → 先 `docs/features/<slug>/FEATURE.md`

## 检查清单（L2+）

- [ ] FEATURE.md 需求与设计
- [ ] `go test ./...`
- [ ] CHANGELOG
- [ ] DOC_SYNC 触发表

## ADR

架构/破坏性变更 → `docs/adr/`
