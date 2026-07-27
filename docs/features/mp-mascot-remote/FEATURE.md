# 小程序素材网络下发

## 状态

done

## 需求

- 背景：控制小程序包体积；大素材由 manager 托管。
- **阈值（规范）**：图片 / 图标 / 视频 / GIF / WebP 等单文件 **> 50KB** 才网络下发；≤50KB 打进包。
- Cursor 规则：`.cursor/rules/05-miniprogram-assets.mdc`（编辑 miniprogram / `static/mp` 时自动加载）

## 当前公开静态（>50KB）

| URL | 用途 |
|-----|------|
| `GET /static/mp/home-bg.png` | 首页英雄区背景（**固定远端**） |
| `GET /static/mp/dili-flower.gif` | 首页右侧吉祥物 GIF（**固定远端**） |
| `GET /static/mp/dili-voice-record.gif` | 新建留言页：语音模式英雄区动效（**固定远端**） |
| `GET /static/mp/dili-type-record.gif` | 新建留言页：文字模式英雄区动效（**固定远端**） |
| `GET /static/mp/voice-ball.gif` | 新建留言 / 列表：语音球活跃动效（录音、试听、播放中）（**固定远端**） |
| `GET /static/mp/voice-ball.png` | 新建留言 / 列表：语音球静帧（空闲、试听/播放结束）（**固定远端**） |
| `GET /static/mp/dili-screen.gif` | 「我的」页顶部吉祥物（**固定远端**） |
| `GET /static/mp/dili-ble-connect.gif` | 设备蓝牙绑定页顶部吉祥物（**固定远端**） |
| `GET /static/mp/ble-ball.gif` | 设备蓝牙绑定页中间扫描动效（扫描中）（**固定远端**） |
| `GET /static/mp/ble-ball.png` | 设备蓝牙绑定页中间扫描球静帧（未扫描 / 扫描结束）（**固定远端**） |
| `GET /static/mp/devices-empty-mascot.svg` | 「我的设备」无设备空态插画 |

包内（≤50KB）：顶栏 `logo-face.svg`、登录 `brand.svg`、其余小图标 SVG。

## 设计

- 后端：`manager/backend/static/mp` + `embed_mp.go` + `router` `/static/mp`
- 小程序：`utils/assets.js`（`REMOTE_ASSET_THRESHOLD_BYTES = 50 * 1024`）
- 首页：`home-bg.png` 铺满英雄区；右侧 `<image>` 播 `dili-flower.gif`；logo + 问候语叠在左侧
- 留言页：语音模式英雄区播 `dili-voice-record.gif`，文字模式播 `dili-type-record.gif`；语音中间球空闲/结束用 `voice-ball.png`，录音/试听/列表播放用 `voice-ball.gif`；失败回退包内 logo / 本地音球
- 「我的」页顶部：`dili-screen.gif`；设备蓝牙绑定：顶部 `dili-ble-connect.gif`、中间扫描球空闲用 `ble-ball.png`、扫描中用 `ble-ball.gif`；失败回退包内 logo / 本地扫描球
- 绑定页进入后不自动扫描，需用户点击「扫描绑定设备」

## 开发计划

- [x] 实现
- [x] 写入 Cursor 规则
- [x] CHANGELOG

## 改动记录

| 日期 | 摘要 |
|------|------|
| 2026-07-22 | 首页视频 / 我的 SVG / >10KB 迁出 |
| 2026-07-22 | 阈值改为 **50KB**；≤50KB 回包；规则 `05-miniprogram-assets.mdc` |
| 2026-07-22 | `thankful_plush.mp4` 固定网络下发（后续正式片体积大） |
| 2026-07-22 | 新增 `create-orb.png`（新建留言页音球） |
| 2026-07-23 | 透明 animated WebP；后升至 960² / 16fps |
| 2026-07-23 | 首页改为 `home.mp4` 头部全幅视频（logo/问候语叠加）；WebP 改名 `record.webp` 用于留言页 |
| 2026-07-23 | 首页改回「背景图 + GIF 吉祥物」：`home-bg.png` + `dili-flower.gif`，去掉 `home.mp4` |
| 2026-07-23 | 留言页按模式切换动效：语音 `dili-voice-record.gif`、文字 `dili-type-record.gif`（替换原 `record.webp`） |
| 2026-07-23 | 「我的」页 / 设备绑定页改用 `ble-connect.gif`（替换 `profile-mascot.svg` 与包内 `hero-mascot.png`） |
| 2026-07-24 | 语音中间球 `dili-voice-ball.gif`；「我的」顶部 `dili-screen.gif`；蓝牙页 `dili-ble-ball.gif` |
| 2026-07-24 | 蓝牙页拆分：顶部 `dili-ble-connect.gif`、中间扫描 `ble-ball.gif`；语音球文件改名 `voice-ball.gif` |
| 2026-07-24 | 语音球 / 扫描球增加静帧 PNG；活跃态才播 GIF；绑定页取消进入自动扫描。详见 `docs/features/mp-ball-still-anim/FEATURE.md` |
