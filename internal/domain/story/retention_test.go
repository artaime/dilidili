package story

import (
	"testing"
	"time"
)

func TestLastNightWindow(t *testing.T) {
	loc := time.Local
	// 2026-06-30 10:00 — 白天，昨晚窗口应为 6/29 18:00 ~ 6/30 07:00
	now := time.Date(2026, 6, 30, 10, 0, 0, 0, loc)
	start, end := LastNightWindow(now, 18, 7)
	wantStart := time.Date(2026, 6, 29, 18, 0, 0, 0, loc)
	wantEnd := time.Date(2026, 6, 30, 7, 0, 0, 0, loc)
	if !start.Equal(wantStart) || !end.Equal(wantEnd) {
		t.Fatalf("unexpected window: %v ~ %v", start, end)
	}

	// 2026-06-30 06:00 — 凌晨，仍在今日 7 点前
	now2 := time.Date(2026, 6, 30, 6, 0, 0, 0, loc)
	start2, end2 := LastNightWindow(now2, 18, 7)
	if !InTimeWindow(now2.Add(-2*time.Hour), start2, end2) {
		t.Fatalf("expected recent play in last night window")
	}
	_ = end2
}

func TestRetentionDays(t *testing.T) {
	cfg := Config{MinRetentionDays: 7, MaxRetentionDays: 90}
	rec := StoryRecord{PlayCount: 3, CompleteCount: 1}
	days := RetentionDays(rec, cfg)
	if days != 7+3*2+1*3 {
		t.Fatalf("unexpected retention days: %d", days)
	}
}

func TestShouldEvict(t *testing.T) {
	cfg := Config{MinRetentionDays: 7, MaxRetentionDays: 90}
	old := StoryRecord{
		CreatedAt: time.Now().Add(-10 * 24 * time.Hour),
		PlayCount: 0,
	}
	if !ShouldEvict(old, time.Now(), cfg) {
		t.Fatal("expected eviction for 10-day-old story with no replays")
	}
	series := StoryRecord{
		CreatedAt:      time.Now().Add(-30 * 24 * time.Hour),
		SeriesID:       "s1",
		SeriesComplete: false,
	}
	if ShouldEvict(series, time.Now(), cfg) {
		t.Fatal("ongoing series should not evict")
	}
}
