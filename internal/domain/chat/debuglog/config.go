package debuglog

import "github.com/spf13/viper"

// TTSOnlyEnabled 为 true 时进入「仅 TTS 调试」模式：打印 TTS 合成正文，抑制 ASR/LLM 流式调试日志。
// 优先读取 chat.debug_log_tts_only；未配置时兼容 story.debug_log_tts_synthesized。
func TTSOnlyEnabled() bool {
	if viper.IsSet("chat.debug_log_tts_only") {
		return viper.GetBool("chat.debug_log_tts_only")
	}
	if viper.IsSet("story.debug_log_tts_synthesized") {
		return viper.GetBool("story.debug_log_tts_synthesized")
	}
	return false
}
