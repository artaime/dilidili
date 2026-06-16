package websocket

import (
	"dili-esp32-server-golang/internal/ota/aitoy"
)

func isAIToyOTARequest(req *OtaRequest, userAgent string) bool {
	if req == nil {
		return false
	}
	return aitoy.IsAIToyDevice(aitoy.DeviceReport{
		BoardType:      req.Board.Type,
		ApplicationVer: req.Application.Version,
		UserAgent:      userAgent,
	})
}

func buildAIToyFirmware(req *OtaRequest) FirmwareInfo {
	report := aitoy.DeviceReport{ApplicationVer: ""}
	if req != nil {
		report.ApplicationVer = req.Application.Version
	}
	fw := aitoy.BuildFirmware(report)
	if len(fw.Subs) == 0 {
		return FirmwareInfo{}
	}
	subs := make(map[string]FirmwareSubPackage, len(fw.Subs))
	for k, v := range fw.Subs {
		subs[k] = FirmwareSubPackage(v)
	}
	return FirmwareInfo{Subs: subs}
}
