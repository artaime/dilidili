package story

import "testing"

func TestDraftPlaybackPlanB(t *testing.T) {
	full := "第一句。第二句。第三句。第四句。第五句。"
	rec := &StoryRecord{
		FullText:       full,
		LastPlayStatus: PlayStatusInterrupted,
		LastPosition: PlayPosition{
			CharOffset:        8, // 听完「第一句。第二句。」
			LastSentenceIndex: 1,
			LastSentence:      "第二句。",
		},
	}
	spoken := SpokenTextPrefix(rec)
	if spoken != "第一句。第二句。" {
		t.Fatalf("spoken prefix: got %q", spoken)
	}
	draft := DraftPlaybackSentences(rec)
	want := []string{"第三句。", "第四句。", "第五句。"}
	if len(draft) != len(want) {
		t.Fatalf("draft sentences: got %v want %v", draft, want)
	}
	for i := range want {
		if draft[i] != want[i] {
			t.Fatalf("draft[%d]: got %q want %q", i, draft[i], want[i])
		}
	}
}

func TestDraftPlaybackAllHeard(t *testing.T) {
	full := "第一句。第二句。"
	rec := &StoryRecord{
		FullText: full,
		LastPosition: PlayPosition{
			LastSentenceIndex: 1,
			LastSentence:      "第二句。",
		},
	}
	if got := DraftPlaybackSentences(rec); got != nil {
		t.Fatalf("expected no draft, got %v", got)
	}
}

func TestMergeHeardStoryText(t *testing.T) {
	got := MergeHeardStoryText("第一句。第二句。", "第三句。")
	if got != "第一句。第二句。第三句。" {
		t.Fatalf("unexpected: %q", got)
	}
}

func TestPlanContinueGenerateDraft(t *testing.T) {
	svc := NewService(nil)
	full := "第一句。第二句。第三句。第四句。"
	rec := &StoryRecord{
		StoryID:  "sid-1",
		FullText: full,
		LastPosition: PlayPosition{
			LastSentenceIndex: 0,
			LastSentence:      "第一句。",
		},
	}
	plan, err := svc.PlanContinueGenerate(nil, rec)
	if err != nil {
		t.Fatal(err)
	}
	if plan.ContinueFrom != full {
		t.Fatalf("continue from full text: %q", plan.ContinueFrom)
	}
	if plan.SpokenBaseline != "第一句。" {
		t.Fatalf("spoken baseline: %q", plan.SpokenBaseline)
	}
	if len(plan.DraftPlaybackSentences) != 3 {
		t.Fatalf("draft sentences: %v", plan.DraftPlaybackSentences)
	}
}
