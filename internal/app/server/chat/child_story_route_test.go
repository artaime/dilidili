package chat

import (
	"strings"
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

func TestPrependStoryNarrationIntro(t *testing.T) {
	cfg := story.Config{FillerEnabled: true, FillerDefault: "好呀，我给你讲一个故事。"}
	body := "很久很久以前，有一只兔子。"
	got := prependStoryNarrationIntro(cfg, &story.ToolResult{
		Status: story.StatusReady,
		Title:  "龟兔赛跑",
	}, body)
	wantPrefix := "好呀，接下来给你讲龟兔赛跑的故事。"
	if !strings.HasPrefix(got, wantPrefix) || !strings.HasSuffix(got, body) {
		t.Fatalf("got %q", got)
	}

	resumeBody := "上次讲到森林里，我们接着往下讲——小熊继续走。"
	got = prependStoryNarrationIntro(cfg, &story.ToolResult{
		Status: story.StatusResume,
		Title:  "小熊找星星",
	}, resumeBody)
	// 正文已有续讲衔接语时不再叠「继续讲标题」
	if got != resumeBody {
		t.Fatalf("resume with existing transition should not prepend, got %q", got)
	}

	resumeBare := "小熊继续走。"
	got = prependStoryNarrationIntro(cfg, &story.ToolResult{
		Status: story.StatusResume,
		Title:  "小熊找星星",
	}, resumeBare)
	wantLead := "好的，我们继续讲小熊找星星的故事。"
	if !strings.HasPrefix(got, wantLead) || !strings.HasSuffix(got, resumeBare) {
		t.Fatalf("resume bare got %q", got)
	}

	// 已有过渡语时不重复
	already := wantPrefix + body
	if got := prependStoryNarrationIntro(cfg, &story.ToolResult{Status: story.StatusReplay, Title: "龟兔赛跑"}, already); got != already {
		t.Fatalf("should skip duplicate intro, got %q", got)
	}
}
