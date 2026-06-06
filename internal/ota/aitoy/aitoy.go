package aitoy

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/spf13/viper"
)

const (
	BoardTypePangdou = "pangdou-toy"
	PdVoiceDefault   = 1
)

// FirmwareSubPackage 单个子固件升级包信息。
type FirmwareSubPackage struct {
	Version string `json:"version"`
	Size    int64  `json:"size"`
	Url     string `json:"url"`
}

// FirmwareInfo AIToy OTA firmware 段（仅 subs）。
type FirmwareInfo struct {
	Subs map[string]FirmwareSubPackage `json:"subs,omitempty"`
}

// DeviceReport 设备 OTA 上报摘要。
type DeviceReport struct {
	BoardType      string
	ApplicationVer string
	UserAgent      string
}

var defaultFirmwareTargets = map[string]FirmwareSubPackage{
	"pd-v1-gx8006": {Version: "2.13.0", Size: 1577024, Url: "http://127.0.0.1:8080/ota/firmware/pd-v1-gx8006"},
	"pd-v1-ln882":  {Version: "2.34.0", Size: 669154, Url: "http://127.0.0.1:8080/ota/firmware/pd-v1-ln882"},
	"pd-v1-re3220": {Version: "2.13.0", Size: 1408184, Url: "http://127.0.0.1:8080/ota/firmware/pd-v1-re3220"},
}

func IsAIToyDevice(report DeviceReport) bool {
	ua := strings.TrimSpace(report.UserAgent)
	if strings.HasPrefix(strings.ToLower(ua), "aitoy/") {
		return true
	}
	boardType := strings.TrimSpace(strings.ToLower(report.BoardType))
	if boardType == BoardTypePangdou || strings.HasPrefix(boardType, "pangdou-") {
		return true
	}
	return isMultiSubApplicationVersion(report.ApplicationVer)
}

func BuildFirmware(report DeviceReport) FirmwareInfo {
	current := parseApplicationVersions(report.ApplicationVer)
	targets := loadFirmwareTargets()
	subs := make(map[string]FirmwareSubPackage)
	for name, target := range targets {
		cur := strings.TrimSpace(current[name])
		if cur == "" || CompareSemver(cur, target.Version) < 0 {
			subs[name] = target
		}
	}
	return FirmwareInfo{Subs: subs}
}

func isMultiSubApplicationVersion(version string) bool {
	version = strings.TrimSpace(version)
	if version == "" || !strings.HasPrefix(version, "{") {
		return false
	}
	var subs map[string]string
	if err := json.Unmarshal([]byte(version), &subs); err != nil {
		return false
	}
	for key := range subs {
		if strings.HasPrefix(key, "pd-v1-") {
			return true
		}
	}
	return false
}

func parseApplicationVersions(version string) map[string]string {
	out := make(map[string]string)
	version = strings.TrimSpace(version)
	if version == "" || !strings.HasPrefix(version, "{") {
		return out
	}
	_ = json.Unmarshal([]byte(version), &out)
	return out
}

func loadFirmwareTargets() map[string]FirmwareSubPackage {
	cfg := viper.GetStringMap("ota.aitoy.subs")
	if len(cfg) == 0 {
		out := make(map[string]FirmwareSubPackage, len(defaultFirmwareTargets))
		for k, v := range defaultFirmwareTargets {
			out[k] = v
		}
		return out
	}
	out := make(map[string]FirmwareSubPackage)
	for name, raw := range cfg {
		entry, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		pkg := FirmwareSubPackage{
			Version: asString(entry["version"]),
			Url:     asString(entry["url"]),
		}
		switch v := entry["size"].(type) {
		case int:
			pkg.Size = int64(v)
		case int64:
			pkg.Size = v
		case float64:
			pkg.Size = int64(v)
		case string:
			if n, err := strconv.ParseInt(v, 10, 64); err == nil {
				pkg.Size = n
			}
		}
		if pkg.Version != "" {
			out[name] = pkg
		}
	}
	if len(out) == 0 {
		for k, v := range defaultFirmwareTargets {
			out[k] = v
		}
	}
	return out
}

func asString(v interface{}) string {
	if v == nil {
		return ""
	}
	s, _ := v.(string)
	return strings.TrimSpace(s)
}

func CompareSemver(a, b string) int {
	a = strings.TrimSpace(strings.TrimPrefix(a, "v"))
	b = strings.TrimSpace(strings.TrimPrefix(b, "v"))
	if a == b {
		return 0
	}
	ap := parseVersionParts(a)
	bp := parseVersionParts(b)
	n := len(ap)
	if len(bp) > n {
		n = len(bp)
	}
	for i := 0; i < n; i++ {
		av, bv := 0, 0
		if i < len(ap) {
			av = ap[i]
		}
		if i < len(bp) {
			bv = bp[i]
		}
		if av < bv {
			return -1
		}
		if av > bv {
			return 1
		}
	}
	return 0
}

func parseVersionParts(v string) []int {
	segments := strings.Split(v, ".")
	parts := make([]int, 0, len(segments))
	for _, seg := range segments {
		seg = strings.TrimSpace(seg)
		if seg == "" {
			parts = append(parts, 0)
			continue
		}
		digits := strings.Builder{}
		for _, ch := range seg {
			if ch >= '0' && ch <= '9' {
				digits.WriteRune(ch)
			} else {
				break
			}
		}
		if digits.Len() == 0 {
			parts = append(parts, 0)
			continue
		}
		n, _ := strconv.Atoi(digits.String())
		parts = append(parts, n)
	}
	return parts
}
