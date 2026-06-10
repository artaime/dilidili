# Changelog

格式基于 [Keep a Changelog](https://keepachangelog.com/zh-CN/)。

## [Unreleased]

### Added

- 初始化项目治理（full 档）
- 家长留言微信小程序（子模块 `manager/miniprogram` → `dilidili_mp`）：微信登录、设备绑定、文字/录音留言
- 小程序 API（`/api/mp/*`）：鉴权、设备 check/bind、留言 CRUD
- `parent_messages` 数据模型与内部 pending 查询 API
- 主服务设备上线家长留言 TTS 询问 + ASR 确认播报流程
- 管理端设备 MAC 出厂预登记校验提示

### Changed

### Fixed

### Removed

## [0.1.0] - 2026-06-10

### Added

- 初始版本

[Unreleased]: compare/link/here
