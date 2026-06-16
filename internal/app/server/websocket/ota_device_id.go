package websocket

import "strings"

// resolveOTADeviceID 解析 OTA 用于激活/配置/MQTT 的设备标识。
// 优先 board.sn，其次 Device-Id Header（均为 SN）。
func resolveOTADeviceID(headerDeviceID string, req *OtaRequest) string {
	if req != nil {
		if id := strings.TrimSpace(req.Board.Sn); id != "" {
			return id
		}
	}
	return strings.TrimSpace(headerDeviceID)
}

func resolveActivateDeviceID(headerDeviceID, serialNumber string) string {
	if id := strings.TrimSpace(serialNumber); id != "" {
		return id
	}
	return strings.TrimSpace(headerDeviceID)
}
