# 故事资产库 / 进度保护 / 聊天短卡片

## 状态

done

## 需求

- 背景：故事进度与存储按设备 Redis 耦合；聊天历史塞入全文；流式生成随 TTS/Session 一并取消；无法复用其它用户已完成经典正文。
- 验收标准：
  1. 播放进度以已播字数/已生成字数呈现，并标记 `generation_complete`。
  2. 已播字数 ≥ `protect_continue_threshold`（默认 300）时，用户插话停播但不取消 LLM 续写；&lt;300 可打断生成。
  3. 聊天记录展示「播放故事：标题」，metadata 含 `story_id`；点击可拉全文。
  4. MySQL `story_assets` / `story_playbacks` 为持久 SoT；Redis 作设备热缓存。
  5. 双池共享：点名池 `named`（规范名+别名）与开放池 `open`/`bedtime`；设备近 7 天排斥 + Top-K 随机；指定原创剧情不入池。
  6. resume / replay / 断点续讲行为兼容。

## 共享规则（方案 A）

| 池 | 入池条件 | 取用匹配 |
|----|----------|----------|
| `named` | `ShouldTellCanonical` 且主题非空 | 意图 LLM 输出 `canonical`（ASR 纠错）优先查库；辅以 `theme_raw`/词典归一与 `story_asset_aliases` |
| `bedtime` | 睡前且无点名主题 | 本桶优先；空则回落 `open` |
| `open` | 「讲个/随便」或 theme 空 | `pool_kind` + age_band |
| 不共享 | `creative/original` 且具体原创剧情 | 始终新生成 |

防重复：排除设备近 `share_exclude_days`（默认 7）天 playback 的 `story_id`；候选按 `reuse_count` 取前 `share_pick_top_k`（默认 5）条随机一条；命中后 `reuse_count++`。

## 设计

- 影响模块：`internal/domain/story`、`internal/app/server/chat`、`manager/backend`（models/API）、小程序 `device-records`。
- API/配置变更：
  - `story.protect_continue_threshold`
  - `story.share_exclude_days` / `story.share_pick_top_k`
  - `POST/GET /api/internal/stories/*`（`shareable` 支持 `pool_kind`/`theme`/`device_sn`/`exclude_days`/`top_k`）
  - `GET /api/mp/devices/:id/stories/:storyId`
  - MySQL：`story_assets.pool_kind`/`canonical_key`；表 `story_asset_aliases`

## 开发计划

- [x] FEATURE + ADR
- [x] 阶段1 生成保护
- [x] 阶段2 聊天短卡片
- [x] 阶段3 MySQL 资产/播放 + 共享
- [x] 阶段4 小程序
- [x] `go test`（story/chat/manager）/ CHANGELOG / DOC_SYNC

## 测试

- 域单测：阈值判断、进度、shareable、短卡片 Extra。
- Manager：asset upsert / shareable 查询。
- 回归：`internal/domain/story`、`child_story_*`。

## 改动记录

| 日期 | 摘要 |
|------|------|
| 2026-07-14 | 未完整生成不入共享池；播报进度/完播态与管理端对话短卡片见 `story-playback-ux-fix` |
| 2026-07-14 | 意图 LLM 纠出 `canonical`（如后裔射太阳→后羿射日）再查共享池；`theme_raw` 入别名 |
| 2026-07-14 | 方案 A：双池 + 轻量别名复用（named/open/bedtime、排斥、Top-K） |
| 2026-07-14 | 立项并按 ADR 分阶段实现落地 |
