package aitoy

import (
	"encoding/json"
	"testing"
)

func TestIsAIToyDevice_UserAgent(t *testing.T) {
	if !IsAIToyDevice(DeviceReport{UserAgent: "AIToy/2.31.0"}) {
		t.Fatal("expected AIToy user-agent to match")
	}
	if IsAIToyDevice(DeviceReport{UserAgent: "ESP32/1.0"}) {
		t.Fatal("expected ESP32 user-agent not to match")
	}
}

func TestIsAIToyDevice_BoardType(t *testing.T) {
	if !IsAIToyDevice(DeviceReport{BoardType: "pangdou-toy"}) {
		t.Fatal("expected pangdou board type to match")
	}
}

func TestIsAIToyDevice_MultiSubVersion(t *testing.T) {
	report := DeviceReport{
		ApplicationVer: `{"pd-v1-gx8006":"2.12.0","pd-v1-ln882":"2.31.0","pd-v1-re3220":""}`,
	}
	if !IsAIToyDevice(report) {
		t.Fatal("expected multi-sub application.version to match")
	}
}

func TestIsAIToyDevice_ESP32Version(t *testing.T) {
	report := DeviceReport{
		ApplicationVer: "0.9.9",
		BoardType:      "esp-box-3",
	}
	if IsAIToyDevice(report) {
		t.Fatal("expected plain ESP32 version not to match AIToy")
	}
}

func TestBuildFirmware_UpgradeWhenOlder(t *testing.T) {
	report := DeviceReport{
		ApplicationVer: `{"pd-v1-gx8006":"2.12.0","pd-v1-ln882":"2.31.0","pd-v1-re3220":"2.12.0"}`,
	}
	fw := BuildFirmware(report)
	if len(fw.Subs) == 0 {
		t.Fatal("expected firmware subs for older versions")
	}
	if _, ok := fw.Subs["pd-v1-ln882"]; !ok {
		t.Fatal("expected pd-v1-ln882 upgrade entry")
	}
}

func TestBuildFirmware_NoUpgradeWhenUpToDate(t *testing.T) {
	report := DeviceReport{
		ApplicationVer: `{"pd-v1-gx8006":"9.99.0","pd-v1-ln882":"9.99.0","pd-v1-re3220":"9.99.0"}`,
	}
	fw := BuildFirmware(report)
	if len(fw.Subs) != 0 {
		t.Fatalf("expected no subs when up to date, got %#v", fw.Subs)
	}
}

func TestCompareSemver(t *testing.T) {
	if CompareSemver("2.31.0", "2.34.0") >= 0 {
		t.Fatal("expected 2.31.0 < 2.34.0")
	}
	if CompareSemver("2.34.0", "2.34.0") != 0 {
		t.Fatal("expected equal versions")
	}
}

func TestFirmwareInfo_JSONShape(t *testing.T) {
	fw := FirmwareInfo{
		Subs: map[string]FirmwareSubPackage{
			"pd-v1-ln882": {Version: "2.34.0", Size: 1, Url: "http://example.com/a.bin"},
		},
	}
	raw, err := json.Marshal(fw)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]interface{}
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	if _, ok := m["version"]; ok {
		t.Fatal("AIToy subs firmware should not contain top-level version")
	}
	if _, ok := m["subs"]; !ok {
		t.Fatal("AIToy firmware JSON should contain subs")
	}
}
