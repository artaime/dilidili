package intent

import (
	"testing"
	"time"
)

func TestBuildSelectTimeRangeYesterdayMorning(t *testing.T) {
	now := time.Date(2026, 6, 15, 10, 0, 0, 0, time.Local)
	start, end := BuildSelectTimeRange("昨天", "早上", now)
	if start != "2026-06-14T05:00:00" || end != "2026-06-14T11:00:00" {
		t.Fatalf("unexpected range: %s - %s", start, end)
	}
}

func TestBuildSelectTimeRangeTodayAfternoon(t *testing.T) {
	now := time.Date(2026, 6, 15, 10, 0, 0, 0, time.Local)
	start, end := BuildSelectTimeRange("今天", "下午", now)
	if start != "2026-06-15T12:00:00" || end != "2026-06-15T19:00:00" {
		t.Fatalf("unexpected range: %s - %s", start, end)
	}
}
