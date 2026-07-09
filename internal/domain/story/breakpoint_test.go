package story

import "testing"

func TestComputePlayPosition(t *testing.T) {
	full := "第一句。第二句。第三句。"
	pos := ComputePlayPosition(full, "第一句。第二句。")
	if pos.CharOffset != 8 {
		t.Fatalf("char offset: got %d want 8", pos.CharOffset)
	}
	if pos.LastSentenceIndex != 1 {
		t.Fatalf("last sentence index: got %d want 1", pos.LastSentenceIndex)
	}
	if pos.LastSentence != "第二句。" {
		t.Fatalf("last sentence: got %q", pos.LastSentence)
	}
}

func TestTextFromResumePositionEarlyProgress(t *testing.T) {
	rec := &StoryRecord{
		FullText:       "第一句。第二句。第三句。",
		LastPlayStatus: PlayStatusInterrupted,
		LastPosition: PlayPosition{
			CharOffset:        4,
			LastSentenceIndex: 0,
			LastSentence:      "第一句。",
		},
	}
	body := TextFromResumePosition(rec)
	if body != "第二句。第三句。" {
		t.Fatalf("expected continue after first sentence, got %q", body)
	}
}

func TestTextFromResumePosition(t *testing.T) {
	rec := &StoryRecord{
		FullText:       "第一句。第二句。第三句。",
		LastPlayStatus: PlayStatusInterrupted,
		LastPosition: PlayPosition{
			CharOffset:        8,
			LastSentenceIndex: 1,
			LastSentence:      "第二句。",
		},
	}
	body := TextFromResumePosition(rec)
	if body != "第二句。第三句。" {
		t.Fatalf("unexpected resume body: %q", body)
	}
}

func TestResumeSpeakPlan(t *testing.T) {
	rec := &StoryRecord{
		FullText:       "第一句。第二句。第三句。",
		Segments:       SegmentText("第一句。第二句。第三句。"),
		LastPlayStatus: PlayStatusInterrupted,
		LastPosition: PlayPosition{
			CharOffset:        8,
			LastSentenceIndex: 1,
			LastSentence:      "第二句。",
		},
	}
	startSeg, prefix, body := ResumeSpeakPlan(rec)
	if body == "" || prefix == "" {
		t.Fatalf("empty plan: prefix=%q body=%q", prefix, body)
	}
	if startSeg < 0 {
		t.Fatalf("invalid start segment: %d", startSeg)
	}
}

func TestPlaybackProgressByCharOffset(t *testing.T) {
	rec := &StoryRecord{
		FullText:       "一二三四五六七八九十",
		Segments:       []string{"一二三四五", "六七八九十"},
		LastPlayStatus: PlayStatusInterrupted,
		LastPosition:   PlayPosition{CharOffset: 5, SegmentIndex: 0},
	}
	_, _, pct := PlaybackProgress(rec)
	if pct != 50 {
		t.Fatalf("expected 50%%, got %d", pct)
	}
}
