# 文档同步矩阵

代码变更后对照下表（L2+ 必做；L0/L1 仅 CHANGELOG）。

| 改了什么 | 同步什么 |
|----------|----------|
| 入口/路由/API | PROJECT_MAP、README、CHANGELOG |
| 配置/环境变量 | .gitignore、config 样例、PROJECT_MAP |
| 新 provider/adapter | PROJECT_MAP 扩展点表、config 样例、doc/ 对应模块文档 |
| 新目录/包 | PROJECT_MAP 目录地图 |
| 新错误模式（L1） | BUG_TRIAGE 表 |
| WebSocket/MQTT 协议变更 | `doc/websocket_server.md` 或 `doc/mqtt_udp.md`、PROJECT_MAP |

## 规则维护

| 文件 | 何时改 |
|------|--------|
| `00-core.mdc` | L0–L3 或入口变化 |
| `02-go.mdc` | Go 约定变化 |
| `03-extensions.mdc` | 新增 provider 模式变化 |
| `04-api.mdc` | API 路由/协议约定变化 |

 governance 本身变更 → CHANGELOG + HOW_TO_USE。
