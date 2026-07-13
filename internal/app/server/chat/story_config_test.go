package chat

import (
	"testing"

	"github.com/spf13/viper"
)

func TestStoryDebugLogTTSSynthesized(t *testing.T) {
	viper.Reset()
	defer viper.Reset()

	if storyDebugLogTTSSynthesized() {
		t.Fatal("expected default false")
	}

	viper.Set("chat.debug_log_tts_only", true)
	if !storyDebugLogTTSSynthesized() {
		t.Fatal("expected true when chat.debug_log_tts_only configured")
	}
}
