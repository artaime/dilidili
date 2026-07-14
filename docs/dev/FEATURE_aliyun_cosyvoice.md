# 百炼 CosyVoice TTS（aliyun_cosyvoice）

## 背景

新增阿里云百炼官方 **CosyVoice HTTP SpeechSynthesizer** 作为 TTS provider，支持管理台配置、配置测试与设备端流式播放。

与现有 `cosyvoice`（linkerai HTTP 网关）区分：本 provider 使用 DashScope API Key + POST JSON + SSE 流式。

## 前置条件

1. 在 [阿里云百炼控制台](https://dashscope.console.aliyun.com/) 开通 CosyVoice 语音合成
2. 创建 **API Key**（`sk-xxx`）
3. 获取 **业务空间 Workspace ID**（或使用完整 `api_url`）
4. 确认音色与模型版本匹配（见 [CosyVoice 音色列表](https://help.aliyun.com/zh/model-studio/cosyvoice-tts-http-api)）

## 配置字段（json_data）

| 字段 | 必填 | 说明 |
|------|------|------|
| `provider` | 是 | 固定 `aliyun_cosyvoice` |
| `api_key` | 是 | DashScope API Key |
| `workspace_id` | 二选一 | 业务空间 ID，用于拼默认 endpoint |
| `api_url` | 二选一 | 完整 endpoint，如 `https://{WorkspaceId}.cn-beijing.maas.aliyuncs.com/api/v1/services/audio/tts/SpeechSynthesizer` |
| `model` | 是 | 如 `cosyvoice-v3-flash`、`cosyvoice-v3-plus` |
| `voice` | 是 | 系统音色或复刻音色 ID，如 `longjielidou_v3` |
| `format` | 否 | `wav`（默认）/ `pcm` / `mp3` |
| `sample_rate` | 否 | 默认 `24000` |
| `stream` | 否 | 默认 `true`，主链路使用流式 SSE |
| `instruction` | 否 | 方言/风格指令（官方 `instruction` 字段） |
| `frame_duration` | 否 | Opus 帧时长（毫秒），默认 `60` |

## 管理台使用

1. 管理台 → **TTS 配置** → 添加配置
2. 提供商选择 **「百炼 CosyVoice」**（不是「CosyVoice (linkerai)”）
3. 填写 API Key、Workspace ID（或 API URL）、模型、音色（下拉可选，随模型变化）
4. 保存后点「测试」验证合成链路

智能体配置 TTS 时，选择百炼 CosyVoice 配置后可在「TTS 音色」下拉中选择对应模型的系统音色。

## 实现位置

- Provider：`internal/domain/tts/aliyun_cosyvoice/cosyvoice.go`
- 工厂注册：`internal/domain/tts/base.go`
- 前端表单：`manager/frontend/src/views/admin/forms/TTSConfigForm.vue`

## 限制

- 首期仅支持北京地域 MaaS endpoint（`cn-beijing.maas.aliyuncs.com`）
- 不支持 LLM 双流式（`double_stream`）
- 声音复刻（`customization` API）留第二期
