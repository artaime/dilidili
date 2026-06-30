package memobaseuserid

import "testing"

func TestMemobaseUserIDStable(t *testing.T) {
	sn := "3Z73XX06PEV8FXV4G0NQD5R0FZ"
	got := MemobaseUserID(sn)
	want := "4cf315dc-a5e5-57cd-89e7-256c8ebb1316"
	if got != want {
		t.Fatalf("MemobaseUserID() = %q, want %q", got, want)
	}
	if got != MemobaseUserID(sn) {
		t.Fatal("MemobaseUserID should be stable")
	}
}

func TestLegacyMemobaseUserIDDoubleHash(t *testing.T) {
	sn := "3Z73XX06PEV8FXV4G0NQD5R0FZ"
	got := LegacyMemobaseUserID(sn)
	want := "bf0c5a5b-dd45-5839-90bf-8e8c92369f49"
	if got != want {
		t.Fatalf("LegacyMemobaseUserID() = %q, want %q", got, want)
	}
}

func TestPrimaryAndLegacyDiffer(t *testing.T) {
	sn := "test-device-sn"
	if MemobaseUserID(sn) == LegacyMemobaseUserID(sn) {
		t.Fatal("single and legacy hash should differ")
	}
}
