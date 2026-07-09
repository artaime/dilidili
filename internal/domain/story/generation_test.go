package story

import "testing"

func TestIsGenerationComplete(t *testing.T) {
	draft := &StoryRecord{
		FullText: "",
		ParamsSnapshot: map[string]any{"draft": true, "generation_complete": false},
	}
	if IsGenerationComplete(draft) {
		t.Fatal("draft should be incomplete")
	}
	partial := &StoryRecord{
		FullText:           "第一句。",
		GenerationComplete: false,
		ParamsSnapshot:     map[string]any{"generation_complete": false},
	}
	if IsGenerationComplete(partial) {
		t.Fatal("explicit incomplete should be false")
	}
	complete := &StoryRecord{
		FullText:           "完整故事。",
		GenerationComplete: true,
	}
	if !IsGenerationComplete(complete) {
		t.Fatal("expected complete")
	}
}

func TestPlanContinueGenerate(t *testing.T) {
	svc := NewService(nil)
	rec := &StoryRecord{
		StoryID:  "sid-1",
		FullText: "很久很久以前，有个小孩。他出发了。",
		LastPosition: PlayPosition{
			LastSentenceIndex: 0,
			LastSentence:      "很久很久以前，有个小孩。",
		},
		Mode: "classic",
		ParamsSnapshot: map[string]any{
			"theme": "小冒险",
		},
	}
	plan, err := svc.PlanContinueGenerate(nil, rec)
	if err != nil {
		t.Fatal(err)
	}
	if !plan.IsContinuation || plan.StoryID != "sid-1" {
		t.Fatalf("unexpected plan: %+v", plan)
	}
	if plan.ContinueFrom != rec.FullText {
		t.Fatalf("unexpected continue from: %q", plan.ContinueFrom)
	}
	if len(plan.DraftPlaybackSentences) != 1 || plan.DraftPlaybackSentences[0] != "他出发了。" {
		t.Fatalf("unexpected draft: %v", plan.DraftPlaybackSentences)
	}
}

func TestMergeStoryText(t *testing.T) {
	got := mergeStoryText("第一句。", "第二句。")
	if got != "第一句。第二句。" {
		t.Fatalf("unexpected merge: %q", got)
	}
}
