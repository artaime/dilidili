package chat

import (
	"strings"
	"testing"

	data_client "dili-esp32-server-golang/internal/data/client"
	"dili-esp32-server-golang/internal/domain/story"
)

func TestBuildStoryRoutingPolicy(t *testing.T) {
	policy := buildStoryRoutingPolicy()
	if policy == "" {
		t.Fatal("expected non-empty story routing policy")
	}
	for _, kw := range []string{"create_child_story", "need_params", "replay"} {
		if !strings.Contains(policy, kw) {
			t.Fatalf("policy missing %q: %s", kw, policy)
		}
	}
}

func TestStoryPlaybackActivateAndProgress(t *testing.T) {
	clientState := &data_client.ClientState{
		DeviceID:  "dev-story",
		AgentID:   "agent1",
		SessionID: "sess1",
		Dialogue:  &data_client.Dialogue{},
	}
	session := NewChatSession(clientState, NewServerTransport(nil, clientState), nil, nil)

	result := &story.ToolResult{
		Status:       story.StatusReady,
		StoryID:      "sid-1",
		TextToSpeak:  "第一句。第二句。",
		Segments:     []string{"第一句。", "第二句。"},
		StartSegment: 0,
	}
	session.ActivateStoryPlayback(result)
	if !session.IsStoryPlaybackActive() {
		t.Fatal("expected story playback active")
	}
	session.OnStoryTextSent("第一句。")
	session.ClearStoryPlayback()
	if session.IsStoryPlaybackActive() {
		t.Fatal("expected story playback cleared")
	}
}
