package story

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestBuildStoryContextBrief(t *testing.T) {
	rec := &StoryRecord{
		Title:    "叶公好龙的故事",
		FullText: "叶公非常喜欢龙。",
		ParamsSnapshot: map[string]any{
			"theme": "叶公好龙",
		},
	}
	brief := BuildStoryContextBrief(rec)
	if brief == "" {
		t.Fatal("expected brief")
	}
	if !strings.Contains(brief, "叶公好龙") || !strings.Contains(brief, "叶公非常喜欢龙") {
		t.Fatalf("unexpected brief: %s", brief)
	}
}

func TestFindLatestByThemeSkipsEmptyDraft(t *testing.T) {
	cfg := Config{MinRetentionDays: 7, MaxRetentionDays: 90}
	mem := newMemoryBackend()
	store := NewStoreWithBackend(mem, "test", cfg)
	ctx := context.Background()
	now := time.Now()

	_ = store.Save(ctx, &StoryRecord{
		DeviceID: "dev1", Title: "叶公好龙的故事", FullText: "",
		ParamsSnapshot: map[string]any{"theme": "叶公好龙"},
		LastPlayedAt: now,
	})
	full := &StoryRecord{
		DeviceID: "dev1", Title: "叶公好龙的故事",
		FullText: "叶公喜欢龙。", Segments: []string{"叶公喜欢龙。"},
		ParamsSnapshot: map[string]any{"theme": "叶公好龙"},
		LastPlayedAt: now.Add(time.Second),
	}
	if err := store.Save(ctx, full); err != nil {
		t.Fatal(err)
	}
	got, err := store.FindLatestByTheme(ctx, "dev1", "叶公好龙", true)
	if err != nil {
		t.Fatal(err)
	}
	if got.StoryID != full.StoryID {
		t.Fatalf("expected full story, got %s", got.StoryID)
	}
}
