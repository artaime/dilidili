package chat

import (
	"fmt"
	"testing"
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
	if parseParentMessageIntentJSON(`说明：{"intent":"skip"}`) != parentMessageIntentNegative {
		t.Fatal("expected skip")
	}
}
