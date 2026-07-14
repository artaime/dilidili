package intent

import (
	"strings"
	"testing"
)

func TestBuildClassifierSystemPromptIncludesDeviceIntent(t *testing.T) {
	got := BuildClassifierSystemPrompt("你是测试助手")
	if !strings.Contains(got, `intent":"msg_inquiry|msg_play|device|general"`) &&
		!strings.Contains(got, "device|general") {
		t.Fatalf("expected device in output format, got %q", got)
	}
	if !strings.Contains(got, "- device：") {
		t.Fatalf("expected device intent definition, got %q", got)
	}
	if !strings.Contains(got, "音量") || !strings.Contains(got, "电量") {
		t.Fatalf("expected volume/battery examples, got %q", got)
	}
	if !strings.Contains(got, "本机音量/电量等请用 intent=device") {
		t.Fatalf("expected general must not steal device ops, got %q", got)
	}
}

func TestParseRouterResponseDevice(t *testing.T) {
	resp, err := ParseRouterResponse(`{"intent":"device","confidence":"0.97","data":{}}`)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if resp.Intent != IntentDevice {
		t.Fatalf("intent=%s want device", resp.Intent)
	}
}
