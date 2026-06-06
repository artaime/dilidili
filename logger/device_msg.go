package logger

import (
	"encoding/json"
	"fmt"
	"strings"
)

// SummarizeDevicePayload 提取 JSON 消息类型与关键字段，避免日志过长或泄露密钥。
func SummarizeDevicePayload(payload []byte) string {
	if len(payload) == 0 {
		return "<empty>"
	}
	var root map[string]interface{}
	if err := json.Unmarshal(payload, &root); err != nil {
		if len(payload) > 200 {
			return string(payload[:200]) + "..."
		}
		return string(payload)
	}
	parts := make([]string, 0, 8)
	if t, ok := root["type"].(string); ok && t != "" {
		parts = append(parts, "type="+t)
	}
	if sid, ok := root["session_id"].(string); ok && sid != "" {
		if len(sid) > 12 {
			parts = append(parts, "session_id="+sid[:8]+"...")
		} else {
			parts = append(parts, "session_id="+sid)
		}
	}
	if transport, ok := root["transport"].(string); ok && transport != "" {
		parts = append(parts, "transport="+transport)
	}
	if state, ok := root["state"].(string); ok && state != "" {
		parts = append(parts, "state="+state)
	}
	if mode, ok := root["mode"].(string); ok && mode != "" {
		parts = append(parts, "mode="+mode)
	}
	if udp, ok := root["udp"].(map[string]interface{}); ok {
		server, _ := udp["server"].(string)
		port, _ := udp["port"].(float64)
		enc, _ := udp["encryption"].(string)
		if server != "" {
			parts = append(parts, "udp="+server)
		}
		if port > 0 {
			parts = append(parts, "udp_port="+jsonNumber(port))
		}
		if enc != "" {
			parts = append(parts, "encryption="+enc)
		}
		if _, hasKey := udp["key"]; hasKey {
			parts = append(parts, "udp_key=<redacted>")
		}
	}
	if payloadObj, ok := root["payload"].(map[string]interface{}); ok {
		if method, ok := payloadObj["method"].(string); ok && method != "" {
			parts = append(parts, "mcp.method="+method)
		}
		if id := payloadObj["id"]; id != nil {
			parts = append(parts, "mcp.id="+jsonNumber(id))
		}
		if result, ok := payloadObj["result"].(map[string]interface{}); ok {
			if tools, ok := result["tools"].([]interface{}); ok {
				parts = append(parts, "mcp.tools="+jsonNumber(float64(len(tools))))
			}
		}
	}
	if len(parts) == 0 {
		if len(payload) > 160 {
			return string(payload[:160]) + "..."
		}
		return string(payload)
	}
	return strings.Join(parts, " ")
}

func jsonNumber(v interface{}) string {
	switch n := v.(type) {
	case float64:
		return fmt.Sprintf("%g", n)
	case int:
		return fmt.Sprintf("%d", n)
	case int64:
		return fmt.Sprintf("%d", n)
	default:
		return fmt.Sprintf("%v", v)
	}
}
