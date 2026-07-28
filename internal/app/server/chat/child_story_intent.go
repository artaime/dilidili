package chat

import (
	"context"
	"strings"

	"dili-esp32-server-golang/internal/domain/story"
	log "dili-esp32-server-golang/logger"
)

// classifyStoryIntent 故事意图分类（含讲故事与追问）。
func (c *ChatManager) classifyStoryIntent(ctx context.Context, text string) (story.IntentResult, bool) {
	if c == nil {
		return story.IntentResult{}, false
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return story.IntentResult{}, false
	}

		userPrompt := buildClassifierUserPrompt(c.clientState, text)
		raw, err := c.callLLMSyncText(ctx, story.BuildStoryIntentSystemPrompt(), userPrompt)
		if err != nil {
			log.Warnf("设备 %s 故事 LLM 意图识别失败: %v", c.DeviceID, err)
			return story.IntentResult{}, false
		}

		intent, err := story.ParseStoryIntentJSON(raw)
		if err != nil {
			log.Warnf("设备 %s 故事意图 JSON 解析失败 raw=%q err=%v", c.DeviceID, raw, err)
			return story.IntentResult{}, false
		}
		if intent.Confidence < story.StoryIntentMinConfidence {
			return intent, false
		}
		if intent.NeedsDialogue || intent.Action == story.ActionNone || intent.Action == "none" {
			log.Infof("设备 %s 故事意图交主对话 needs_dialogue=%v action=%s text=%q",
				c.DeviceID, intent.NeedsDialogue, intent.Action, text)
			return intent, false
		}
		if intent.IsStoryFollowup || intent.Action == story.ActionFollowup {
		intent.Action = story.ActionFollowup
		intent.IsStoryFollowup = true
		intent.IsStoryRequest = false
		log.Infof("设备 %s 故事追问意图 theme=%q type=%s confidence=%.2f",
			c.DeviceID, intent.Canonical, intent.StoryType, intent.Confidence)
		return intent, true
	}
	if !intent.IsStoryRequest {
		return intent, false
	}
	if intent.Action == "" || intent.Action == "none" {
		return intent, false
	}
	log.Infof("设备 %s 故事 LLM 意图 action=%s theme=%q theme_raw=%q type=%s narration=%s confidence=%.2f",
		c.DeviceID, intent.Action, intent.Canonical, intent.Theme, intent.StoryType, intent.NarrationMode, intent.Confidence)
	return intent, true
}

func (c *ChatManager) classifyChildStoryIntent(ctx context.Context, text string) (*CreateChildStoryParams, bool) {
	intent, ok := c.classifyStoryIntent(ctx, text)
	if !ok || intent.IsStoryFollowup {
		return nil, false
	}
	return intentResultToCreateParams(intent), true
}

func intentResultToCreateParams(intent story.IntentResult) *CreateChildStoryParams {
	sp := story.IntentToStoryParams(intent)
	theme, themeRaw := story.ResolveIntentTheme(intent)
	params := &CreateChildStoryParams{
		Action:         strings.TrimSpace(intent.Action),
		StoryRef:       strings.TrimSpace(intent.StoryRef),
		RequestType:    sp.RequestType,
		NarrationMode:  sp.NarrationMode,
		Theme:          theme,
		ThemeRaw:       themeRaw,
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

func isStoryPlaybackAction(action string) bool {
	switch strings.TrimSpace(action) {
	case story.ActionGenerate, story.ActionReplay, story.ActionResume, story.ActionListRecent:
		return true
	default:
		return false
	}
}
