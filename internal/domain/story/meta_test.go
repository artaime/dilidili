package story

import "testing"

func TestParseMetaLine(t *testing.T) {
	meta, ok := ParseMetaLine("[[meta:title=普罗米修斯盗火|genre=神话|theme=普罗米修斯]]")
	if !ok || meta.Title != "普罗米修斯盗火" || meta.Genre != "神话" {
		t.Fatalf("unexpected meta: %+v ok=%v", meta, ok)
	}
}

func TestMetaStreamFilter(t *testing.T) {
	var f MetaStreamFilter
	out := f.Feed("[[meta:title=小星星|genre=童话|theme=星星]]\n")
	if out != "" || f.Meta == nil {
		t.Fatalf("expected meta only, out=%q meta=%v", out, f.Meta)
	}
	out = f.Feed("很久很久以前，")
	if out != "很久很久以前，" {
		t.Fatalf("unexpected out: %q", out)
	}
}

func TestStripLeadingMeta(t *testing.T) {
	meta, body := StripLeadingMeta("[[meta:title=测试|genre=冒险]]\n正文开始。")
	if meta.Title != "测试" || meta.Genre != "冒险" || body != "正文开始。" {
		t.Fatalf("meta=%+v body=%q", meta, body)
	}
}

func TestApplyStoryMeta(t *testing.T) {
	rec := &StoryRecord{ParamsSnapshot: map[string]any{}}
	applyStoryMeta(rec, StoryMeta{Title: "月球之旅", Genre: "科幻"}, StoryParams{})
	if rec.Title != "月球之旅" {
		t.Fatalf("title=%q", rec.Title)
	}
	if rec.ParamsSnapshot["genre"] != "科幻" {
		t.Fatalf("genre=%v", rec.ParamsSnapshot["genre"])
	}
}

func TestInferGenreFromParams(t *testing.T) {
	if g := InferGenreFromParams(StoryParams{RequestType: StoryModeMyth}); g != "神话" {
		t.Fatalf("got %q", g)
	}
}
