package chat

import (
	"context"
	"strings"

	"dili-esp32-server-golang/internal/domain/story"
	log "dili-esp32-server-golang/logger"

	"github.com/spf13/viper"
)

func childStoryRoutingEnabled() bool {
	if viper.IsSet("story.enabled") && !viper.GetBool("story.enabled") {
		return false
	}
	if viper.IsSet("local_mcp.create_child_story") && !viper.GetBool("local_mcp.create_child_story") {
		return false
	}
	if viper.IsSet("story.llm_route_enabled") && !viper.GetBool("story.llm_route_enabled") {
		return false
	}
	return true
}

func (c *ChatManager) tryHandleChildStoryRequest(ctx context.Context, text string) (bool, error) {
	if c == nil {
		return false, nil
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return false, nil
	}

	routingOn := childStoryRoutingEnabled()
	followupOn := c.storyFollowupEnabled()
	if !routingOn && !followupOn {
		return false, nil
	}

	intent, classified := story.IntentResult{}, false
	if routingOn || followupOn {
		intent, classified = c.classifyStoryIntent(ctx, text)
	}

	// 澄清多轮：用户补充线索；若改口要求讲/复播则退出澄清走现网。
	if followupOn && c.isFollowupClarifying() {
		if classified && isStoryPlaybackAction(intent.Action) {
			c.clearFollowupClarify()
		} else {
			log.Infof("设备 %s 故事追问澄清轮 text=%q", c.DeviceID, text)
			return true, c.handleFollowupClarifyTurn(ctx, text, intent, classified)
		}
	}

	// 分类器命中追问（无关键词快路径）
	if followupOn && classified && intent.IsStoryFollowup {
		handled, err := c.handleStoryFollowup(ctx, text, intent)
		if !handled {
			return false, nil
		}
		return true, err
	}

	if !routingOn || !classified || intent.IsStoryFollowup || !isStoryPlaybackAction(intent.Action) {
		return false, nil
	}

	params := intentResultToCreateParams(intent)
	log.Infof("设备 %s LLM 故事路由 action=%s theme=%q type=%s narration=%s text=%q",
		c.DeviceID, params.Action, params.Theme, params.RequestType, params.NarrationMode, text)

	if params.Action == story.ActionGenerate && c.shouldRejectStoryGenerateWhileActive() {
		log.Infof("设备 %s 故事播报中忽略重复 generate 请求 text=%q", c.DeviceID, text)
		return true, nil
	}

	result, err := c.LocalMcpCreateChildStory(ctx, params)
	if err != nil {
		log.Errorf("设备 %s 儿童故事直达失败: %v", c.DeviceID, err)
		return false, nil
	}

	return true, c.deliverChildStoryResult(ctx, result)
}

func (c *ChatManager) deliverChildStoryResult(ctx context.Context, result *story.ToolResult) error {
	if result == nil {
		return nil
	}

	switch result.Status {
	case story.StatusNeedParams, story.StatusNotFound, story.StatusCandidates:
		msg := strings.TrimSpace(result.Message)
		if msg == "" {
			msg = "要不要听一个新的故事呢？"
		}
		return c.InjectMessage(msg, true, true)
	case story.StatusReady, story.StatusReplay, story.StatusResume:
		text := strings.TrimSpace(result.TextToSpeak)
		if text == "" {
			if len(result.Candidates) > 0 {
				return c.deliverChildStoryResult(ctx, &story.ToolResult{
					Status:     story.StatusCandidates,
					Candidates: result.Candidates,
					Message:    strings.TrimSpace(result.Message),
				})
			}
			msg := strings.TrimSpace(result.Message)
			if msg == "" {
				msg = "故事准备好了，但内容为空，我们换一个试试吧。"
			}
			return c.InjectMessage(msg, true, true)
		}
		if result.SuggestNewStory {
			_ = c.getStoryService().Store().MarkSuggestShown(ctx, c.DeviceID, result.StoryID)
		}
		return c.narrateChildStory(ctx, result, text)
	case story.StatusStreaming:
		log.Infof("设备 %s 儿童故事流式生成已启动", c.DeviceID)
		return nil
	default:
		if msg := strings.TrimSpace(result.Message); msg != "" {
			return c.InjectMessage(msg, true, true)
		}
		return nil
	}
}

func (c *ChatManager) narrateChildStory(ctx context.Context, result *story.ToolResult, text string) error {
	c.cancelRetainedSessionCleanup("child_story_narration")
	session, err := c.ensureSession()
	if err != nil {
		return err
	}
	if result != nil {
		session.ActivateStoryPlayback(result)
	}
	body := strings.TrimSpace(text)
	speakText := prependStoryNarrationIntro(c.getStoryService().Config(), result, body)
	log.Infof("设备 %s 开始直接 TTS 朗读故事 story_id=%s len=%d intro=%t",
		c.DeviceID, result.StoryID, len([]rune(body)), speakText != body)
	storyID := result.StoryID
	return session.AddTextToTTSQueueWithOptions(speakText, llmResponseChannelOptions{
		ttsTurnEndPolicy: ttsTurnEndPolicyNone,
		onEndFunc: func(err error, _ ...any) {
			if session.IsStoryPlaybackActive() {
				// 进度只计正文（不含标题过渡语）
				session.OnStoryTextSent(body)
				session.OnStoryPlaybackFinished(err == nil)
			}
			if err == nil && storyID != "" {
				if updater := session.storyProgressUpdater; updater != nil {
					updater.RememberStoryForFollowUp(session.ctx, session, storyID, body, true)
				}
			}
		},
	})
}

func (l *LLMManager) deliverChildStoryFromTool(ctx context.Context, narrationText string, result *story.ToolResult) error {
	if l == nil || l.session == nil {
		return nil
	}
	body := strings.TrimSpace(narrationText)
	if body == "" {
		return nil
	}
	if result != nil {
		l.session.ActivateStoryPlayback(result)
	}
	cfg := story.LoadConfig()
	speakText := prependStoryNarrationIntro(cfg, result, body)
	log.Infof("设备 %s MCP 工具返回故事，直接 TTS 朗读 len=%d intro=%t",
		l.clientState.DeviceID, len([]rune(body)), speakText != body)
	session := l.session
	return session.AddTextToTTSQueueWithOptions(speakText, llmResponseChannelOptions{
		ttsTurnEndPolicy: ttsTurnEndPolicyNone,
		onEndFunc: func(err error, _ ...any) {
			if session.IsStoryPlaybackActive() {
				session.OnStoryTextSent(body)
				session.OnStoryPlaybackFinished(err == nil)
			}
			if updater := session.storyProgressUpdater; updater != nil {
				storyID := ""
				if result != nil {
					storyID = result.StoryID
				}
				updater.RememberStoryForFollowUp(ctx, session, storyID, body, err == nil)
			}
		},
	})
}

// prependStoryNarrationIntro 在正文前拼接礼貌标题过渡语（进度仍只计 body）。
func prependStoryNarrationIntro(cfg story.Config, result *story.ToolResult, body string) string {
	body = strings.TrimSpace(body)
	if body == "" || story.HasNarrationIntroPrefix(body) {
		return body
	}
	title := ""
	status := ""
	if result != nil {
		title = strings.TrimSpace(result.Title)
		status = result.Status
	}
	var intro string
	switch status {
	case story.StatusResume:
		// 正文已带「上次讲到…」续讲语时不再叠一句「继续讲标题」，避免两句开场。
		if story.HasResumeTransitionPrefix(body) {
			return body
		}
		intro = story.BuildResumeTitleLead(title, cfg)
	default:
		intro = story.BuildNarrationIntro(title, cfg)
	}
	if intro == "" {
		return body
	}
	return intro + body
}

func storyToolResultFromMap(data map[string]interface{}) *story.ToolResult {
	if data == nil {
		return nil
	}
	result := &story.ToolResult{}
	if v, ok := data["status"].(string); ok {
		result.Status = v
	}
	if v, ok := data["story_id"].(string); ok {
		result.StoryID = v
	}
	if v, ok := data["title"].(string); ok {
		result.Title = v
	}
	if v, ok := data["text_to_speak"].(string); ok {
		result.TextToSpeak = v
	}
	if v, ok := data["start_segment"].(float64); ok {
		result.StartSegment = int(v)
	}
	if segs, ok := data["segments"].([]interface{}); ok {
		for _, s := range segs {
			if str, ok := s.(string); ok {
				result.Segments = append(result.Segments, str)
			}
		}
	}
	return result
}
