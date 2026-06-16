package websocket

import "testing"

func TestResolveOTADeviceIDPrefersBoardSn(t *testing.T) {
	req := &OtaRequest{
		MacAddress: "00:50:47:ba:b8:e8",
		Board: Board{
			Sn:   "SN-TEST-001",
			Imei: "865229085717303",
			Mac:  "00:50:47:ba:b8:e8",
		},
	}
	got := resolveOTADeviceID("SN-HEADER-001", req)
	if got != "SN-TEST-001" {
		t.Fatalf("got %q want SN-TEST-001", got)
	}
}

func TestResolveOTADeviceIDIgnoresMacAndImei(t *testing.T) {
	req := &OtaRequest{
		MacAddress: "00:50:47:ba:b8:e8",
		Board: Board{
			Imei: "865229085717303",
			Mac:  "00:50:47:ba:b8:e8",
		},
	}
	got := resolveOTADeviceID("SN-HEADER-001", req)
	if got != "SN-HEADER-001" {
		t.Fatalf("got %q want header SN", got)
	}
}

func TestResolveOTADeviceIDFallsBackToHeader(t *testing.T) {
	got := resolveOTADeviceID("SN-HEADER-001", nil)
	if got != "SN-HEADER-001" {
		t.Fatalf("got %q", got)
	}
}

func TestResolveActivateDeviceIDPrefersSerialNumber(t *testing.T) {
	got := resolveActivateDeviceID("SN-HEADER-001", "SN-ACTIVATE-001")
	if got != "SN-ACTIVATE-001" {
		t.Fatalf("got %q", got)
	}
}
