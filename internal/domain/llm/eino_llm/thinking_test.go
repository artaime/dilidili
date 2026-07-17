package eino_llm

import (
	"encoding/json"
	"testing"
)

func TestApplyProviderThinkingDefaults_DeepSeek(t *testing.T) {
	t.Parallel()

	got := applyProviderThinkingDefaults("deepseek", thinkingConfig{})
	if got.Mode != "disabled" {
		t.Fatalf("expected deepseek default mode disabled, got %q", got.Mode)
	}

	got = applyProviderThinkingDefaults("deepseek", thinkingConfig{Mode: "default"})
	if got.Mode != "disabled" {
		t.Fatalf("expected deepseek default mode disabled, got %q", got.Mode)
	}

	got = applyProviderThinkingDefaults("deepseek", thinkingConfig{Mode: "enabled"})
	if got.Mode != "enabled" {
		t.Fatalf("expected explicit enabled to be preserved, got %q", got.Mode)
	}

	got = applyProviderThinkingDefaults("openai", thinkingConfig{})
	if got.enabled() {
		t.Fatalf("expected non-deepseek empty thinking to stay disabled/empty, got %+v", got)
	}
}

func TestInjectThinkingPayload_DeepSeekDefaultDisabled(t *testing.T) {
	t.Parallel()

	thinking := applyProviderThinkingDefaults("deepseek", thinkingConfig{})
	payload := map[string]interface{}{
		"model": "deepseek-v4-flash",
	}
	if !injectThinkingPayload(payload, "deepseek", thinking) {
		t.Fatal("expected thinking payload injection")
	}

	raw, ok := payload["thinking"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected thinking object, got %#v", payload["thinking"])
	}
	if raw["type"] != "disabled" {
		t.Fatalf("expected thinking.type=disabled, got %#v", raw["type"])
	}
}

func TestParseThinkingConfig_DeepSeekUnset(t *testing.T) {
	t.Parallel()

	cfg := map[string]interface{}{
		"provider":   "deepseek",
		"type":       "openai",
		"model_name": "deepseek-chat",
		"api_key":    "test",
	}
	got := parseThinkingConfig(cfg)
	if got.Mode != "disabled" {
		t.Fatalf("expected parseThinkingConfig to default deepseek to disabled, got %q", got.Mode)
	}

	cfg["thinking"] = map[string]interface{}{"mode": "enabled"}
	got = parseThinkingConfig(cfg)
	if got.Mode != "enabled" {
		t.Fatalf("expected explicit enabled, got %q", got.Mode)
	}

	// ensure inject shape matches DeepSeek API
	payload := map[string]interface{}{}
	if !injectThinkingPayload(payload, "deepseek", got) {
		t.Fatal("expected injection for enabled")
	}
	b, _ := json.Marshal(payload["thinking"])
	if string(b) != `{"type":"enabled"}` {
		t.Fatalf("unexpected thinking json: %s", b)
	}
}
