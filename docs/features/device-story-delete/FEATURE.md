# 管理端设备故事删除 / 清空

## 状态

done

## 需求

- 背景：设备故事页仅只读；运营需清除本机播放记录，但不能破坏共享故事库。
- 验收标准：
  1. 可删除单条设备故事（本机进度）。
  2. 可清空该设备全部故事记录。
  3. 仅删 Redis 设备键 + `story_playbacks`；**不删** `story_assets` / 别名。
  4. 二次确认；提示不影响共享故事库。

## 设计

- API：
  - `DELETE /api/admin/devices/:id/stories/:storyId`
  - `DELETE /api/admin/devices/:id/stories`（清空）
- 服务：`device_story` 双删（MySQL 优先保证列表消失，Redis best-effort）

## 开发计划

- [x] FEATURE + 实现 + 前端 + CHANGELOG

## 改动记录

| 日期 | 摘要 |
|------|------|
| 2026-07-14 | 设备故事单删 / 清空落地 |
