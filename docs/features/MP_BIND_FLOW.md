# 小程序设备绑定流程

## 概述

家长端 BLE 配网与账号绑定的标准流程。绑定通过 `POST /api/mp/devices/bind` 完成，同时将设备标记为 `activated=true`（不调用 `/api/internal/device/activate`）。

## 流程时序

1. 扫描并连接 BLE 设备
2. 发送 `ForceGetSSID`，获取 `ble_mac` 与 WiFi 列表
3. **绑定（WiFi 之前）**
   - `GET /api/mp/devices/check?mac=` 预检登记与占用状态
   - 弹窗确认 → 填写孩子昵称 → `POST /api/mp/devices/bind`
4. WiFi 配网（可跳过）：发送 ssid/password，监听 `sta_code` 1→5
5. 配网完成或跳过后，开放音量/亮度系统设置
6. 返回首页，`GET /api/mp/devices` 展示已绑定设备

## 解绑

| 方法 | 路径 | 说明 |
|------|------|------|
| DELETE | `/api/mp/devices/:id` | 软解绑：清零 `user_id`/`agent_id`，`activated=false` |

小程序「我的设备」页提供二次确认解绑。

## 登录资料

登录页支持 `chooseAvatar` + 昵称输入，随 `POST /api/mp/auth/login` 提交 `nickname`、`avatar_url`。首页与「我的」页展示头像昵称；「我的」页修改后重新登录同步。

## 相关文件

- `manager/miniprogram/pages/ble-connect/` — 扫描、绑定弹窗
- `manager/miniprogram/pages/device-config/` — WiFi 配网、系统设置
- `manager/miniprogram/pages/devices/` — 设备列表与解绑
- `manager/backend/controllers/mp_device.go` — check/bind/list/unbind
