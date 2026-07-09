package chat

import (
	"context"
	"errors"
	"fmt"
)

func storyFailureReason(interrupted bool, ttsErr, genErr error) string {
	if interrupted {
		if ttsErr != nil {
			return "user_tts_interrupt"
		}
		if errors.Is(genErr, context.Canceled) {
			return "generation_canceled"
		}
		return "interrupted"
	}
	if genErr == nil {
		return "unknown"
	}
	if errors.Is(genErr, context.DeadlineExceeded) {
		return "llm_timeout"
	}
	if errors.Is(genErr, context.Canceled) {
		return "generation_context_canceled"
	}
	return fmt.Sprintf("llm_error:%v", genErr)
}
