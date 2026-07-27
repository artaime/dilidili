# 小程序设备绑定流程

## 概述

家长端 BLE 配网与账号绑定的标准流程。绑定通过 `POST /api/mp/devices/bind` 完成，同时将设备标记为 `activated=true`（不调用 `/api/internal/device/activate`）。

设备唯一标识为 **SN**，存储于 `devices.device_name`。详见 [DEVICE_SN_IDENTITY.md](./DEVICE_SN_IDENTITY.md)。

## 流程时序

1. 扫描并连接 BLE 设备（进入绑定页后点击「扫描绑定设备」触发，结束后可「重新扫描」）
2. 发送 `ForceGetSSID`，获取 `sn` 与 WiFi 列表
3. **绑定（WiFi 之前）**
   - `GET /api/mp/devices/check?sn=` 预检登记与占用状态
   - 弹窗确认 → 填写孩子昵称 → `POST /api/mp/devices/bind`（body 含 `sn`）
4. WiFi 配网（可跳过）：发送 ssid/password，监听 `sta_code` 1→5
5. 配网完成或跳过后，开放音量/亮度系统设置
6. 返回首页，`GET /api/mp/devices` 展示已绑定设备

## 解绑

| 方法 | 路径 | 说明 |
|------|------|------|
| DELETE | `/api/mp/devices/:id` | 出厂重置：删除设备全部数据，清零绑定字段，**保留**出厂 `agent_id`；仅属主可操作 |

解绑会清理：Memobase 记忆、Redis 故事/短期记忆、对话记录、家长留言及音频、**家庭成员与邀请码**；并通过 WebSocket 通知主服务踢线。详见 [DEVICE_UNBIND_RESET.md](./DEVICE_UNBIND_RESET.md)。

小程序「我的设备」页提供二次确认解绑（不可逆删除提示）。属主可改孩子昵称（`PATCH /api/mp/devices/:id`）；家庭成员授权见 [device-family-auth/FEATURE.md](./device-family-auth/FEATURE.md)。

## 家庭成员

首位绑定者为属主。属主可生成邀请码，其他家长登录后通过 `POST /api/mp/devices/join` 加入，无需再次 BLE 绑定。成员可查看设备、发留言；不可解绑、不可改昵称。

## 登录资料

登录页支持 `chooseAvatar` + 昵称输入，随 `POST /api/mp/auth/login` 提交 `nickname`、`avatar_url`。首页与「我的」页展示头像昵称；「我的」页修改后重新登录同步。

## 相关文件

- `manager/miniprogram/pages/ble-connect/` — 扫描、绑定弹窗
- `manager/miniprogram/pages/device-config/` — WiFi 配网、系统设置
- `manager/miniprogram/pages/devices/` — 设备列表与解绑
- `manager/backend/controllers/mp_device.go` — check/bind/list/unbind
