package story

import "testing"

func TestPlaybackProgressCompleted(t *testing.T) {
	rec := &StoryRecord{
		Segments:       []string{"a", "b", "c"},
		LastPlayStatus: PlayStatusCompleted,
		LastPosition:   PlayPosition{SegmentIndex: 1},
	}
	idx, total, pct := PlaybackProgress(rec)
	if idx != 2 || total != 3 || pct != 100 {
		t.Fatalf("unexpected progress: idx=%d total=%d pct=%d", idx, total, pct)
	}
}

func TestPlaybackProgressInterrupted(t *testing.T) {
	rec := &StoryRecord{
		Segments:       []string{"a", "b", "c", "d", "e"},
		LastPlayStatus: PlayStatusInterrupted,
		LastPosition:   PlayPosition{SegmentIndex: 2},
	}
	_, total, pct := PlaybackProgress(rec)
	if total != 5 || pct != 60 {
		t.Fatalf("unexpected progress: total=%d pct=%d", total, pct)
	}
}
