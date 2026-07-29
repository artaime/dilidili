# 管理端设备管理：置顶 + 筛选 + 智能体统计

## 状态

done

## 需求

- 背景：管理端设备列表仅有简单绑定状态筛选，管理员调试本人设备时难以快速定位；也无法按智能体总览设备规模与在线情况。千级设备时全量拉取不可行。
- 验收标准：
  1. 当前登录管理员有绑定设备时，这些设备在筛选结果中置顶，并有「我的」标识。
  2. 支持按设备 ID（`device_name` / 数字 `id`）、昵称、绑定用户、激活状态、关联智能体筛选（条件 AND）。
  3. 顶部展示各智能体（含未分配）的关联设备数、激活数、在线数；点击可筛选该智能体，再点取消。
  4. 列表后端分页（`page` / `page_size`），筛选在服务端执行。
  5. 原有编辑 / 出厂重置 / MCP / 隐私门禁行为不变。

## 设计

- 影响模块：
  - 后端：`DeviceService.ListPaged`、`GET /api/admin/devices`、`GET /api/admin/devices/:id`
  - 前端：`Devices.vue`；故事/记忆/对话页改用单设备 GET 拉元信息
- API/配置变更：
  - `GET /api/admin/devices` 查询参数：`page`、`page_size`、`device_id`、`nick_name`、`bind_user`、`activated`、`agent_id`
  - 响应：`{ data: { items, total, page, page_size, agent_stats } }`
  - `agent_stats` 基于全量设备聚合（不受当前筛选影响）
  - 新增 `GET /api/admin/devices/:id` 返回单设备
- 在线判定：`last_active_at` 在 5 分钟内
- 用户端 `GET /api/user/devices` 仍全量（本人设备规模小）

## 开发计划

- [x] FEATURE
- [x] 实现
- [x] `go test`（controllers 设备列表）
- [x] CHANGELOG + PROJECT_MAP

## 测试

- `TestDeviceServiceListPaged`：置顶、device_id / bind_user / agent+activated / 未分配筛选
- 管理端列表：分页翻页、筛选查询、智能体芯片筛选

## 改动记录

| 日期 | 摘要 |
|------|------|
| 2026-07-29 | 初稿：前端置顶 + 多条件筛选 + 智能体统计芯片 |
| 2026-07-29 | 实现完成（纯前端） |
| 2026-07-29 | 升级为后端分页筛选 + agent_stats；新增单设备 GET |
| 2026-07-29 | 加固进入页/重置筛选时本人设备置顶 |
