package story

import (
	"math/rand"
	"strings"
	"testing"
)

func TestNeedsGenreDiversity(t *testing.T) {
	if !NeedsGenreDiversity(StoryParams{UserSaidCasual: true}) {
		t.Fatal("casual should diversify genre")
	}
	if !NeedsGenreDiversity(StoryParams{RequestType: StoryModeOriginal}) {
		t.Fatal("empty original should diversify genre")
	}
	if NeedsGenreDiversity(StoryParams{RequestType: StoryModeMyth, NarrationMode: NarrationCanonical, Theme: "哪吒闹海"}) {
		t.Fatal("canonical named should not diversify genre")
	}
	if NeedsGenreDiversity(StoryParams{RequestType: StoryModeMyth, Theme: ""}) {
		t.Fatal("myth type without theme should keep myth, not random genre")
	}
	if NeedsGenreDiversity(StoryParams{Theme: "恐龙探险"}) {
		t.Fatal("explicit theme should not force genre rotation")
	}
}

func TestPickDiversitySeedAvoidsRecent(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	recent := []StoryRecord{
		{
			Title:    "森林朋友",
			FullText: "从前有个叫小明的孩子，和小兔子去森林玩。",
			ParamsSnapshot: map[string]any{
				"genre":      "童话",
				"theme":      "森林朋友",
				"characters": []string{"小明", "小兔子"},
			},
		},
		{
			Title: "冒险日记",
			ParamsSnapshot: map[string]any{
				"genre": "冒险",
				"theme": "地下图书馆",
			},
		},
	}
	seed := PickDiversitySeed(StoryParams{UserSaidCasual: true, RequestType: StoryModeOriginal}, recent, rng)
	if seed == nil {
		t.Fatal("expected seed")
	}
	if seed.Genre == "" || seed.ProtagonistHint == "" {
		t.Fatalf("expected genre and name, got %+v", seed)
	}
	if !containsFold(seed.AvoidNames, "小明") {
		t.Fatalf("expected 小明 in avoid list: %v", seed.AvoidNames)
	}
	if seed.ProtagonistHint == "小明" || seed.ProtagonistHint == "小兔子" {
		t.Fatalf("protagonist should avoid recent names, got %q", seed.ProtagonistHint)
	}
	lines := FormatDiversityPromptLines(seed)
	if len(lines) < 2 {
		t.Fatalf("expected prompt lines, got %v", lines)
	}
}

func TestExtractCharacterNames(t *testing.T) {
	names := ExtractCharacterNames("从前有个叫阿澄的孩子，小橙也来了。", 8)
	if !containsFold(names, "阿澄") {
		t.Fatalf("expected 阿澄 in %v", names)
	}
	if !containsFold(names, "小橙") {
		t.Fatalf("expected 小橙 in %v", names)
	}
}

func TestBuildUserPromptWithSeed(t *testing.T) {
	seed := &DiversitySeed{
		Genre:           "侦探",
		SubjectHint:     "教室里失踪的粉笔",
		ProtagonistHint: "阿澄",
		AvoidNames:      []string{"小明"},
	}
	got := BuildUserPrompt(StoryParams{UserSaidCasual: true}, nil, seed)
	for _, want := range []string{"侦探", "阿澄", "小明", "粉笔"} {
		if !strings.Contains(got, want) {
			t.Fatalf("prompt missing %q: %s", want, got)
		}
	}
}

func TestParseMetaCharacters(t *testing.T) {
	meta, ok := ParseMetaLine("[[meta:title=粉笔谜案|genre=侦探|theme=粉笔|characters=阿澄,同桌]]")
	if !ok || len(meta.Characters) != 2 || meta.Characters[0] != "阿澄" {
		t.Fatalf("unexpected meta: %+v ok=%v", meta, ok)
	}
	rec := &StoryRecord{ParamsSnapshot: map[string]any{}}
	applyStoryMeta(rec, meta, StoryParams{})
	chars, ok := rec.ParamsSnapshot["characters"].([]string)
	if !ok || len(chars) != 2 {
		t.Fatalf("characters not saved: %v", rec.ParamsSnapshot["characters"])
	}
}
