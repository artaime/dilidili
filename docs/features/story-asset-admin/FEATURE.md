# 管理端故事资产库

## 状态

done

## 需求

- 背景：共享故事正文落在 MySQL `story_assets`，此前只能用设备故事页或 SQL 查看，无法运营维护。
- 验收标准：
  1. 管理端侧栏进入「故事管理」，可按池/关键词分页列表。
  2. 支持新增、编辑、删除资产；可标记入 named/open/bedtime 池。
  3. 支持「AI 生成」：选用系统 LLM 配置按主题生成正文，填入表单后可再编辑保存。
  4. 仅 admin 可访问。

## 设计

- 影响模块：`manager/backend/services/story_persist`、`controllers`、admin 前端。
- API：
  - `GET/POST /api/admin/story-assets`
  - `GET/PUT/DELETE /api/admin/story-assets/:storyId`
  - `POST /api/admin/story-assets/generate`

## 开发计划

- [x] FEATURE
- [x] Manager List/Delete/Admin CRUD + AI generate
- [x] StoryAssets.vue + 路由侧栏
- [x] 单测 + CHANGELOG

## 测试

- `story_persist`：List/Delete
- 手工：管理台新建/AI 生成/编辑/删除

## 改动记录

| 日期 | 摘要 |
|------|------|
| 2026-07-14 | 管理端故事资产 CRUD + AI 新增 |
