package common

import (
	"strings"
	"testing"
)

func TestAppendVoiceReplyStylePrompt(t *testing.T) {
	got := AppendVoiceReplyStylePrompt("你是一个测试助手")

	if !strings.Contains(got, "你是一个测试助手") {
		t.Fatalf("expected original prompt to be preserved, got %q", got)
	}
	if !strings.Contains(got, VoiceReplyStylePrompt) {
		t.Fatalf("expected voice reply style prompt to be appended, got %q", got)
	}

	got = AppendVoiceReplyStylePrompt(got)
	if strings.Count(got, VoiceReplyStylePrompt) != 1 {
		t.Fatalf("expected voice reply style prompt not to be duplicated, got %q", got)
	}
}

func TestBuildVoiceReplyQuery(t *testing.T) {
	got := BuildVoiceReplyQuery("今天北京天气怎么样？")

	if !strings.Contains(got, VoiceReplyStylePrompt) {
		t.Fatalf("expected voice reply style prompt in query, got %q", got)
	}
	if !strings.Contains(got, "用户问题：\n今天北京天气怎么样？") {
		t.Fatalf("expected original query to be preserved, got %q", got)
	}
}
