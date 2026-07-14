# 管理端设备故事查看

## 状态

done

## 需求

- 管理端设备管理增加「故事」子页，查看设备在 Redis Story Memory 中的故事列表与详情
- 展示：标题、播放进度（百分比）、字数、题材、风格、年龄段、播放状态、播放次数、时间

## 设计

- 后端：`GET /api/admin/devices/:id/stories`、`GET /api/admin/devices/:id/stories/:storyId`
- 删除：`DELETE .../stories/:storyId`（单条）、`DELETE .../stories`（清空）；只清本机 playback + Redis，不删 `story_assets`
- 服务：`manager/backend/services/device_story`，只读连接与主服务相同的 Redis（`config.json` → `redis`）
- 设备 SN（`device_name`）作为 Story Store 的 deviceID
- 前端：`DeviceStories.vue`，从设备列表「故事」按钮进入

## 配置

管理端 `manager/backend/config/config.json`：

```json
"redis": {
  "host": "127.0.0.1",
  "port": 6379,
  "password": "",
  "db": 1,
  "key_prefix": "dilidili"
}
```

须与主服务 `config.yaml` / `config.pro.yaml` 中 `redis.key_prefix` **完全一致**（生产环境当前为 `dili`）。

## 测试

```bash
go test ./internal/domain/story/... -run Playback
cd manager/backend && go build ./...
```

## 改动记录

| 日期 | 摘要 |
|------|------|
| 2026-07-14 | 进度仅展示百分比（去掉「y/x 段」） |
| 2026-06-30 | 管理端设备故事列表与详情页 |
