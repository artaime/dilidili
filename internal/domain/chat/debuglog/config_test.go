package debuglog

import (
	"testing"

	"github.com/spf13/viper"
)

func TestTTSOnlyEnabled(t *testing.T) {
	viper.Reset()
	defer viper.Reset()

	if TTSOnlyEnabled() {
		t.Fatal("expected default false")
	}

	viper.Set("story.debug_log_tts_synthesized", true)
	if !TTSOnlyEnabled() {
		t.Fatal("expected story legacy config to enable tts-only debug")
	}

	viper.Reset()
	viper.Set("chat.debug_log_tts_only", false)
	viper.Set("story.debug_log_tts_synthesized", true)
	if TTSOnlyEnabled() {
		t.Fatal("expected chat.debug_log_tts_only=false to override story legacy config")
	}

	viper.Reset()
	viper.Set("chat.debug_log_tts_only", true)
	if !TTSOnlyEnabled() {
		t.Fatal("expected chat.debug_log_tts_only=true")
	}
}
