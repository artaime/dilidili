# 家长留言小程序

## 概述

为家长提供微信小程序，支持微信登录、BLE/手动 MAC 绑定儿童设备、发送语音或文字留言。儿童设备开机联网后，主服务检测待播放留言，TTS 询问是否收听，经 ASR 确认后 TTS 播报留言内容。

## 范围

- 小程序端：`manager/miniprogram/`（独立子模块仓库 `ssh://git@110.87.103.170:6122/xiaodi/dilidili_mp.git`）
- 管理 API 扩展：`manager/backend/` `/api/mp/*`
- 主服务留言投递：`internal/app/server/chat/parent_message.go`
- 内部 API：`/api/internal/devices/:device_name/parent-messages/pending`

## 用户流程

1. 管理员在控制台预录入设备 MAC（`device_name`）
2. 家长微信登录小程序
3. 通过 BLE 或手动输入 MAC 绑定设备，自动创建默认儿童智能体
4. 家长发送文字或录音留言（录音 MVP 经 ASR 转写后以 TTS 播报）
5. 设备上线后，主服务 TTS 询问「要听吗？」→ ASR 识别 → 播放或跳过

## API

### 公开

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/mp/auth/login` | 微信 code 登录 |

### 需 JWT

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/mp/profile` | 家长资料 |
| GET | `/api/mp/devices/check` | 查询设备是否可绑定 |
| POST | `/api/mp/devices/bind` | 绑定设备 |
| GET | `/api/mp/devices` | 已绑定设备 |
| POST | `/api/mp/messages` | 创建留言 |
| GET | `/api/mp/messages` | 留言列表 |
| DELETE | `/api/mp/messages/:id` | 撤回待播放留言 |

### 内部（主服务）

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/internal/devices/:device_name/parent-messages/pending` | 最早 pending 留言 |
| PATCH | `/api/internal/parent-messages/:id/status` | 更新状态 |

## 配置

`manager/backend/config/config.json`：

```json
{
  "wechat": {
    "miniprogram": {
      "app_id": "",
      "app_secret": ""
    }
  },
  "parent_message": {
    "audio_base_path": "./storage/parent_messages/audio",
    "max_file_size": 10485760
  }
}
```

## 数据模型

- `users` 扩展：`wx_openid`、`nickname`、`avatar_url`、`source`
- `parent_messages`：留言记录与状态机

## 状态机

`pending` → `notified` → `played` | `skipped`

## 依赖

- 微信小程序 AppID/Secret
- 管理员预登记设备 MAC
- BLE 协议文档（`doc/ble_protocol.pdf`，二期完善 BLE 页）
