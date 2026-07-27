# 小程序语音球 / 扫描球静动切换

## 状态

done

## 需求

- 背景：`voice-ball.gif` / `ble-ball.gif` 体积大，空闲态持续播放浪费流量与性能；新增静态帧 PNG，仅在活跃态播 GIF。
- 验收标准：
  1. **新建语音留言**：未录音显示 `voice-ball.png`；录音中显示 GIF；试听中显示 GIF；试听结束回静态图。
  2. **留言列表语音卡**：未播放显示静态图；播放中显示 GIF；播放结束回静态图。
  3. **绑定设备**：进入页面不自动扫描；未扫描显示 `ble-ball.png`；点击扫描后显示 GIF；扫描结束回静态图。

## 设计

- 影响模块：`manager/miniprogram`（留言创建/列表、蓝牙绑定）、`utils/assets.js`、`manager/backend/static` embed
- 新增远端素材（均 >50KB）：`/static/mp/voice-ball.png`、`/static/mp/ble-ball.png`
- API/配置变更：无

## 开发计划

- [x] 实现静/动资源与页面状态切换
- [x] 取消绑定页 `onShow` 自动扫描，文案同步
- [x] `go test ./...`
- [x] CHANGELOG + 素材文档同步

## 测试

- 真机：留言页录音/试听球切换；列表播放切换；绑定页手动扫描与球切换。

## 改动记录

| 日期 | 摘要 |
|------|------|
| 2026-07-24 | 立项并落地：静帧 PNG + 活跃态 GIF；绑定页取消默认扫描 |
