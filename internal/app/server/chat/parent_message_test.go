package chat

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestParseParentMessageIntentJSON(t *testing.T) {
	cases := []struct {
		raw    string
		expect parentMessageIntent
	}{
		{`{"intent":"play"}`, parentMessageIntentAffirmative},
		{`{"intent":"skip"}`, parentMessageIntentNegative},
		{`{"intent":"unknown"}`, parentMessageIntentUnknown},
		{`说明：{"intent":"play"}`, parentMessageIntentAffirmative},
		{``, parentMessageIntentUnknown},
	}
	for _, tc := range cases {
		got := parseParentMessageIntentJSON(tc.raw)
		if got != tc.expect {
			t.Fatalf("parseParentMessageIntentJSON(%q) = %d, want %d", tc.raw, got, tc.expect)
		}
	}
}

func TestMapIntentString(t *testing.T) {
	if mapIntentString("PLAY") != parentMessageIntentAffirmative {
		t.Fatal("expected play")
	}
	if mapIntentString("skip") != parentMessageIntentNegative {
		t.Fatal("expected skip")
	}
	if mapIntentString("other") != parentMessageIntentUnknown {
		t.Fatal("expected unknown")
	}
}

func TestBuildAskPromptFallbackDoesNotGuideCommands(t *testing.T) {
	createdAt := time.Date(2026, 6, 16, 14, 12, 0, 0, time.Local)
	prompt := buildAskPromptFallback("爸爸", createdAt, createdAt)
	if strings.Contains(prompt, "请说") || strings.Contains(prompt, "口令") {
		t.Fatalf("ask prompt should not guide commands: %q", prompt)
	}
	if !strings.Contains(prompt, "要播放吗") {
		t.Fatalf("ask prompt should ask whether to play: %q", prompt)
	}
}

func TestSubmitParentMessageIntentNegativeOverridesAffirmative(t *testing.T) {
	state := &parentMessageFlowState{
		intentCh: make(chan parentMessageIntent, 1),
	}
	submitParentMessageIntent(state, parentMessageIntentAffirmative)
	submitParentMessageIntent(state, parentMessageIntentNegative)

	select {
	case got := <-state.intentCh:
		if got != parentMessageIntentNegative {
			t.Fatalf("expected negative override, got %d", got)
		}
	default:
		t.Fatal("expected intent on channel")
	}
}

func TestSubmitParentMessageIntentNegativeIgnoresLaterAffirmative(t *testing.T) {
	state := &parentMessageFlowState{
		intentCh: make(chan parentMessageIntent, 1),
	}
	submitParentMessageIntent(state, parentMessageIntentNegative)
	submitParentMessageIntent(state, parentMessageIntentAffirmative)

	select {
	case got := <-state.intentCh:
		if got != parentMessageIntentNegative {
			t.Fatalf("expected negative to remain, got %d", got)
		}
	default:
		t.Fatal("expected intent on channel")
	}
}

func TestHandleParentMessageASRUnknownDoesNotResolveIntent(t *testing.T) {
	state := &parentMessageFlowState{
		messages: []parentMessageItem{{ID: 1, TextContent: "hi", FamilyRole: "爸爸", CreatedAt: time.Now()}},
		intentCh: make(chan parentMessageIntent, 1),
	}
	manager := &ChatManager{
		DeviceID: "dev-test",
	}
	manager.parentMessageState = state

	handled, _ := manager.handleParentMessageASR("今天天气怎么样")
	if !handled {
		t.Fatal("expected ASR to be handled")
	}
	if state.intentResolved {
		t.Fatal("unknown ASR should not resolve intent")
	}
	select {
	case <-state.intentCh:
		t.Fatal("unknown ASR should not write intent channel")
	default:
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
