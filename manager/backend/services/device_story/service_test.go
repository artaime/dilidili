package device_story

import (
	"testing"
	"time"
)

func TestPlaybackProgressIncompleteGeneration(t *testing.T) {
	rec := &storyRecord{
		Segments:           []string{"a", "b"},
		FullText:           "ab",
		LastPlayStatus:     playStatusInterrupted,
		LastPosition:       playPosition{SegmentIndex: 1, CharOffset: 1},
		GenerationComplete: false,
		ParamsSnapshot:     map[string]any{"generation_complete": false},
	}
	_, _, pct, show := playbackProgress(rec)
	if show || pct != 0 {
		t.Fatalf("expected hidden progress, show=%v pct=%d", show, pct)
	}
}

func TestPlaybackProgressInterrupted(t *testing.T) {
	rec := &storyRecord{
		Segments:           []string{"a", "b", "c", "d", "e"},
		FullText:           "abcde",
		LastPlayStatus:     playStatusInterrupted,
		LastPosition:       playPosition{SegmentIndex: 2},
		GenerationComplete: true,
	}
	_, total, pct, show := playbackProgress(rec)
	if !show || total != 5 || pct != 60 {
		t.Fatalf("unexpected progress: show=%v total=%d pct=%d", show, total, pct)
	}
}

func TestResolveDisplayTitle(t *testing.T) {
	rec := &storyRecord{
		Title: "很久很久以前",
		ParamsSnapshot: map[string]any{
			"story_title": "森林奇遇",
			"genre":       "冒险",
		},
	}
	if got := resolveDisplayTitle("", "冒险", rec); got != "森林奇遇" {
		t.Fatalf("got %q", got)
	}
	if got := resolveDisplayTitle("普罗米修斯", "", nil); got != "普罗米修斯的故事" {
		t.Fatalf("got %q", got)
	}
	if got := resolveDisplayTitle("", "神话", &storyRecord{}); got != "神话故事" {
		t.Fatalf("got %q", got)
	}
}

func TestMapStoryListItemPreview(t *testing.T) {
	longText := stringsRepeat("故事", 50)
	rec := &storyRecord{
		StoryID:  "id1",
		Title:    "测试",
		FullText: longText,
		CreatedAt: time.Now(),
		LastPlayedAt: time.Now(),
		ParamsSnapshot: map[string]any{
			"theme": "冒险",
			"age_band": "primary_low",
		},
	}
	item := mapStoryListItem(rec)
	if item.Title != "冒险的故事" {
		t.Fatalf("unexpected title: %q", item.Title)
	}
	if item.Theme != "冒险" || item.AgeBand != "primary_low" {
		t.Fatalf("unexpected item: %+v", item)
	}
	if item.TextLength != 100 {
		t.Fatalf("unexpected length: %d", item.TextLength)
	}
}

func stringsRepeat(s string, n int) string {
	out := make([]rune, 0, n*len([]rune(s)))
	for i := 0; i < n; i++ {
		out = append(out, []rune(s)...)
	}
	return string(out)
}
