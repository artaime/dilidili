# Changelog

格式基于 [Keep a Changelog](https://keepachangelog.com/zh-CN/)。

## [Unreleased]

### Fixed

- 小程序留言列表播放语音 401：补充 `GET /api/mp/messages/:id/audio`（JWT 鉴权），列表返回 `audio_url` 指向该接口
- 固件端家长留言 JSON 解析失败：主服务兼容 pending API 旧版单对象/新版数组响应；失败后允许重试，MQTT transport_ready 时再次拉取
- `config.yaml` 默认 `manager.auth_token` 与控制台 `internal_auth_token` 对齐，避免内部 API 401
- 固件 WiFi MAC 与 BLE MAC 不一致：小程序绑定时将 BLE MAC 规范为 WiFi MAC，绑定成功后将 `device_name` 同步为该值；后端仅做格式归一与精确匹配
- 家长留言 MQTT 播报：`hello` 完成后再启动；`speak_ready` 超时等可重试错误不标记 skipped；默认等待超时 15s
- AI 玩具协议兼容：默认关闭 `speak_request` 握手，主动播报（含家长留言）改走 hello + session + UDP + tts 标准链路
- 家长留言通知超时：hello/UDP 就绪后重置 60s 检测窗口；UDP 首次绑定时再触发；日志输出分环节就绪状态（`hello`/`session`/`udp_binding` 等）

### Added

- 初始化项目治理（full 档）：`governance-kit/`、`llms.txt`、`scripts/check-governance.ps1`；精简 `AGENTS.md`

### Changed

- MQTT 监听端口 6887、TLS 6888；管理前端 dev 代理指向远程 API

### Added

- 家长留言升级：亲属角色（`family_role`）、留言标题、语音原声存储（最长 60 秒，无需 ASR）
- 小程序留言 UX：按住说话、上滑取消、松手填标题、语音/文字列表徽章
- 内部 API：`GET /api/internal/parent-messages/:id/audio`；pending 返回全部待播留言
- 主服务留言编排：3 小时间隔批量播放、LLM 意图识别、语音 MediaPlayer 原声 / 文字 TTS

### Changed

- 家长留言设备端：由固定文案 + 关键词改为 Agent 人设询问 + LLM 意图（关键词降级）
- `PATCH /api/mp/profile` 支持更新亲属角色；登录/资料返回 `family_role`

### Added
- 家长留言微信小程序（子模块 `manager/miniprogram` → `dilidili_mp`）：微信登录、设备绑定、文字/录音留言
- 小程序 API（`/api/mp/*`）：鉴权、设备 check/bind、留言 CRUD
- `parent_messages` 数据模型与内部 pending 查询 API
- 主服务设备上线家长留言 TTS 询问 + ASR 确认播报流程
- 管理端设备 MAC 出厂预登记校验提示

### Added

- 小程序绑定流程优化：BLE 获取 MAC 后先绑定再 WiFi 配网，配网完成后开放系统设置
- 小程序登录页 chooseAvatar + 昵称；首页/我的页展示头像昵称
- 小程序「我的设备」解绑（`DELETE /api/mp/devices/:id`）
- 文档 [`docs/features/MP_BIND_FLOW.md`](../features/MP_BIND_FLOW.md)

### Changed

- 小程序绑定自动创建智能体时使用光曜星角色介绍模板（支持 `{{assistant_name}}` 占位符）
- 设备表唯一索引由 `device_code` 单列改为 `device_name` + `device_code` 联合唯一
- 小程序 device-config 移除底部手动绑定卡片，绑定前置至 ble-connect
- 更新 [`AI玩具BLE流程.md`](../AI玩具BLE流程.md) 绑定时序说明

### Fixed

- 绑定放在流程末尾导致用户配网后未绑定时首页/我的设备仍显示「未绑定」

### Removed

## [0.1.0] - 2026-06-10

### Added

- 初始版本

[Unreleased]: compare/link/here
