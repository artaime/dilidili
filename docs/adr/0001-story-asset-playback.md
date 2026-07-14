# ADR 0001：故事正文资产与设备播放态拆分

## 状态

已接受

## 背景

故事正文、播放进度与设备 Redis 键强绑定，导致：无法跨用户复用完整经典故事；聊天历史被迫存全文；生成流与 Session/TTS 生命周期耦合，插话即中断长文生成。

## 决策

1. **拆分存储**：`story_assets`（正文，可共享）与 `story_playbacks`（设备播放态）；Manager MySQL 为 SoT，主服务 Redis 为热缓存并 dual-write。
2. **共享范围（方案 A 增补）**：双池——`named`（经典正篇 + `canonical_key`/别名表）、`open`（讲个/随便/theme 空）、`bedtime`（睡前无点名，空则回落 open）；完整生成且入池才 shareable。指定原创剧情不入池。取用排斥设备近 N 天 playback，候选 Top-K 随机；不做向量检索。
3. **生成保护**：已成功 TTS 的故事正文 rune ≥ 配置阈值（默认 300）时停播不停写；生成 context 与 SessionCtx 解耦；不新建故事专用 LLM client pool。
4. **聊天展示**：assistant 消息存短卡片文案，全文经 `story_id` 按需拉取。
5. **续讲/复播**：一律基于本设备 playback，永不借用他人进度。

## 后果

- 新增 internal/mp 故事 API 与 AutoMigrate 表（含 `story_asset_aliases`）。
- 主服务写入路径增加异步持久化；迁移期读 MySQL 未命中回落 Redis。
- 管理端/小程序改为以 MySQL（及回落）展示故事详情。

## 关联

- FEATURE：`docs/features/story-asset-playback/FEATURE.md`
- ADR：本文件
