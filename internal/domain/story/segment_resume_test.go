package story

import "testing"

func TestResumeStartSegment(t *testing.T) {
	rec := &StoryRecord{
		LastPlayStatus: PlayStatusInterrupted,
		LastPosition:   PlayPosition{SegmentIndex: 3},
		Segments:       []string{"a", "b", "c", "d", "e"},
	}
	if got := ResumeStartSegment(rec); got != 2 {
		t.Fatalf("expected segment 2, got %d", got)
	}
	rec.LastPosition.SegmentIndex = 0
	if got := ResumeStartSegment(rec); got != 0 {
		t.Fatalf("expected segment 0 at start, got %d", got)
	}
	rec.LastPlayStatus = PlayStatusCompleted
	rec.LastPosition.SegmentIndex = 4
	if got := ResumeStartSegment(rec); got != 0 {
		t.Fatalf("expected restart after complete, got %d", got)
	}
}

func TestResumeStartSegmentBadEndProgress(t *testing.T) {
	rec := &StoryRecord{
		LastPlayStatus: PlayStatusInterrupted,
		LastPosition:   PlayPosition{SegmentIndex: 4},
		Segments:       []string{"a", "b", "c", "d", "e"},
		CompleteCount:  0,
	}
	if got := ResumeStartSegment(rec); got != 0 {
		t.Fatalf("expected restart from 0 on bogus end progress, got %d", got)
	}
}

func TestBuildResumePrefix(t *testing.T) {
	rec := &StoryRecord{
		LastPlayStatus: PlayStatusInterrupted,
		LastPosition:   PlayPosition{SegmentIndex: 1},
		Segments:       []string{"第一段。", "叶公把龙画得非常像。"},
	}
	prefix := BuildResumePrefix(rec, 0)
	if prefix == "" || !containsAll(prefix, "叶公") {
		t.Fatalf("unexpected prefix: %s", prefix)
	}
}

func containsAll(s string, parts ...string) bool {
	for _, p := range parts {
		if p != "" && !stringsContains(s, p) {
			return false
		}
	}
	return true
}

func stringsContains(s, sub string) bool {
	return len(sub) == 0 || (len(s) >= len(sub) && indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
