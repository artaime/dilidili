# 家长留言小程序

## 概述

为家长提供微信小程序，支持微信登录、BLE/手动 MAC 绑定儿童设备、发送语音或文字留言。儿童设备开机联网后，主服务检测待播放留言，以 Agent 人设友好询问是否收听；LLM 识别儿童意图后，语音留言播放家长原声，文字留言 TTS 播报。

## 范围

- 小程序端：`manager/miniprogram/`（独立子模块仓库 `ssh://git@110.87.103.170:6122/xiaodi/dilidili_mp.git`）
- 管理 API 扩展：`manager/backend/` `/api/mp/*`
- 主服务留言投递：`internal/app/server/chat/parent_message*.go`
- 内部 API：`/api/internal/devices/:device_name/parent-messages/pending`

## 用户流程

1. 管理员在控制台预录入设备 MAC（`device_name`）
2. 家长微信登录小程序，填写头像昵称并选择亲属角色（爸爸/妈妈/爷爷/奶奶等）
3. BLE 连接获取 MAC 后**先绑定账号**，再 WiFi 配网，最后调节音量/亮度
4. 家长发送留言：
   - **语音（默认）**：按住说话，松手填标题（可留空自动生成），上滑取消；保存原声 mp3（最长 60 秒）
   - **文字**：输入净化后的文本（过滤 emoji/特殊符号），可选标题
5. 设备上线后，主服务拉取全部 pending 留言并按 **3 小时间隔规则** 编排播放

详见 [MP_BIND_FLOW.md](./MP_BIND_FLOW.md)。

## 设备端播放规则

设 pending 列表按时间升序为 `M[0..n-1]`：

1. **首条或与前一条间隔 >3 小时**：TTS 友好询问（含亲属角色 + 儿童向时间描述），LLM 判断 play/skip
2. **play** → 语音播原声 / 文字 TTS → 标记 `played`
3. **skip** → 标记 `skipped`，结束本次会话（剩余 pending 留待下次开机）
4. 若下一条与上一条间隔 ≤3 小时：TTS 过渡语后直接播放，不再询问
5. 若间隔 >3 小时：回到步骤 1 再次询问

时间描述示例：「今天/昨天/前天 + 早上/中午/傍晚/晚上 + HH点MM分」

### 固件协议（AI 玩具 / pangdou-toy）

固件 MQTT 协议见 [AI玩具协议.md](../../AI玩具协议.md)，**不含** `speak_request` / `speak_ready`。主服务默认 `chat.speak_request_enabled: false`，家长留言主动播报走 **hello → session → UDP → tts** 标准下行，与固件「讲个故事」等场景一致。

## API

### 公开

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/mp/auth/login` | 微信 code 登录（可选 `nickname`、`avatar_url`、`family_role`） |

### 需 JWT

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/mp/profile` | 家长资料（含 `family_role`） |
| PATCH | `/api/mp/profile` | 更新昵称 / 亲属角色 |
| GET | `/api/mp/devices/check` | 查询设备是否可绑定 |
| POST | `/api/mp/devices/bind` | 绑定设备 |
| GET | `/api/mp/devices` | 已绑定设备 |
| DELETE | `/api/mp/devices/:id` | 解绑设备 |
| POST | `/api/mp/messages` | 创建留言（JSON 文字 / multipart 语音） |
| GET | `/api/mp/messages` | 留言列表 |
| GET | `/api/mp/messages/:id/audio` | 流式返回本人语音留言 mp3 |
| DELETE | `/api/mp/messages/:id` | 撤回待播放留言 |

### 内部（主服务）

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/internal/devices/:device_name/parent-messages/pending` | 全部 pending 留言（含 `family_role`、`audio_url`） |
| GET | `/api/internal/parent-messages/:id/audio` | 流式返回语音留言 mp3 |
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

- `users` 扩展：`wx_openid`、`nickname`、`avatar_url`、`family_role`、`source`
- `parent_messages`：`title`、`text_content`（文字必填，语音可空）、`audio_path`、`audio_duration_sec`、`source_type`、状态机

## 状态机

`pending` → `notified` → `played` | `skipped`

## 依赖

- 微信小程序 AppID/Secret
- 管理员预登记设备 MAC
- 主服务与 Manager 内网可达（语音原声通过 internal audio API 拉取）
