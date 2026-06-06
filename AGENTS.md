# AGENTS.md

本文件面向 Codex 和其他 AI 编程助手，用于在本仓库内安全、低风险地开展代码阅读、修改和验证工作。

## 项目概述

本项目是面向 ESP32 / IoT 语音设备的小智 AI 后端服务，核心目标是提供设备接入、语音对话编排、ASR、LLM、TTS、VAD、MCP、OTA、MQTT/UDP、WebSocket 以及管理后台能力。

仓库当前包含三类主要服务：

- 主服务：`cmd/server`，负责设备 WebSocket / MQTT+UDP 接入、会话编排、ASR/LLM/TTS 调度、MCP、OTA、vision 等。
- 管理后台后端：`manager/backend`，独立 Go module，基于 Gin + GORM，提供管理 API、用户/设备/智能体/配置/知识库/声纹/历史记录等接口。
- 管理后台前端：`manager/frontend`，Vue 3 + Vite + Element Plus / Vant。

另有 `asr_server` Git submodule，按 `.gitmodules` 指向 `https://github.com/hackers365/asr_server.git`，在 AIO 构建和声纹服务链路中被引用。当前工作区内 `asr_server` 目录未展开出源码文件，相关细节只能根据父仓库文档、构建脚本和接口调用推断，需标记为待确认。

## 技术栈

- Go：根模块 `go.mod` 使用 `go 1.24.2`，toolchain `go1.24.11`。
- 主服务 HTTP/WebSocket：标准库 `net/http`、`gorilla/websocket`。
- 管理后台后端：Gin、GORM、JWT、SQLite/MySQL。
- 配置：Viper；主配置在 `config/config.yaml`；管理后台配置在 `manager/backend/config/*.json`。
- 缓存/历史：Redis 可选；管理后台数据库也保存历史记录。
- MQTT：`mochi-mqtt/server/v2` 作为内置 MQTT server，`paho.mqtt.golang` 作为 MQTT client。
- AI/语音能力：CloudWeGo Eino、OpenAI 兼容接口、Doubao、Aliyun、Xunfei、Edge TTS、FunASR、Silero VAD、WebRTC VAD、ten-vad 等。
- 前端：Vue 3、Vite、Vue Router、Pinia、Element Plus、Vant、Axios。
- 部署：Dockerfile、Docker Compose、GitHub Actions release/docker build workflow。

## 常用命令

主服务：

```bash
go build -o xiaozhi_server ./cmd/server
./xiaozhi_server -c config/config.yaml
```

AIO 构建（需前端构建产物、`asr_server` 子模块、CGO/ONNX/ten-vad 等环境）：

```bash
cd manager/frontend
npm ci
npm run build

cd ../..
go build -tags "nolibopusfile asr_server manager embed_ui" -ldflags "-s -w" -o xiaozhi_server ./cmd/server
```

管理后台后端：

```bash
cd manager/backend
go build -o main .
go run main.go -c config/config.json
```

管理后台前端：

```bash
cd manager/frontend
npm ci
npm run dev
npm run build
```

Docker Compose：

```bash
cd docker/docker-composer
docker compose up -d
docker compose ps
docker compose logs -f
```

Windows AIO 包服务脚本：

```powershell
.\scripts\xiaozhi-service.ps1 status
.\scripts\xiaozhi-service.ps1 start
.\scripts\xiaozhi-service.ps1 stop
.\scripts\xiaozhi-service.ps1 restart
```

测试：

```bash
go test ./...

cd manager/backend
go test ./...
```

说明：根模块和 `manager/backend` 是两个 Go module，需要分别测试。部分包依赖 CGO、ONNX Runtime、ten-vad、opus 或子模块，若本地环境不完整可能失败。

## 目录结构说明

- `cmd/server`：主服务入口、配置初始化、内嵌 manager/asr_server 的 build tag 适配。
- `cmd/mqtt`：独立 MQTT server 启动入口，读取 `config/mqtt_config.json`。
- `cmd/mock_ai_server`：本地 mock ASR/LLM/TTS HTTP/WebSocket 服务，供联调或测试。
- `internal/app/server`：主服务应用层，统一启动 WebSocket、MQTT server、MQTT+UDP adapter、事件处理、消息 worker。
- `internal/app/server/websocket`：设备 WebSocket、OTA、MCP WebSocket、OpenClaw WebSocket、vision API、消息注入等 HTTP handler。
- `internal/app/server/mqtt_udp`：MQTT + UDP 传输适配、UDP session、AES 加密音频通道。
- `internal/app/server/chat`：核心会话编排，包含 ChatManager、ChatSession、ASR/LLM/TTS 队列、MCP 工具、OpenClaw、媒体播放、打断逻辑等。
- `internal/app/mqtt_server`：内置 MQTT server、认证 hook、设备生命周期 hook。
- `internal/domain`：领域能力实现，包含 ASR、TTS、LLM、VAD、MCP、memory、rag、speaker、openclaw、eventbus、config 等。
- `internal/data`：客户端状态、消息类型、音频结构、历史客户端。
- `internal/components/http`：主服务访问 manager backend 的 HTTP client。
- `internal/db/redis`：Redis 初始化。
- `internal/pool`：资源池与统计上报。
- `internal/util`：音频、加密、队列、签名、manager auth、backend URL 等工具。
- `manager/backend`：管理后台后端，独立 Go module。
- `manager/backend/router`：Gin 路由集中定义。
- `manager/backend/controllers`：用户、管理员、设备、智能体、配置、知识库、声纹、历史、OpenClaw、MCP market 等控制器。
- `manager/backend/models`：GORM 模型。
- `manager/backend/database`：数据库连接、AutoMigrate、兼容迁移。
- `manager/backend/middleware`：JWT、管理员、内部服务、OpenAPI Token 鉴权。
- `manager/frontend`：Vue 管理后台。
- `config`：主服务示例配置、MQTT 配置、模型文件。
- `doc`：已有部署、协议、功能模块文档。
- `docker`：Dockerfile、Compose 文件、ONNX Runtime 包。
- `scripts`：Windows 服务管理、防火墙辅助脚本。
- `test`：手工/集成测试客户端与压测脚本，不是统一测试框架。
- `logger`：logrus 封装、设备日志。
- `build` / `dist` / `logs`：构建产物、运行日志或发布包相关目录，通常不应作为业务代码修改目标。

## 开发规范

- 修改前先阅读相关模块现有实现和测试，优先搜索同类 provider、controller、middleware、handler 的写法。
- 保持最小改动，不重构无关代码。
- Go 代码修改后运行 `gofmt`；如项目后续引入 goimports，再遵循项目约定。
- 不擅自新增大型依赖，尤其不要替换 Gin/GORM/Viper/Eino/MQTT/前端框架。
- 不擅自改数据库结构；如必须改模型，需确认 AutoMigrate 影响，并补充迁移/兼容说明。
- 不输出或硬编码真实密钥、token、密码、证书私钥。现有配置文件中有示例值或默认值，新增文档和示例应使用占位符。
- 不修改 `.env`、生产配置、发布包、日志、数据库文件，除非用户明确要求。
- README 和部分文档在当前终端中可能出现中文编码乱码；修改文档时应确认实际文件编码，避免引入二次乱码。

## API / RPC / 数据库相关约定

- 主服务设备入口集中在 `internal/app/server/websocket/websocket_server.go`：
  - `/xiaozhi/v1/`：设备 WebSocket 对话入口。
  - `/xiaozhi/mqtt_udp/v1/`：MQTT+UDP 模式的 WebSocket 协商入口。
  - `/xiaozhi/ota/`、`/xiaozhi/ota/activate`：OTA/激活相关。
  - `/mcp`：MCP WebSocket。
  - `/ws/openclaw`：OpenClaw WebSocket。
  - `/xiaozhi/api/mcp/tools/`：MCP tools API。
  - `/xiaozhi/api/vision`：视觉 API。
  - `/admin/inject_msg`：管理注入消息入口。
- 管理后台 REST API 集中在 `manager/backend/router/router.go`：
  - `/api/login`、`/api/register`、`/api/setup/*` 为公开接口。
  - `/api/internal/*` 和 `/api/configs`、`/api/system/configs` 中的内部调用使用 `InternalServiceAuth`。
  - `/api/user/*` 为普通用户功能。
  - `/api/admin/*` 需要 JWT + 管理员角色。
  - `/api/open/v1/*` 支持 JWT 或 API Token。
- 管理后台响应多数直接使用 `gin.H{"data": ...}` 或 `gin.H{"error": ...}`，未发现统一 response wrapper。新增接口应优先沿用附近 controller 风格。
- 数据模型集中在 `manager/backend/models/models.go`，由 `manager/backend/database/database.go` 中 `AutoMigrate` 管理。
- 主服务从 manager 获取配置的抽象在 `internal/domain/config/interface.go`，manager provider 实现在 `internal/domain/config/manager`。
- Redis provider 和 manager provider 都实现 `UserConfigProvider`，修改配置获取逻辑时要注意两种 provider。

## 测试和验证方式

- 根模块优先运行：`go test ./...`。
- 管理后台后端单独运行：`cd manager/backend && go test ./...`。
- 前端构建验证：`cd manager/frontend && npm ci && npm run build`。
- Docker/AIO 构建依赖较重，应先确认是否需要验证发布流程。
- 涉及设备协议、MQTT/UDP、MCP、OTA 的修改，应结合 `test/` 下对应客户端或 `doc/` 协议说明做手工联调。
- 涉及管理后台 API 的修改，应同时检查 `manager/frontend/src/utils/api.js`、`manager/frontend/src/router/index.js` 和对应页面调用。

## 修改代码前的注意事项

- 先确认目标属于主服务、manager backend、manager frontend 还是 asr_server 子模块。
- 主服务和 manager backend 是两个 Go module，依赖和测试命令不同。
- `cmd/server` 有多个 build tag 文件：
  - `manager` 控制是否内嵌管理后台。
  - `asr_server` 控制是否内嵌声纹/asr 子服务。
  - `embed_ui` 控制 manager 前端静态资源是否嵌入后端。
  - `nosilero` 影响 VAD 实现。
- MQTT+UDP 和 WebSocket 共用 ChatManager，但传输层行为不同，改会话逻辑要同时考虑两条链路。
- 设备鉴权和管理后台鉴权是两套逻辑：设备侧在 `internal/app/server/auth`，管理后台在 `manager/backend/middleware`。
- `internal/app/server/auth.AuthManager.ValidateToken` 当前直接返回 true，真实设备鉴权语义待确认，不要误认为已完成严格 token 校验。
- `asr_server` 当前未展开，涉及声纹服务源码时应先初始化子模块。

## 禁止事项

- 不要修改业务代码来“顺手整理”结构。
- 不要删除已有注释、配置、迁移或文档。
- 不要提交日志、SQLite 数据库、构建产物、密钥配置。
- 不要在文档中写入真实密钥、密码、token、证书内容。
- 不要把 `manager/backend/config/config.json`、`config/config.yaml` 中的示例密钥当作生产配置。
- 不要在未确认的情况下新增/删除数据库字段。
- 不要强行美化结构；发现混乱或待确认项要如实记录。
- 不要执行 `git commit`、`git push`、`git config`，除非用户明确要求。

## 完成任务时需要输出的内容

每次完成修改后，用中文说明：

- 修改了哪些文件。
- 为什么这样改。
- 如何验证，包含实际执行过的命令和结果；未执行也要说明原因。
- 哪些地方仍不确定或需要用户确认。

## 当前仍待确认的问题

- `asr_server` 子模块当前未展开，声纹服务内部入口、配置、模型和存储实现未能直接扫描。
- README 和部分 `doc/*.md` 在当前终端读取时出现中文乱码，实际编码、历史提交中是否存在混合编码需确认。
- 主服务设备侧 `auth.enable` 和 `AuthManager.ValidateToken` 的真实安全目标待确认。
- 管理后台 JWT secret 当前代码里有硬编码默认值，同时配置文件也有 `jwt.secret`，实际是否读取配置待确认。
- 根模块 `go test ./...` 在缺少 CGO/ONNX/ten-vad/asr_server 子模块时可能无法完整通过，CI 与本地验证边界需确认。
- OpenAPI 文档页面存在前端路由 `/openapi-docs`，但未发现独立 OpenAPI spec 文件，接口契约来源待确认。
