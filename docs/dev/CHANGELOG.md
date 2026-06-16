# Changelog

格式基于 [Keep a Changelog](https://keepachangelog.com/zh-CN/)。

## [Unreleased]

### Changed

- 管理端添加设备：所属用户改为选填且下拉仅显示非普通用户；设备标识（SN）改为必填
- 设备唯一标识全局改用 SN：OTA 仅认 `board.sn`（AIToy 必填）；MQTT topic/clientId 中间段不再做 MAC `_`↔`:` 转换；小程序 BLE 绑定仅认 `sn`，移除 `ble_mac` 回退（详见 `docs/features/DEVICE_SN_IDENTITY.md`）
- 设备唯一标识从 MAC 切换为 SN：`device_name` 存 SN；小程序 check/bind 改用 `sn` 参数；BLE Notify 解析 `sn`；管理端预登记与 Web 绑定 UI 同步；MQTT clientId 中间段与 Device-Id 均使用 SN；激活时校验 `serial_number == device_id`
- 小程序绑定页仅保留蓝牙扫描绑定，移除手动输入 SN

### Fixed

- 家长留言询问语未播完即播放原声：询问/重试 TTS 结束后再开启 ASR 确认监听，避免「要听吗」等播报被误识别为肯定；留言流程进行中阻止主 LLM 并发接管
- 家长留言确认播放后缺少过渡语：孩子明确同意后先播报「好的，接下来将播放…」再播放原声/正文
- 小程序 BLE 扫描漏设备：完整解析广播 AD 结构、名称前缀大小写不敏感、已识别设备不因后续回调丢失 `advertisData` 被过滤；进入页面自动扫描
- OTA 激活检查仍用 Device-Id Header MAC 导致查不到设备：改为优先使用 OTA 请求体 `board.sn` 作为 `device_id` 查询激活状态

- 小程序留言列表播放语音 401：补充 `GET /api/mp/messages/:id/audio`（JWT 鉴权），列表返回 `audio_url` 指向该接口
- 固件端家长留言 JSON 解析失败：主服务兼容 pending API 旧版单对象/新版数组响应；失败后允许重试
- `config.yaml` 默认 `manager.auth_token` 与控制台 `internal_auth_token` 对齐，避免内部 API 401
- 家长留言 MQTT 播报：`hello` 完成后再启动；`speak_ready` 超时等可重试错误保持 `pending`
- AI 玩具协议兼容：默认关闭 `speak_request` 握手，主动播报（含家长留言）改走 hello + session + UDP + tts 标准链路
- 家长留言通知超时：hello/UDP 就绪后重置 60s 检测窗口；UDP 首次绑定时再触发；日志输出分环节就绪状态
- pending 查询包含 `notified` 状态，避免询问后列表漏条
- `msg_play` 按时间/亲属筛选时搜不到留言：新增内部 search API，在库内按 `created_at` 检索含 `played` 的留言；`latest`/筛选播放不再受已播列表 20 条上限影响
- 小程序留言列表：「新建留言」移至顶部；支持删除任意状态留言（含已播放）
- 小程序留言列表：`看记录` 仅显示当前设备留言；底部 tab「留言」显示全部设备留言

### Added

- 留言意图路由：`msg_inquiry`（查询）、`msg_play`（播放/重播/按条件筛选）、`general`（常规聊天）；JSON schema 见 `internal/domain/chat/intent/schemas.go`
- 设备留言档案 `DeviceMessageProfile`（新留言状态、已播历史、重播上一条）
- 内部 API：`GET .../parent-messages/played`、`GET .../parent-messages/:id`
- 内部 API：`GET .../parent-messages/search`（按时间检索含已播留言，供 `msg_play` select/latest）
- 初始化项目治理（full 档）：`governance-kit/`、`llms.txt`、`scripts/check-governance.ps1`；精简 `AGENTS.md`

### Changed

- 删除留言 `skipped` 状态；拒绝收听保持 `pending`，不可播标 `expired`；历史 `skipped` 迁移为 `pending`
- 取消 `manager_created` / `mqtt_transport_ready` 过早触发家长留言通知
- 家长留言：开机 hello+UDP 就绪后主动询问；使用中轮询检测新留言并主动询问（拒绝后不重复问同一条）
- `msg_play` 支持 `latest`（最近一条）与 `select`（按亲属 + start/end 时间范围筛选播放）
- 修复 `msg_play`/留言重播多段 TTS 时过渡语触发 goodbye 导致正文未播放
- 文字留言播放：过渡语/确认语使用 `ttsTurnEndPolicyNone`，正文 TTS 播完后再 goodbye；多段播报间等待 TTS 排空
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
