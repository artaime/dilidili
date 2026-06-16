package chat

import (
	"testing"
	"time"
)

func TestFormatChildFriendlyTimeToday(t *testing.T) {
	now := time.Date(2026, 6, 12, 15, 0, 0, 0, time.Local)
	created := time.Date(2026, 6, 12, 12, 5, 0, 0, time.Local)
	got := formatChildFriendlyTime(created, now)
	want := "今天中午12点5分"
	if got != want {
		t.Fatalf("formatChildFriendlyTime() = %q, want %q", got, want)
	}
}

func TestFormatChildFriendlyTimeYesterdayEvening(t *testing.T) {
	now := time.Date(2026, 6, 12, 10, 0, 0, 0, time.Local)
	created := time.Date(2026, 6, 11, 21, 18, 0, 0, time.Local)
	got := formatChildFriendlyTime(created, now)
	want := "昨天晚上21点18分"
	if got != want {
		t.Fatalf("formatChildFriendlyTime() = %q, want %q", got, want)
	}
}

func TestBuildConfirmTransitionPrompt(t *testing.T) {
	now := time.Date(2026, 6, 16, 15, 0, 0, 0, time.Local)
	created := time.Date(2026, 6, 16, 14, 12, 0, 0, time.Local)
	got := buildConfirmTransitionPrompt("爸爸", created, now)
	want := "好的，接下来将播放爸爸今天傍晚14点12分的留言。"
	if got != want {
		t.Fatalf("buildConfirmTransitionPrompt() = %q, want %q", got, want)
	}
}

func TestParentMessageNeedsAsk(t *testing.T) {
	base := time.Date(2026, 6, 12, 10, 0, 0, 0, time.Local)
	messages := []parentMessageItem{
		{CreatedAt: base},
		{CreatedAt: base.Add(2 * time.Hour)},
		{CreatedAt: base.Add(5*time.Hour + time.Minute)},
	}
	if !parentMessageNeedsAsk(0, messages) {
		t.Fatal("first message should ask")
	}
	if parentMessageNeedsAsk(1, messages) {
		t.Fatal("second message within 3h should not ask")
	}
	if !parentMessageNeedsAsk(2, messages) {
		t.Fatal("third message after 3h gap should ask")
	}
}
