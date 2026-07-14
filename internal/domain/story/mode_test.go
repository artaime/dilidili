package story

import "testing"

func TestShouldTellCanonical(t *testing.T) {
	if !ShouldTellCanonical(StoryParams{RequestType: StoryModeClassic, NarrationMode: NarrationCanonical, Theme: "龟兔赛跑"}) {
		t.Fatal("classic canonical expected")
	}
	if ShouldTellCanonical(StoryParams{RequestType: StoryModeClassic, NarrationMode: NarrationCreative, Theme: "龟兔赛跑"}) {
		t.Fatal("creative should not be canonical")
	}
	if !ShouldTellCanonical(StoryParams{RequestType: StoryModeMyth, Theme: "女娲补天"}) {
		t.Fatal("myth default canonical")
	}
	if ShouldTellCanonical(StoryParams{RequestType: StoryModeOriginal, Theme: "小恐龙"}) {
		t.Fatal("original should be creative")
	}
}

func TestParseStoryIntentJSON(t *testing.T) {
	raw := `{"is_story_request":true,"confidence":0.95,"action":"generate","theme":"后裔射太阳","canonical":"后羿射日","story_type":"myth","narration_mode":"canonical"}`
	got, err := ParseStoryIntentJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	if !got.IsStoryRequest || got.Theme != "后裔射太阳" || got.Canonical != "后羿射日" || got.StoryType != StoryModeMyth {
		t.Fatalf("unexpected: %+v", got)
	}
}

func TestIntentToStoryParamsPrefersCanonical(t *testing.T) {
	p := IntentToStoryParams(IntentResult{
		StoryType:     StoryModeMyth,
		Theme:         "后裔射太阳",
		Canonical:     "后羿射日",
		NarrationMode: NarrationCanonical,
	})
	if p.Theme != "后羿射日" || p.ThemeRaw != "后裔射太阳" {
		t.Fatalf("unexpected: %+v", p)
	}
	if p.RequestType != StoryModeMyth || p.NarrationMode != NarrationCanonical {
		t.Fatalf("unexpected mode: %+v", p)
	}
}

func TestResolveIntentTheme(t *testing.T) {
	theme, raw := ResolveIntentTheme(IntentResult{Theme: "哪吒脑海", Canonical: "哪吒闹海"})
	if theme != "哪吒闹海" || raw != "哪吒脑海" {
		t.Fatalf("got theme=%q raw=%q", theme, raw)
	}
	theme, raw = ResolveIntentTheme(IntentResult{Theme: "小恐龙"})
	if theme != "小恐龙" || raw != "小恐龙" {
		t.Fatalf("fallback got theme=%q raw=%q", theme, raw)
	}
}
