package chat

import "dili-esp32-server-golang/internal/domain/chat/debuglog"

// storyDebugLogTTSSynthesized 兼容旧配置；与 chat.debug_log_tts_only 共用 TTS 调试开关。
func storyDebugLogTTSSynthesized() bool {
	return debuglog.TTSOnlyEnabled()
}
