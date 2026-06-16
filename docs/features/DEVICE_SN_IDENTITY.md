# 设备唯一标识改用 SN

## 概述

设备唯一标识（Device-ID）从 MAC 切换为 **SN（序列号）**，存储于 `devices.device_name`。固件 OTA/WebSocket/MQTT 的 `Device-Id`、小程序 BLE 绑定、管理端预登记均使用 SN。

**不做 MAC 存量兼容**；已用 MAC 预登记的设备需人工迁移 `device_name`。

## 标识链路

| 环节 | 字段 | 说明 |
|------|------|------|
| 管理端预登记 | `device_name` | 填 SN |
| BLE Notify | `sn` 或 `serial_number` | 固件 ForceGetSSID 响应 |
| 小程序 check/bind | `?sn=` / `{ "sn" }` | 查 `device_name` |
| 固件联网 | `Device-Id` Header | SN |
| OTA 激活查询 | 请求体 `board.sn`，缺省时用 Header `Device-Id` | 与 `device_name` 对齐 |
| 设备激活 | `device_id` + `serial_number` | 二者必须一致且等于 `device_name` |
| MQTT clientId | `GID_xxx@@@{sn}@@@{uuid}` | 中间段为 SN |

## BLE 协议变更

ForceGetSSID 响应除 WiFi 列表外，设备需上报 SN：

```json
{"sn":"SN-XXXXXXXX-XXXXXXXX"}
```

也兼容 `serial_number` / `serialNumber` 字段名。不再使用 `ble_mac` 或 MAC 作为设备标识。

## 小程序 API

| 方法 | 路径 | 参数 |
|------|------|------|
| GET | `/api/mp/devices/check` | `?sn=` |
| POST | `/api/mp/devices/bind` | `{ "sn", "child_nick_name" }` |

## 管理端绑定

Web 控制台绑定设备时，除 6 位验证码外，可填 SN（JSON 字段 `sn`）。

## 数据迁移

```sql
UPDATE devices SET device_name = '<SN>' WHERE device_name = '<OLD_MAC>';
```

## 联调前提

固件需同步发布：

1. BLE Notify 返回 `sn`
2. OTA/WebSocket `Device-Id` 使用 SN
3. MQTT topic 与 clientId 中间段使用 SN

## 相关文件

- `manager/backend/controllers/mp_device.go` — 小程序 check/bind
- `manager/miniprogram/utils/ble.js` — BLE 解析 SN
- `manager/frontend/src/components/common/DeviceForm.vue` — 预登记 SN
- `internal/app/mqtt_server/device_hook.go` — MQTT 设备标识解析
