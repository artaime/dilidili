# 聊天历史仅保存用户说话音频

## 状态

done

## 需求

- 背景：聊天历史此前会持久化 AI TTS 音频（普通回复、故事短卡片、文字留言播报等），占用磁盘且非产品所需；只需保留用户 ASR 语音供回放与声纹训练。
- 验收标准：
  1. 用户语音对话后，`chat_messages.role=user` 有 `audio_path`，文件可播。
  2. AI 回复结束后，对应 `role=assistant` 有文本、无新 `audio_path`。
  3. 讲故事后，故事短卡片有文本、无新音频。
  4. 文字留言 TTS 播完后，chat_history 不新增 assistant 音频；家长语音留言原文件仍可在留言侧播放。
  5. 设备端回复/故事/留言播报正常；`go test ./...` 通过。

## 设计

- 影响模块：
  - `internal/app/server/chat/llm.go`：切断 assistant TTS 二阶段 `UpdateMessageAudio` 事件。
  - `internal/app/server/chat/tts.go`：去掉仅为历史服务的 `audioHistoryBuffer`。
  - `internal/app/server/message_handle.go`：非 user 角色跳过音频更新/写入（防御）。
- API/配置变更：无。家长语音留言仍存 `parent_messages/audio`，不在本次范围。
- 历史数据：已落盘的 assistant `audio_path` 不批量清理；新会话不再产生。

## 开发计划

- [x] 实现
- [x] `go test ./...`
- [x] CHANGELOG

## 测试

- 用户 ASR → 历史有用户音频。
- AI 回复 / 故事 / 文字留言 TTS → assistant 无新音频，设备播报正常。

## 改动记录

| 日期 | 摘要 |
|------|------|
| 2026-07-28 | 仅持久化用户 ASR 语音，停止 AI TTS 写入 chat_history |
