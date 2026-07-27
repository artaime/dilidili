# Agent 指南

**人类请先读** [docs/dev/HOW_TO_USE.md](docs/dev/HOW_TO_USE.md)。

## 项目

ESP32 智能终端 AI 语音后端，端到端全流式 ASR → LLM → TTS，支持 WebSocket / MQTT-UDP 多协议接入与管理控制台。

技术栈：Go 1.24 主服务 + Gin 管理后端 + Vue 前端 · 治理档位：full

## 开始

1. [HOW_TO_USE.md](docs/dev/HOW_TO_USE.md) — L0–L3
2. [PROJECT_MAP.md](docs/PROJECT_MAP.md) — 架构与入口
3. L1 bug → [BUG_TRIAGE.md](docs/dev/BUG_TRIAGE.md)（standard+）

## 任务分档

| 档位 | 必做 |
|------|------|
| L0 | code + CHANGELOG 一行 |
| L1 | 定位 → fix + CHANGELOG Fixed |
| L2 | FEATURE.md → code + DOC_SYNC |
| L3 | Plan + ADR |

L0/L1 **不要** FEATURE.md。

## 收尾

- `go test ./...`
- CHANGELOG `[Unreleased]`

## 规则

- alwaysApply：`.cursor/rules/00-core.mdc`
- 小程序素材：`.cursor/rules/05-miniprogram-assets.mdc`（**>50KB** 才 `/static/mp` 网络下发）
- 其他：编辑匹配文件时按 glob 加载

## 禁止

- 勿提交：`.env`、`config/*.pro.yaml`、`*.dev.yaml`、`*.local.*`、`*.pem`、`*.key`、`logs/`、`*.db`
