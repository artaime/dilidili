package device_memory

import (
	"errors"
	"testing"

	"dili-esp32-server-golang/pkg/memobaseuserid"
)

func TestNormalizeMemoryMode(t *testing.T) {
	if normalizeMemoryMode("LONG") != "long" {
		t.Fatal("expected long")
	}
	if normalizeMemoryMode("unknown") != "short" {
		t.Fatal("expected short default")
	}
}

func TestParseMemobaseConfigMissing(t *testing.T) {
	_, err := parseMemobaseConfig(`{"base_url":"http://localhost:6019"}`)
	if !errors.Is(err, ErrMemobaseNotConfigured) {
		t.Fatalf("expected ErrMemobaseNotConfigured, got %v", err)
	}
}

func TestLegacyFallbackIDs(t *testing.T) {
	sn := "3Z73XX06PEV8FXV4G0NQD5R0FZ"
	if memobaseuserid.MemobaseUserID(sn) == memobaseuserid.LegacyMemobaseUserID(sn) {
		t.Fatal("primary and legacy ids must differ")
	}
}

func TestHasMemoryData(t *testing.T) {
	if hasMemoryData(nil, nil, "  ") {
		t.Fatal("empty should be false")
	}
	if !hasMemoryData([]ProfileItem{{Content: "x"}}, nil, "") {
		t.Fatal("profile should count")
	}
}
