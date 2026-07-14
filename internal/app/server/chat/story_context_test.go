package chat

import (
	"testing"
	"time"
)

func TestTryBeginStoryStreamDedupe(t *testing.T) {
	m := &ChatManager{}
	if !m.tryBeginStoryStream("叶公好龙") {
		t.Fatal("expected first start allowed")
	}
	if m.tryBeginStoryStream("叶公好龙") {
		t.Fatal("expected duplicate blocked while active")
	}
	m.endStoryStreamGuard()
	m.storyStreamGuard.startedAt = time.Now().Add(-storyStreamDedupeWindow - time.Second)
	if !m.tryBeginStoryStream("叶公好龙") {
		t.Fatal("expected start after window and end")
	}
	m.endStoryStreamGuard()
	if !m.tryBeginStoryStream("女娲补天") {
		t.Fatal("expected different theme allowed after end")
	}
	m.endStoryStreamGuard()
	if !m.tryBeginStoryStream("") {
		t.Fatal("expected empty theme allowed")
	}
	if m.tryBeginStoryStream("") {
		t.Fatal("expected empty theme duplicate blocked")
	}
}

func TestStorySpokenText(t *testing.T) {
	got := storySpokenText("好呀。", "从前有个小孩。")
	if got != "好呀。从前有个小孩。" {
		t.Fatalf("unexpected: %q", got)
	}
}
