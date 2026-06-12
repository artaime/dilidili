# Changelog

格式基于 [Keep a Changelog](https://keepachangelog.com/zh-CN/)。

## [Unreleased]

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
