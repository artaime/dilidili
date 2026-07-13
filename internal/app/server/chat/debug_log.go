package chat

import (
	"dili-esp32-server-golang/internal/domain/chat/debuglog"
	log "dili-esp32-server-golang/logger"
)

func chatDebugLogTTSOnly() bool {
	return debuglog.TTSOnlyEnabled()
}

// chatDebugLogf 在 debug_log_tts_only 模式下抑制非 TTS 调试日志。
func chatDebugLogf(format string, args ...any) {
	if chatDebugLogTTSOnly() {
		return
	}
	log.Debugf(format, args...)
}

// chatWarnLogf 在 debug_log_tts_only 模式下抑制非 TTS 警告日志。
func chatWarnLogf(format string, args ...any) {
	if chatDebugLogTTSOnly() {
		return
	}
	log.Warnf(format, args...)
}

// chatInfoLogf 在 debug_log_tts_only 模式下抑制非 TTS 信息日志。
func chatInfoLogf(format string, args ...any) {
	if chatDebugLogTTSOnly() {
		return
	}
	log.Infof(format, args...)
}
