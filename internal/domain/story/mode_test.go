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
	raw := `{"is_story_request":true,"confidence":0.95,"action":"generate","theme":"龟兔赛跑","story_type":"classic","narration_mode":"canonical"}`
	got, err := ParseStoryIntentJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	if !got.IsStoryRequest || got.Theme != "龟兔赛跑" || got.StoryType != StoryModeClassic {
		t.Fatalf("unexpected: %+v", got)
	}
}

func TestIntentToStoryParams(t *testing.T) {
	p := IntentToStoryParams(IntentResult{
		StoryType:     StoryModeMyth,
		Theme:         "夸父逐日",
		NarrationMode: NarrationCanonical,
	})
	if p.RequestType != StoryModeMyth || p.NarrationMode != NarrationCanonical {
		t.Fatalf("unexpected: %+v", p)
	}
}
