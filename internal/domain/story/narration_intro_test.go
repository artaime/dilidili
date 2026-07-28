package story

import "testing"

func TestSpeakableStoryTitle(t *testing.T) {
	if got := SpeakableStoryTitle("龟兔赛跑"); got != "龟兔赛跑的故事" {
		t.Fatalf("got %q", got)
	}
	if got := SpeakableStoryTitle("哪吒闹海的故事"); got != "哪吒闹海的故事" {
		t.Fatalf("got %q", got)
	}
	if got := SpeakableStoryTitle("儿童故事"); got != "" {
		t.Fatalf("generic title should be empty, got %q", got)
	}
	if got := SpeakableStoryTitle("很久很久以前有一座山"); got != "" {
		t.Fatalf("opening-like title should be empty, got %q", got)
	}
}

func TestBuildNarrationIntro(t *testing.T) {
	cfg := Config{FillerEnabled: true, FillerDefault: "好呀，我给你讲一个故事。"}
	got := BuildNarrationIntro("龟兔赛跑", cfg)
	want := "好呀，接下来给你讲龟兔赛跑的故事。"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
	if got := BuildNarrationIntro("", cfg); got != cfg.FillerDefault {
		t.Fatalf("empty title fallback got %q", got)
	}
	cfg.FillerEnabled = false
	if got := BuildNarrationIntro("龟兔赛跑", cfg); got != "" {
		t.Fatalf("disabled filler should be empty, got %q", got)
	}
}

func TestBuildResumeTitleLead(t *testing.T) {
	cfg := Config{FillerEnabled: true}
	got := BuildResumeTitleLead("普罗米修斯", cfg)
	want := "好的，我们继续讲普罗米修斯的故事。"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
	if got := BuildResumeTitleLead("", cfg); got != "" {
		t.Fatalf("empty title lead should be empty, got %q", got)
	}
}

func TestBuildMetaTitleAnnounce(t *testing.T) {
	cfg := Config{FillerEnabled: true}
	got := BuildMetaTitleAnnounce("森林里的新朋友", cfg)
	want := "这篇故事叫森林里的新朋友。"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
	got = BuildMetaTitleAnnounce("小熊找星星的故事", cfg)
	want = "这篇故事叫小熊找星星的故事。"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
	if got := BuildMetaTitleAnnounce("儿童故事", cfg); got != "" {
		t.Fatalf("generic title should skip announce, got %q", got)
	}
}

func TestHasNarrationIntroPrefix(t *testing.T) {
	if !HasNarrationIntroPrefix("好呀，接下来给你讲龟兔赛跑的故事。很久以前…") {
		t.Fatal("expected intro prefix")
	}
	if HasNarrationIntroPrefix("很久很久以前，有一只兔子") {
		t.Fatal("story body should not match intro prefix")
	}
}

func TestHasResumeTransitionPrefix(t *testing.T) {
	if !HasResumeTransitionPrefix("上次讲到森林里，我们接着往下讲——") {
		t.Fatal("expected resume prefix")
	}
	if HasResumeTransitionPrefix("小熊继续走。") {
		t.Fatal("bare body should not match")
	}
}

func TestStripRedundantStoryOpening(t *testing.T) {
	if _, skip := StripRedundantStoryOpening("好呀，我给你讲一个故事。"); !skip {
		t.Fatal("expected skip system-like opening")
	}
	if _, skip := StripRedundantStoryOpening("让我来给你讲龟兔赛跑"); !skip {
		t.Fatal("expected skip LLM opening")
	}
	got, skip := StripRedundantStoryOpening("很久很久以前，有一只兔子。")
	if skip || got == "" {
		t.Fatalf("body should keep, got %q skip=%v", got, skip)
	}
}
