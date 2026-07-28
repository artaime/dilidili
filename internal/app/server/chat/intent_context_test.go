package chat

import (
	"strings"
	"testing"

	. "dili-esp32-server-golang/internal/data/client"

	"github.com/cloudwego/eino/schema"
)

func TestBuildClassifierUserPromptIncludesRecentDialogue(t *testing.T) {
	state := &ClientState{Dialogue: &Dialogue{}}
	state.AddMessage(schema.UserMessage("说说北京有哪些著名建筑"))
	state.AddMessage(schema.AssistantMessage("有长城、故宫、天坛呀。", nil))

	got := buildClassifierUserPrompt(state, "介绍一下第二个")
	if !strings.Contains(got, "近期对话：") {
		t.Fatalf("expected recent dialogue header: %q", got)
	}
	if !strings.Contains(got, "长城") || !strings.Contains(got, "故宫") {
		t.Fatalf("expected prior assistant content: %q", got)
	}
	if !strings.Contains(got, "当前用户说：介绍一下第二个") {
		t.Fatalf("expected current utterance: %q", got)
	}
}

func TestBuildClassifierUserPromptWithoutHistory(t *testing.T) {
	got := buildClassifierUserPrompt(nil, "有留言吗")
	if got != "当前用户说：有留言吗" {
		t.Fatalf("got %q", got)
	}
}

func TestBuildStoryRoutingPolicyFactVsStory(t *testing.T) {
	got := buildStoryRoutingPolicy()
	if !strings.Contains(got, "事实介绍") {
		t.Fatalf("expected fact vs story rule: %q", got)
	}
	if !strings.Contains(got, "禁止调用本工具") {
		t.Fatalf("expected forbid tool for factual intro: %q", got)
	}
}
