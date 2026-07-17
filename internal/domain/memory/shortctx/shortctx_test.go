package shortctx

import "testing"

func TestDeviceKeyPattern(t *testing.T) {
	got := DeviceKeyPattern("dili", "SN-001")
	want := "dili:shortctx:*:SN-001:*"
	if got != want {
		t.Fatalf("DeviceKeyPattern = %q, want %q", got, want)
	}
}

func TestStoreKey(t *testing.T) {
	s := &Store{keyPrefix: "dili"}
	got := s.key(7, "SN-A", "3")
	want := "dili:shortctx:7:SN-A:3"
	if got != want {
		t.Fatalf("key = %q, want %q", got, want)
	}
}
