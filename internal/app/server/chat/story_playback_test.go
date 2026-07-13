package chat

import (
	"strings"
	"testing"

	data_client "dili-esp32-server-golang/internal/data/client"
	"dili-esp32-server-golang/internal/domain/story"
	"github.com/spf13/viper"
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
	if !session.isStoryPlaybackAudioGateActive() {
		t.Fatal("expected story audio gate active during playback")
	}
	if !session.shouldIgnoreASRDuringStoryPlayback("回声误识别") {
		t.Fatal("expected ASR to be ignored during story playback")
	}
	session.OnStoryTextSent("第一句。")
	session.ClearStoryPlayback()
	if session.IsStoryPlaybackActive() {
		t.Fatal("expected story playback cleared")
	}
	if session.shouldIgnoreASRDuringStoryPlayback("你好") {
		t.Fatal("expected ASR not ignored after story playback cleared")
	}
}

func TestAssistantOutputAudioGateIgnoresEchoInAutoMode(t *testing.T) {
	clientState := &data_client.ClientState{
		DeviceID:   "dev-assistant-gate",
		ListenMode: "auto",
		Status:     data_client.ClientStatusTTSStart,
		Dialogue:   &data_client.Dialogue{},
	}
	clientState.SetTtsStart(true)
	session := NewChatSession(clientState, NewServerTransport(nil, clientState), nil, nil)

	if !session.isAssistantOutputAudioGateActive() {
		t.Fatal("expected assistant output gate active during TTS")
	}
	if !session.shouldIgnoreASRDuringAssistantOutput("助手正在说的回声") {
		t.Fatal("expected auto mode echo ASR to be ignored during assistant output")
	}

	clientState.ListenMode = "realtime"
	if session.shouldIgnoreASRDuringAssistantOutput("用户插话") {
		t.Fatal("expected realtime mode to allow barge-in during assistant output")
	}
}

func TestLogStoryTTSSynthesizedRespectsConfig(t *testing.T) {
	viper.Reset()
	defer viper.Reset()

	clientState := &data_client.ClientState{DeviceID: "dev-story"}
	session := NewChatSession(clientState, NewServerTransport(nil, clientState), nil, nil)
	session.ActivateStoryPlayback(&story.ToolResult{
		Status:  story.StatusReady,
		StoryID: "sid-log",
	})

	viper.Set("story.debug_log_tts_synthesized", false)
	session.LogStoryTTSSynthesized("不应打印")

	viper.Set("story.debug_log_tts_synthesized", true)
	session.LogStoryTTSSynthesized("合成完成的一句。")
}
