package device_memory

import (
	"errors"
	"fmt"
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

func TestIsMemobaseUserNotFound(t *testing.T) {
	cases := []struct {
		err  error
		want bool
	}{
		{fmt.Errorf("User 4cf315dc-a5e5-57cd-89e7-256c8ebb1316 not found"), true},
		{fmt.Errorf("404 Not Found"), true},
		{nil, false},
		{fmt.Errorf("connection refused"), false},
	}
	for _, tc := range cases {
		if got := isMemobaseUserNotFound(tc.err); got != tc.want {
			t.Fatalf("isMemobaseUserNotFound(%v) = %v, want %v", tc.err, got, tc.want)
		}
	}
}
