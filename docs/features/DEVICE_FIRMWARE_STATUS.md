# 设备固件状态问答与控制

## 状态

done

## 需求

- 背景：用户会问电量/音量/亮度，也会口头要求调音量、亮度、睡眠、关机。`data_insight` 不含可靠的音量等字段，不能作为问答数据源。
- 验收标准：
  1. 问状态时主对话 LLM 应调用设备 MCP `get_device_status`，答案依据 tool 结果，不依赖 `data_insight`。
  2. 调音量/亮度、睡眠、关机必须调用对应设备 MCP set/动作工具；无 tool call 时能力地面禁止完成态乱答。
  3. 「大一点/小一点」等相对指令：工具描述约定先 get 再按 ±10 计算绝对目标再 set；服务端不做正则意图短路。
  4. 单测覆盖工具描述增强；相关包单测通过。

## 设计

- 对话面：设备 IoT-over-MCP 已暴露的工具进入当轮 LLM tools（现有 `GetToolsByDeviceIdWithTransport` → `ConvertMCPToolsToEinoTools`），在 `ConvertMcpToolListToInvokableToolList` 时对已知固件工具**追加描述引导**。
- 涉及工具（固件协议）：`get_device_status`、`set_speaker_volume`、`set_screen_brightness`、`enter_sleep_mode`、`power_off_device`。
- 相对步长：`DefaultRelativeStep = 10`，写在 tool description；LLM 自行 clamp 0–100。
- 能力地面：有固件工具时追加「须主动 get、禁猜测」规则；无 tool 完成态改写覆盖睡眠/关机话术。
- **不做**：`data_insight` 落库问答、正则/关键词意图短路、本地假查询缓存。

影响模块：`internal/domain/mcp`（`device_firmware_tools.go`）、`internal/domain/llm/common`（地面护栏）、文档。

API/配置变更：无。

## 开发计划

- [x] 描述增强 + 单测
- [x] 能力地面睡眠/关机补强 + 单测
- [x] CHANGELOG / PROJECT_MAP / DOC_SYNC
- [x] 相关包 `go test` 通过

## 测试

- `TestEnrichFirmwareToolDescription`
- `TestConvertMcpToolListEnrichesFirmwareDescriptions`
- `TestBuildCapabilityGroundingPolicyWithFirmwareTools`
- `TestMaybeRewriteUngroundedActionClaim`（关机/睡眠用例）

## 改动记录

| 日期 | 摘要 |
|------|------|
| 2026-07-14 | 主动 MCP get + set；描述增强与地面护栏落地 |
| 2026-07-14 | 修复：工具成功后仍改写「做不到」；set 成功附确认语义 |
| 2026-07-14 | 修复：意图路由新增 device，避免 general 短路阻 MCP |
