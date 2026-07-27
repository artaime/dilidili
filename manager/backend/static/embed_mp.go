package static

import "embed"

// 网络下发的小程序素材（>50KB，或已知将增大的首页/留言动效）。见 05-miniprogram-assets.mdc
//go:embed mp/home-bg.png mp/dili-flower.gif mp/dili-voice-record.gif mp/dili-type-record.gif mp/voice-ball.gif mp/voice-ball.png mp/dili-screen.gif mp/dili-ble-connect.gif mp/ble-ball.gif mp/ble-ball.png mp/devices-empty-mascot.svg
var MpFS embed.FS
