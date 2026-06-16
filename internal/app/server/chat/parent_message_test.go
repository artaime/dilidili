package chat

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestClassifyParentMessageIntent(t *testing.T) {
	cases := []struct {
		text   string
		expect parentMessageIntent
	}{
		{"要", parentMessageIntentAffirmative},
		{"好的听一下", parentMessageIntentAffirmative},
		{"想听", parentMessageIntentAffirmative},
		{"不要", parentMessageIntentNegative},
		{"不用了", parentMessageIntentNegative},
		{"算了", parentMessageIntentNegative},
		{"今天天气怎么样", parentMessageIntentUnknown},
		{"", parentMessageIntentUnknown},
	}

	for _, tc := range cases {
		got := classifyParentMessageIntent(tc.text)
		if got != tc.expect {
			t.Fatalf("classifyParentMessageIntent(%q) = %d, want %d", tc.text, got, tc.expect)
		}
	}
}

func TestHasPlayableParentMessage(t *testing.T) {
	manager := &ChatManager{}

	if manager.hasPlayableParentMessage(parentMessageItem{SourceType: "voice", AudioURL: "/audio"}) != true {
		t.Fatal("expected voice message with audio to be playable")
	}
	if manager.hasPlayableParentMessage(parentMessageItem{SourceType: "voice"}) {
		t.Fatal("expected voice message without audio to be unplayable")
	}
	if manager.hasPlayableParentMessage(parentMessageItem{SourceType: "text", TextContent: "你好"}) != true {
		t.Fatal("expected text message with content to be playable")
	}
	if manager.hasPlayableParentMessage(parentMessageItem{SourceType: "text"}) {
		t.Fatal("expected empty text message to be unplayable")
	}
}

func TestWaitActiveTTSDrainReturnsWhenSessionMissing(t *testing.T) {
	manager := &ChatManager{}
	manager.waitActiveTTSDrain(context.Background())
}

func TestIsParentMessageRetryableError(t *testing.T) {
	if !isParentMessageRetryableError(fmt.Errorf("等待 speak_ready 超时")) {
		t.Fatal("expected speak_ready timeout to be retryable")
	}
	if isParentMessageRetryableError(fmt.Errorf("文字留言内容为空")) {
		t.Fatal("expected content error to be non-retryable")
	}
}

func TestParseParentMessageIntentJSON(t *testing.T) {
	if parseParentMessageIntentJSON(`{"intent":"play"}`) != parentMessageIntentAffirmative {
		t.Fatal("expected play")
	}
	if parseParentMessageIntentJSON(`说明：{"intent":"unknown"}`) != parentMessageIntentUnknown {
		t.Fatal("expected unknown")
	}
}

func TestClassifyParentMessageIntentIgnoresAskPromptEcho(t *testing.T) {
	createdAt := time.Date(2026, 6, 16, 14, 12, 0, 0, time.Local)
	prompt := buildAskPromptFallback("爸爸", createdAt, createdAt)
	if classifyParentMessageIntent(prompt) != parentMessageIntentAffirmative {
		t.Fatalf("ask prompt should classify as affirmative (echo risk): %q", prompt)
	}
}
