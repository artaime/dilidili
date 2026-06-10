# 如何应用本项目的流程规范

人类主入口。智能体见 [AGENTS.md](../../AGENTS.md)。

**项目**：xiaozhi-esp32-server-golang · **档位**：full · **栈**：Go 1.24 主服务 + Gin 管理后端 + Vue 前端

---

## 1. 结构

| 层级 | 文件 |
|------|------|
| 日常 | `.cursor/rules/00-core.mdc` |
| 地图 | `docs/PROJECT_MAP.md` |
| 按需 | WORKFLOW、BUG_TRIAGE、DOC_SYNC（standard+） |

**原则**：L0/L1 先代码后文档；L2/L3 先 FEATURE/Plan。

---

## 2. 任务分档

```
L0  typo/单行     → code → go test ./... → CHANGELOG 一行
L1  bug           → BUG_TRIAGE → code → CHANGELOG Fixed
L2  新功能/行为   → FEATURE.md → code → DOC_SYNC → CHANGELOG
L3  架构/破坏性   → Plan → ADR → L2 流程
```

### 对 Cursor 说

- L0：「L0，CHANGELOG 一行。」
- L1：「L1，先 BUG_TRIAGE。」
- L2：「L2，先 FEATURE 我确认。」
- L3：「L3，Plan + ADR。」

---

## 3. 提交前

```bash
go test ./...
```

可选（full 档）：`powershell -File scripts/check-governance.ps1`

- [ ] CHANGELOG 已更新
- [ ] 未提交 `.env`、`config/*.pro.yaml`、`*.dev.yaml`、`*.local.*`、`*.pem`、`*.key`、`logs/`、`*.db`

---

## 4. 档位说明

本项目为 **full** 档。若觉得过重，可跟 Cursor 说「本次 L0」跳过 FEATURE。

---

## 5. 文档索引

| 文档 | 何时 |
|------|------|
| [PROJECT_MAP.md](../PROJECT_MAP.md) | 找模块 |
| [BUG_TRIAGE.md](BUG_TRIAGE.md) | L1 |
| [DOC_SYNC.md](DOC_SYNC.md) | L2+ 改完 |
| [WORKFLOW.md](WORKFLOW.md) | L2/L3 细节 |
| [CHANGELOG.md](CHANGELOG.md) | 每次提交 |

---

## 6. 快速参考

```
L0: code → test → CHANGELOG
L1: TRIAGE → code → test → CHANGELOG Fixed
L2: FEATURE → code → test → DOC_SYNC → CHANGELOG
L3: Plan → ADR → L2
```
