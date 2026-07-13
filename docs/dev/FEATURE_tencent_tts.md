# 腾讯 TTS（tencent_tts）

## 背景

新增腾讯云**流式文本语音合成 v2**（`stream_wsv2`）作为 TTS provider，支持管理台配置与 LLM 双流式（`double_stream: true`）边出字边合成。

## 前置条件

1. 在 [腾讯云语音合成控制台](https://console.cloud.tencent.com/tts) 开通服务
2. 在 [API 密钥管理](https://console.cloud.tencent.com/cam/capi) 获取 `SecretId`、`SecretKey`
3. 在账号信息页查看 `AppId`

## 配置字段（json_data）

| 字段 | 必填 | 说明 |
|------|------|------|
| `provider` | 是 | 固定 `tencent_tts` |
| `app_id` | 是 | 腾讯云 AppId（整数） |
| `secret_id` | 是 | API SecretId |
| `secret_key` | 是 | API SecretKey |
| `voice_type` | 是 | 音色 ID，见[音色列表](https://cloud.tencent.com/document/product/1073/92668) |
| `codec` | 否 | `pcm`（默认）或 `mp3` |
| `sample_rate` | 否 | `8000` / `16000`（默认） / `24000` |
| `speed` | 否 | 语速，范围 `[-2, 6]`，默认 `0` |
| `volume` | 否 | 音量，范围 `[-10, 10]`，默认 `0` |
| `ws_url` | 否 | 默认 `wss://tts.cloud.tencent.com/stream_wsv2` |
| `double_stream` | 否 | `true` 时启用 LLM 双流式合成 |
| `frame_duration` | 否 | Opus 帧时长（毫秒），默认 `60` |

## 管理台使用

1. 管理台 → **TTS 配置** → 添加配置
2. 提供商选择「腾讯」，填写 AppId / SecretId / SecretKey / 音色
3. 需要 LLM 流式跟读时开启「双流式」
4. 保存后可点「测试」验证合成链路

## 实现位置

- Provider：`internal/domain/tts/tencent/tencent_tts.go`
- 工厂注册：`internal/domain/tts/base.go`
- 前端表单：`manager/frontend/src/views/admin/forms/TTSConfigForm.vue`

## 限制

- 单会话总字数不超过 10000 字
- 流式文本不支持 SSML
- 文本需含正确标点，否则可能长时间缓存未合成
- 默认并发 20 路（与其他腾讯 TTS 接口共用）
