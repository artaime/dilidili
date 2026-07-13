# 腾讯 ASR（tencent_asr）

## 背景

新增腾讯云**实时语音识别 WebSocket v2**（`wss://asr.cloud.tencent.com/asr/v2/{appid}`）作为 ASR provider，支持流式识别与管理台配置测试。

## 前置条件

1. 在 [腾讯云语音识别控制台](https://console.cloud.tencent.com/asr) 开通服务
2. 在 [API 密钥管理](https://console.cloud.tencent.com/cam/capi) 获取 `SecretId`、`SecretKey`
3. 在账号信息页查看 `AppId`（可与 TTS 共用凭证，但 ASR 需单独开通）

## 配置字段（json_data）

| 字段 | 必填 | 说明 |
|------|------|------|
| `provider` | 是 | 固定 `tencent_asr` |
| `app_id` | 是 | 腾讯云 AppId（整数） |
| `secret_id` | 是 | API SecretId |
| `secret_key` | 是 | API SecretKey |
| `engine_model_type` | 是 | 引擎模型，默认 `16k_zh`（中文通用） |
| `voice_format` | 否 | 音频格式，`1`=PCM（默认） |
| `sample_rate` | 否 | `8000` / `16000`（默认） |
| `needvad` | 否 | 是否开启 VAD，`0`（默认）/ `1` |
| `filter_dirty` | 否 | 过滤脏词，`0`（默认）/ `1` |
| `filter_modal` | 否 | 过滤语气词，`0`（默认）/ `1` |
| `filter_punc` | 否 | 过滤句末句号，`0`（默认）/ `1` |
| `convert_num_mode` | 否 | 数字转换模式，默认 `1` |
| `timeout` | 否 | 超时秒数，默认 `30` |

## 管理台使用

1. 管理台 → **ASR 配置** → 添加配置
2. 提供商选择「腾讯」，填写 AppId / SecretId / SecretKey / 引擎模型
3. 保存后可点「测试」验证识别链路
4. 在 Agent 或设备配置中绑定该 ASR 配置

## 实现位置

- Provider：`internal/domain/asr/tencent/`
- 适配器：`internal/domain/asr/tencent_adapter.go`
- 工厂注册：`internal/domain/asr/base.go`
- 前端表单：`manager/frontend/src/views/admin/forms/ASRConfigForm.vue`

## 协议要点

- 连接 URL 含 HMAC-SHA1 签名（签名原文不含 `wss://` 前缀）
- 音频以 binary PCM 帧上传，结束发送 `{"type":"end"}` 文本帧
- `slice_type=1/2` 为 partial 结果，`final=1` 表示会话结束

## 限制

- 建议按约 200ms/包发送 PCM，避免超实时率或间隔过长导致断连
- 主服务输入为 16k float32 PCM；8k 场景可配 `sample_rate=8000`
- 密钥仅存 DB `json_data`，勿写入 yaml 或提交 git
