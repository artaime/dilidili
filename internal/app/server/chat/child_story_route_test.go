package chat

import (
	"testing"

	"dili-esp32-server-golang/internal/domain/story"
)

func TestIntentResultToCreateParams(t *testing.T) {
	params := intentResultToCreateParams(story.IntentResult{
		IsStoryRequest: true,
		Action:         "generate",
		Theme:          "龟兔赛跑",
		StoryType:      "classic",
		NarrationMode:  "canonical",
	})
	if params.Theme != "龟兔赛跑" || params.RequestType != story.StoryModeClassic {
		t.Fatalf("unexpected: %+v", params)
	}
	if params.NarrationMode != story.NarrationCanonical {
		t.Fatalf("expected canonical, got %s", params.NarrationMode)
	}
}

func TestIntentResultToCreateParamsUsesCanonical(t *testing.T) {
	params := intentResultToCreateParams(story.IntentResult{
		IsStoryRequest: true,
		Action:         "generate",
		Theme:          "后裔射太阳",
		Canonical:      "后羿射日",
		StoryType:      "myth",
		NarrationMode:  "canonical",
	})
	if params.Theme != "后羿射日" || params.ThemeRaw != "后裔射太阳" {
		t.Fatalf("unexpected: %+v", params)
	}
}

func TestStoryToolResultFromMap(t *testing.T) {
	data := map[string]interface{}{
		"status":        "ready",
		"story_id":      "abc",
		"text_to_speak": "从前有个小孩。",
		"start_segment": float64(0),
		"segments":      []interface{}{"从前有个小孩。"},
	}
	result := storyToolResultFromMap(data)
	if result == nil || result.StoryID != "abc" || result.TextToSpeak == "" {
		t.Fatalf("unexpected result: %+v", result)
	}
}
