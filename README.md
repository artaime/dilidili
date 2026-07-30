# dilidili（狄哩 AI 语音后端）

基于开源 `dili-esp32-server-golang` 深度改造的 **ESP32 / AI 玩具** 语音后端。端到端全流式 **ASR → LLM → TTS**，对接 WebSocket / MQTT-UDP，配套 Web 管理控制台与家长微信小程序。

技术栈：Go 1.24 主服务 · Gin 管理后端 · Vue 前端 · 微信小程序

> 架构与模块地图见 [docs/PROJECT_MAP.md](docs/PROJECT_MAP.md)。日常协作分档见 [docs/dev/HOW_TO_USE.md](docs/dev/HOW_TO_USE.md)。

---

## 本仓库相对上游的主要能力

| 能力 | 说明 |
|------|------|
| 家长留言 | 小程序发语音/文字留言；设备端询问、意图播报、轮询新留言 |
| 设备绑定 | BLE 配网 + SN 绑定、家庭成员邀请、解绑出厂重置 |
| 儿童故事 | Local MCP 生成/复播/续讲、素材库、情节追问 |
| 短时上下文 | 跨 session 按 user+device+agent 灌入近期对话 |
| 意图路由 | ASR 后分类留言/故事/设备等，再决定是否进主 LLM |
| 隐私加密 | 对话与留言落盘 AES-GCM；管理端隐私闸门 |
| 运行时监控 | 主服务上报 + 管理端看板 / SSE |
| 固件协议 | AI 玩具协议路径：`/dili/v1/`、`/dili/ota/` 等 |

特性说明集中在 `docs/features/`。

---

## 部署方式概览

本仓库支持三种形态，**不要混用**（分离部署的主程序不要再开 AIO 的内嵌 manager，反之亦然）：

| 形态 | 适用场景 | 入口 |
|------|----------|------|
| **AIO 一体化** | 交付包、本机联调 | 单个 `dili_server`（内嵌控制台 + 声纹） |
| **源码分离** | 日常开发调试 | `cmd/server` + `manager/backend` + `manager/frontend`（+ 可选 `asr_server`） |
| **Docker Compose** | 容器化部署 | `docker/docker-composer/` |

详细步骤见 [doc/compile_deploy.md](doc/compile_deploy.md)、[doc/docker_compose.md](doc/docker_compose.md)、[doc/aio/](doc/aio/)。

---

## 方式一：AIO 一体化（推荐交付）

AIO 用 build tags 把管理后端、声纹服务、前端静态页打进主程序：

```text
go build -tags "nolibopusfile asr_server manager embed_ui" ...
```

打好后进程内默认：

- 配置文件：`main_config.yaml`（见 `cmd/server/defaults_config_embedded.go`）
- 内嵌 manager / asr_server：默认开启（可用 `-manager-enable=false` / `-asr-enable=false` 关闭）
- manager 配置默认：`manager.json`；asr 配置默认：`asr_server.json`

### 启动

解压发布目录后（结构参考 [doc/aio/README_linux.md](doc/aio/README_linux.md)）：

```bash
chmod +x dili_server
# AIO 包同目录需有 main_config.yaml / manager.json / asr_server.json
./dili_server
```

显式指定配置：

```bash
./dili_server \
  -c main_config.yaml \
  -manager-config manager.json \
  -asr-config asr_server.json
```

Windows 可用仓库脚本管理本机 bundle（路径需按实际包目录调整）：

```powershell
.\scripts\dili-service.ps1 start|stop|restart|status
```

### 自行从源码打 AIO

```bash
git submodule update --init --recursive

cd manager/frontend && npm ci && npm run build
mkdir -p ../backend/static/dist && cp -r dist/* ../backend/static/dist/

cd ../..
go mod tidy
go build -tags "nolibopusfile asr_server manager embed_ui" -ldflags "-s -w" -o dili_server ./cmd/server
```

运行时依赖、systemd、防火墙见 `doc/aio/README_*.md`；打包样例配置在 `build/common/`。

### AIO 默认端口（以包内配置为准）

| 端口 | 用途 |
|------|------|
| 8080 | 管理控制台 + API（`manager.json` → `server.port`） |
| 8989 | 设备 WebSocket / OTA |
| 2883 | MQTT |
| 8990 | UDP 音频（以控制台/配置为准） |
| 9000 | 内嵌声纹服务 |

控制台：`http://<主机>:8080/`

---

## 方式二：源码分离（推荐开发）

无 `manager` / `asr_server` tags 时，主程序**不**内嵌控制台与声纹，需分别启动。默认配置路径为 `config/config.yaml`。

### 1. 环境

- Go **1.24.x**
- Node.js **20.x**（前端）
- 子模块：`asr_server`、`manager/miniprogram`

```bash
git submodule update --init --recursive
```

Linux 公共依赖示例：

```bash
sudo apt-get install -y pkg-config libopus0 libopusfile-dev libc++1 libc++abi1
# Silero VAD 等还需 ONNX Runtime 1.21.0，见 doc/compile_deploy.md
```

### 2. 推荐启动顺序

1. 数据库（MySQL 或 SQLite）
2. （可选）Redis、Qdrant
3. （可选）声纹：`asr_server`
4. 管理后端：`manager/backend`
5. 主程序：`cmd/server`
6. 管理前端：`manager/frontend`（开发态）

**地址必须对齐**：主程序 `manager.backend_url` ↔ 后端实际端口；前端 `VITE_API_TARGET` ↔ 后端；声纹 `voice_identify.base_url` / `speaker_service.url` 一致。详见 [doc/compile_deploy.md](doc/compile_deploy.md)。

### 3. 管理后端

```bash
cd manager/backend
go run main.go -c config/config.json
# 或 ./start.sh / ./start.sh dev
```

默认监听配置中的 `server.port`（样例多为 `8080`）。也可用环境变量覆盖 MySQL：`DB_HOST` / `DB_PORT` / `DB_USER` / `DB_PASSWORD` / `DB_NAME`。

### 4. 主程序

```bash
# 编译
go build -o dili_server ./cmd/server

# 启动（分离形态：不要加 manager/asr_server tags，除非你有意内嵌）
./dili_server -c config/config.yaml
```

开发快速拉起（仅主程序）：

```bash
./run_server.sh
# 等价于：go run ./cmd/server -c config/config.yaml
```

命令行参数（`cmd/server/main.go`）：

| 参数 | 含义 |
|------|------|
| `-c` | 主配置路径 |
| `-manager-enable` | 是否启动内嵌 manager（需 `manager` build tag） |
| `-manager-config` | manager 配置路径 |
| `-asr-enable` | 是否启动内嵌 asr_server（需 `asr_server` build tag） |
| `-asr-config` | asr_server 配置路径 |

> 仓库内 `config/config.yaml` 的端口、内控地址可能按本机环境调整（例如 WebSocket 非 8989）。以**你正在使用的配置文件**为准，并与控制台 OTA / 设备下发地址一致。

### 5. 管理前端

```bash
cd manager/frontend
npm ci
npm run dev
```

开发页默认 `http://127.0.0.1:3000`，API 代理目标可用 `VITE_API_TARGET`。

### 6. 声纹服务（可选）

```bash
cd asr_server
CGO_ENABLED=1 go build -o voice_server main.go
./voice_server
```

---

## 方式三：Docker Compose

```bash
cd docker/docker-composer
docker compose up -d
docker compose ps
docker compose logs -f
```

`docker-compose.yml` 典型服务：MySQL、主程序、管理后端、管理前端、Qdrant、voice-server。本地源码构建可用 `docker-compose.local.yml`。

常见宿主机映射（以当前 compose 为准）：

| 服务 | 宿主机 |
|------|--------|
| 前端 | `:8080` |
| 后端 API | `:8081` |
| WebSocket | `:8989` |
| MQTT | `:2883` |
| voice-server | `:8082` |
| MySQL | `:23306` |

挂载：`config/` → 主程序；`manager/backend/config/` → 后端。说明见 [doc/docker_compose.md](doc/docker_compose.md)。

---

## 设备接入地址（协议路径）

主服务 WebSocket 路由（代码）：

| 路径 | 用途 |
|------|------|
| `ws://<host>:<port>/dili/v1/` | 语音会话 |
| `http://<host>:<port>/dili/ota/` | OTA |
| `ws://<host>:<port>/dili/mqtt_udp/v1/` | MQTT-UDP 相关 |
| `/dili/api/mcp/tools/{deviceId}` | 设备 MCP |
| `/ws/openclaw` | OpenClaw |

固件侧协议说明：[AI玩具协议.md](AI玩具协议.md)、[doc/esp32_dili_backend_guide.md](doc/esp32_dili_backend_guide.md)。

---

## 目录与入口

```
dilidili/
├── cmd/server/              # 主服务（-c；可选内嵌 manager / asr）
├── cmd/mqtt/                # MQTT 独立进程
├── internal/                # 协议层、Chat、VAD/ASR/LLM/TTS/MCP/故事等
├── manager/
│   ├── backend/             # Gin 管理 API + 小程序 API
│   ├── frontend/            # Vue 控制台
│   └── miniprogram/         # 家长小程序（子模块）
├── asr_server/              # 声纹服务（子模块）
├── config/                  # 主服务配置样例（*.pro/dev/local 勿提交）
├── build/                   # AIO 样例配置与平台依赖
├── docker/docker-composer/  # Compose 部署
├── scripts/                 # 运维脚本（含 Windows 服务管理）
├── doc/                     # 业务/部署文档
└── docs/                    # 治理与特性文档
```

| 项 | 命令 / 路径 |
|----|-------------|
| 主服务入口 | `cmd/server/main.go` |
| 管理后端 | `manager/backend/main.go` |
| 测试 | `go test ./...` |

---

## Provider（可插拔）

| 模块 | 实现目录（节选） |
|------|------------------|
| VAD | Silero / WebRTC / ten_vad |
| ASR | FunASR / 豆包 / 讯飞 / 腾讯 / 阿里云 等 |
| LLM | Eino（OpenAI 兼容等）/ Dify / Coze |
| TTS | 豆包 / Edge / CosyVoice / 腾讯 等 |
| MCP / RAG | `internal/domain/mcp/`、`rag/` |

配置入口：`config/config.yaml` + 管理控制台。扩展点表见 PROJECT_MAP。

---

## 文档索引

| 文档 | 内容 |
|------|------|
| [HOW_TO_USE.md](docs/dev/HOW_TO_USE.md) | L0–L3 协作流程 |
| [PROJECT_MAP.md](docs/PROJECT_MAP.md) | 架构与扩展点 |
| [CHANGELOG.md](docs/dev/CHANGELOG.md) | 变更记录 |
| [compile_deploy.md](doc/compile_deploy.md) | 分离部署与 AIO 打包 |
| [docker_compose.md](doc/docker_compose.md) | Compose |
| [config.md](doc/config.md) | 配置项 |
| [manager_console_guide.md](doc/manager_console_guide.md) | 控制台使用 |
| [websocket_server.md](doc/websocket_server.md) / [mqtt_udp.md](doc/mqtt_udp.md) | 协议接入 |
| `docs/features/` | 留言、故事、绑定、隐私等特性 |

---

## 仓库约定

- **勿提交**：`.env`、`config/*.pro.yaml`、`*.dev.yaml`、`*.local.*`、`*.pem`、`*.key`、`logs/`、`*.db`
- 小程序静态素材：**单文件 >50KB** 才放入 `manager/backend/static/mp` 网络下发
- 文档与提交说明：简体中文

---

## License

基于上游 [MIT](LICENSE) 许可；本仓库为内部改造版本，按团队规范使用与分发。
