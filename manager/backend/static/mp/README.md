# 小程序公开静态资源

由 `go:embed` 嵌入，对外路径前缀：`/static/mp/`。

**阈值**：单文件 **> 50KB** 才放本目录网络下发；≤50KB 打进 `manager/miniprogram/assets/`。  
例外：首页 / 留言 /「我的」/ 蓝牙绑定等大体积 GIF·背景固定网络下发（勿打进包）。  
规范：`.cursor/rules/05-miniprogram-assets.mdc`、`docs/features/mp-mascot-remote/FEATURE.md`。

| 文件 | URL | 说明 |
|------|-----|------|
| `home-bg.png` | `/static/mp/home-bg.png` | 首页英雄区背景 |
| `dili-flower.gif` | `/static/mp/dili-flower.gif` | 首页右侧吉祥物 GIF（透明底） |
| `dili-voice-record.gif` | `/static/mp/dili-voice-record.gif` | 新建留言页：语音模式英雄区动效 |
| `dili-type-record.gif` | `/static/mp/dili-type-record.gif` | 新建留言页：文字模式英雄区动效 |
| `voice-ball.gif` | `/static/mp/voice-ball.gif` | 语音球活跃动效（录音 / 试听 / 列表播放） |
| `voice-ball.png` | `/static/mp/voice-ball.png` | 语音球静帧（空闲 / 播放结束） |
| `dili-screen.gif` | `/static/mp/dili-screen.gif` | 「我的」页顶部吉祥物 |
| `dili-ble-connect.gif` | `/static/mp/dili-ble-connect.gif` | 设备蓝牙绑定页顶部吉祥物 |
| `ble-ball.gif` | `/static/mp/ble-ball.gif` | 设备蓝牙绑定页扫描动效（扫描中） |
| `ble-ball.png` | `/static/mp/ble-ball.png` | 设备蓝牙绑定页扫描球静帧（未扫描 / 扫描结束） |
| `devices-empty-mascot.svg` | `/static/mp/devices-empty-mascot.svg` | 「我的设备」无设备空态插画 |
| `create-orb.png` | （不嵌入、不公开） | 旧语音音球备份 |
| `profile-mascot.svg` | （不嵌入、不公开） | 旧「我的」吉祥物备份 |
| `thankful_plush_source.mp4` | （不嵌入、不公开） | 历史 WebP 导出用白底源片 |
| `record.webp` | （不嵌入、不公开） | 旧留言页动效备份，已由 voice/type GIF 替代 |

替换首页背景：覆盖 `home-bg.png` 后发版 manager 后端。  
替换首页吉祥物：覆盖 `dili-flower.gif` 后发版。  
替换留言 /「我的」/ 蓝牙动效：覆盖对应 GIF 后发版。
