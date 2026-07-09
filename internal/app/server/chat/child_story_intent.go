package chat

import (
	"context"
	"strings"

	"dili-esp32-server-golang/internal/domain/story"
	log "dili-esp32-server-golang/logger"
)

func (c *ChatManager) classifyChildStoryIntent(ctx context.Context, text string) (*CreateChildStoryParams, bool) {
	if c == nil {
		return nil, false
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, false
	}

	raw, err := c.callLLMSyncText(ctx, story.BuildStoryIntentSystemPrompt(), "用户说："+text)
	if err != nil {
		log.Warnf("设备 %s 故事 LLM 意图识别失败: %v", c.DeviceID, err)
		return nil, false
	}

	intent, err := story.ParseStoryIntentJSON(raw)
	if err != nil {
		log.Warnf("设备 %s 故事意图 JSON 解析失败 raw=%q err=%v", c.DeviceID, raw, err)
		return nil, false
	}
	if !intent.IsStoryRequest || intent.Confidence < story.StoryIntentMinConfidence {
		return nil, false
	}
	if intent.Action == "" || intent.Action == "none" {
		return nil, false
	}

	params := intentResultToCreateParams(intent)
	log.Infof("设备 %s 故事 LLM 意图 action=%s theme=%q type=%s narration=%s confidence=%.2f",
		c.DeviceID, params.Action, params.Theme, params.RequestType, params.NarrationMode, intent.Confidence)
	return params, true
}

func intentResultToCreateParams(intent story.IntentResult) *CreateChildStoryParams {
	sp := story.IntentToStoryParams(intent)
	params := &CreateChildStoryParams{
		Action:         strings.TrimSpace(intent.Action),
		StoryRef:       strings.TrimSpace(intent.StoryRef),
		RequestType:    sp.RequestType,
		NarrationMode:  sp.NarrationMode,
		Theme:          sp.Theme,
		IsBedtime:      sp.IsBedtime,
		UserSaidCasual: intent.UserSaidCasual,
	}
	if params.Action == "" {
		params.Action = story.ActionGenerate
	}
	normalizeCreateChildStoryParams(params)
	return params
}

func normalizeCreateChildStoryParams(params *CreateChildStoryParams) {
	if params == nil {
		return
	}
	sp := story.StoryParams{
		RequestType:    params.RequestType,
		NarrationMode:  params.NarrationMode,
		Theme:          params.Theme,
		IsBedtime:      params.IsBedtime,
		UserSaidCasual: params.UserSaidCasual,
	}
	story.NormalizeStoryParams(&sp)
	params.RequestType = sp.RequestType
	params.NarrationMode = sp.NarrationMode
}
