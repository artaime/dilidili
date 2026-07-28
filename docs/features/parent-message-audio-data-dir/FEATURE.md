# 家长留言音频迁入 data/

## 状态

done

## 需求

- 背景：聊天录音已落在 `manager/backend/data/chat_history/`，家长留言音频仍默认在 `storage/parent_messages/`，与 Go 源码包 `storage/` 混放，语义不一致。
- 验收标准：
  - 默认路径为 `./data/parent_messages/audio`
  - 配置样例与硬编码回退一致
  - 运行时音频由 `manager/backend/data/` 覆盖忽略

## 设计

- 影响模块：Manager `parent_message.audio_base_path`、留言上传/解绑清理/隐私迁移默认回退
- API/配置变更：仅默认路径变更；已显式配置的部署不受影响
- DB：`audio_path` 多为落盘时的完整相对路径；旧文件可继续按原路径读取。迁移存量时：移动目录并按需 `REPLACE` 路径前缀

## 开发计划

- [x] 实现
- [x] `go test ./...`
- [x] CHANGELOG

## 测试

- 配置为空时回退到 `./data/parent_messages/audio`
- 显式配置旧路径时行为不变

## 改动记录

| 日期 | 摘要 |
|------|------|
| 2026-07-28 | 默认路径迁至 `data/parent_messages`；清理 gitignore 冗余 |
