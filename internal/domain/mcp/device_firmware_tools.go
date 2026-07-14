package mcp

import (
	"encoding/json"
	"fmt"
	"strings"
)

// DefaultRelativeStep 相对调节默认步长（音量/亮度「大一点/小一点」）。
// 写进 tool description，由 LLM 先 get 再算绝对值；服务端不做正则意图路由。
const DefaultRelativeStep = 10

// enrichFirmwareToolDescription 对已知设备固件 MCP 工具追加调用引导（查须主动 get，相对调节先 get 再 ±Step）。
// 未知工具原样返回 baseDesc。
func enrichFirmwareToolDescription(toolName, baseDesc string) string {
	key := normalizeFirmwareToolName(toolName)
	guide, ok := firmwareToolGuides[key]
	if !ok {
		return baseDesc
	}
	base := strings.TrimSpace(baseDesc)
	if base == "" {
		return guide
	}
	if strings.Contains(base, guide) {
		return base
	}
	return base + " " + guide
}

func normalizeFirmwareToolName(toolName string) string {
	key := strings.ToLower(strings.TrimSpace(toolName))
	return strings.ReplaceAll(key, "-", "_")
}

var firmwareToolGuides = map[string]string{
	"get_device_status": "用户询问当前音量、电量、亮度或其它本机状态时必须调用本工具获取实时结果；不要猜测或编造数值。调用失败时如实说明暂时读不到。",
	"set_speaker_volume": fmt.Sprintf(
		"用户要求调节喇叭音量时调用。volume 为绝对整数 0-100。用户说「大声一点/小一点」等相对量时：先调用 get_device_status 取当前音量，再按 ±%d 计算目标（夹紧到 0-100）后调用本工具。工具调用成功即表示设置已生效，请简洁确认已调好，禁止再说失败或做不到。",
		DefaultRelativeStep,
	),
	"set_screen_brightness": fmt.Sprintf(
		"用户要求调节屏幕亮度时调用。brightness 为绝对整数 0-100。用户说「亮一点/暗一点」等相对量时：先调用 get_device_status 取当前亮度，再按 ±%d 计算目标（夹紧到 0-100）后调用本工具。工具调用成功即表示设置已生效，请简洁确认已调好，禁止再说失败或做不到。",
		DefaultRelativeStep,
	),
	"enter_sleep_mode": "用户明确要求设备休眠、睡觉、进入睡眠模式时调用。工具调用成功即表示已进入睡眠，请简洁确认；禁止仅口头假装或声称失败。",
	"power_off_device":  "用户明确要求关机、关闭设备电源时调用。工具调用成功即表示已关机，请简洁确认；禁止仅口头假装或声称失败。",
}

// IsFirmwareControlTool 是否为会改变设备状态的固件控制工具（不含 get）。
func IsFirmwareControlTool(toolName string) bool {
	switch normalizeFirmwareToolName(toolName) {
	case "set_speaker_volume", "set_screen_brightness", "enter_sleep_mode", "power_off_device":
		return true
	default:
		return false
	}
}

// AnnotateFirmwareToolSuccess 固件控制类工具调用成功时，附加明确成功语义供 LLM 确认。
func AnnotateFirmwareToolSuccess(toolName, argumentsJSON, rawResult string) string {
	key := normalizeFirmwareToolName(toolName)
	if !IsFirmwareControlTool(key) {
		return rawResult
	}
	msg := firmwareSuccessHint(key, argumentsJSON)
	raw := strings.TrimSpace(rawResult)
	if raw == "" {
		return msg
	}
	return msg + " 设备原始返回: " + raw
}

func firmwareSuccessHint(toolName, argumentsJSON string) string {
	switch toolName {
	case "set_speaker_volume":
		if v := extractIntArg(argumentsJSON, "volume"); v >= 0 {
			return fmt.Sprintf("成功：音量已设置为 %d。请用一句话简洁告诉用户已调好。", v)
		}
		return "成功：音量已设置。请用一句话简洁告诉用户已调好。"
	case "set_screen_brightness":
		if v := extractIntArg(argumentsJSON, "brightness"); v >= 0 {
			return fmt.Sprintf("成功：亮度已设置为 %d。请用一句话简洁告诉用户已调好。", v)
		}
		return "成功：亮度已设置。请用一句话简洁告诉用户已调好。"
	case "enter_sleep_mode":
		return "成功：设备已进入睡眠模式。请用一句话简洁告诉用户。"
	case "power_off_device":
		return "成功：设备已关机。请用一句话简洁告诉用户。"
	default:
		return "成功：操作已完成。"
	}
}

func extractIntArg(argumentsJSON, key string) int {
	argumentsJSON = strings.TrimSpace(argumentsJSON)
	if argumentsJSON == "" {
		return -1
	}
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(argumentsJSON), &m); err != nil {
		return -1
	}
	v, ok := m[key]
	if !ok {
		return -1
	}
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	case int64:
		return int(n)
	case json.Number:
		i, err := n.Int64()
		if err != nil {
			return -1
		}
		return int(i)
	default:
		return -1
	}
}
