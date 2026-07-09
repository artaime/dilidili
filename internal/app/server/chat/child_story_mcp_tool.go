package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"dili-esp32-server-golang/internal/domain/story"
	log "dili-esp32-server-golang/logger"

	"github.com/spf13/viper"
)

const localMcpChildStoryToolName = "create_child_story"

type CreateChildStoryParams struct {
	Action         string             `json:"action" description:"操作：generate(生成)|replay(复播)|resume(续讲)|list_recent(近期列表)" required:"true"`
	StoryRef       string             `json:"story_ref,omitempty" description:"故事引用：last|last_night|favorite|story_id"`
	FromBeginning  *bool              `json:"from_beginning,omitempty" description:"复播是否从头开始，默认 true"`
	RequestType    string             `json:"request_type,omitempty" description:"故事类型：classic|myth|fable|fairy_tale|bedtime|original，由你根据用户意图判断"`
	NarrationMode  string             `json:"narration_mode,omitempty" description:"讲述方式：canonical=讲经典/神话正篇勿魔改；creative=原创或新编"`
	Theme          string             `json:"theme,omitempty" description:"规范化故事名或主题，如龟兔赛跑、女娲补天、小恐龙冒险"`
	Style         string             `json:"style,omitempty" description:"故事风格"`
	AgeBand       string             `json:"age_band,omitempty" description:"年龄档：preschool|primary_low|primary_high|junior_high"`
	AgeYears      *int               `json:"age_years,omitempty" description:"孩子年龄（岁）"`
	IsBedtime     *bool              `json:"is_bedtime,omitempty" description:"是否睡前故事"`
	DurationHint  string             `json:"duration_hint,omitempty" description:"篇幅：short|medium|long"`
	Interests     []string           `json:"interests,omitempty" description:"兴趣爱好"`
	MemoryHints   []story.MemoryHint `json:"memory_hints,omitempty" description:"来自记忆的候选参数（须标注 confidence）"`
	UserSaidCasual bool               `json:"user_said_casual,omitempty" description:"用户说随便/都可以"`
}

func registerChildStoryMCPTool() {
	if viper.IsSet("local_mcp.create_child_story") && !viper.GetBool("local_mcp.create_child_story") {
		return
	}
	if viper.IsSet("story.enabled") && !viper.GetBool("story.enabled") {
		return
	}
	desc := "当用户要求讲故事、听故事、编故事、童话、寓言、神话，或复播/续讲时使用。" +
		"你必须判断：theme（规范故事名）、request_type（classic|myth|fable|fairy_tale|bedtime|original）、narration_mode（canonical 讲正篇 / creative 新编）。" +
		"用户点名经典/神话/寓言时用 canonical；用户要编故事、随便讲、或原创主题用 creative。" +
		"纯闲聊不要调用。事实问答用 search_knowledge。" +
		"generate 新故事；replay（story_ref: last|last_night|favorite）；resume 续讲；list_recent 列表。" +
		"缺年龄等信息时返回 need_params，请短句追问，不要自行编造正文。"
	if err := RegisterLocalMcpFunc(
		localMcpChildStoryToolName,
		desc,
		CreateChildStoryParams{Action: "generate"},
		childStoryHandler,
	); err != nil {
		log.Errorf("注册儿童故事 MCP 工具失败: %v", err)
	}
}

func childStoryHandler(ctx context.Context, argumentsInJSON string) (string, error) {
	log.Info("执行儿童故事工具")

	var params CreateChildStoryParams
	if argumentsInJSON != "" {
		if err := json.Unmarshal([]byte(argumentsInJSON), &params); err != nil {
			response := NewErrorResponse(localMcpChildStoryToolName, "参数解析失败", "PARSE_ERROR", "请检查 action 等参数格式")
			return response.ToJSON()
		}
	}
	params.Action = strings.TrimSpace(params.Action)
	if params.Action == "" {
		params.Action = story.ActionGenerate
	}

	opValue := ctx.Value("chat_session_operator")
	if opValue == nil {
		return "", fmt.Errorf("从context中未找到chat_session_operator")
	}
	operator, ok := opValue.(ChatSessionOperator)
	if !ok {
		return "", fmt.Errorf("chat_session_operator 类型错误")
	}

	result, err := operator.LocalMcpCreateChildStory(ctx, &params)
	if err != nil {
		response := NewErrorResponse(localMcpChildStoryToolName, fmt.Sprintf("故事处理失败: %v", err), "STORY_FAILED", "请稍后重试")
		return response.ToJSON()
	}

	data := map[string]interface{}{
		"status":            result.Status,
		"story_id":          result.StoryID,
		"title":             result.Title,
		"text_to_speak":     result.TextToSpeak,
		"segments":          result.Segments,
		"start_segment":     result.StartSegment,
		"missing":           result.Missing,
		"candidates":        result.Candidates,
		"suggest_new_story": result.SuggestNewStory,
		"meta":              result.Meta,
	}
	msg := result.Message
	if result.Status == story.StatusStreaming {
		data := map[string]interface{}{
			"status": result.Status,
		}
		response := NewStoryStreamingResponse(localMcpChildStoryToolName, data)
		return response.ToJSON()
	}
	if result.TextToSpeak != "" && (result.Status == story.StatusReady || result.Status == story.StatusReplay || result.Status == story.StatusResume) {
		response := NewStoryNarrationResponse(localMcpChildStoryToolName, data, result.TextToSpeak)
		return response.ToJSON()
	}
	response := NewContentResponse(localMcpChildStoryToolName, data, msg)
	return response.ToJSON()
}

func (c *ChatManager) getStoryService() *story.Service {
	if c == nil {
		return story.NewService(nil)
	}
	if c.storyService == nil {
		c.storyService = story.NewService(c.storyGenerateFunc)
	}
	return c.storyService
}

func (c *ChatManager) storyGenerateFunc(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	return c.callLLMSyncTextForStory(ctx, systemPrompt, userPrompt)
}

func (c *ChatManager) LocalMcpCreateChildStory(ctx context.Context, params *CreateChildStoryParams) (*story.ToolResult, error) {
	if c == nil || c.clientState == nil || params == nil {
		return nil, fmt.Errorf("会话不可用")
	}
	normalizeCreateChildStoryParams(params)
	svc := c.getStoryService()

	// 按主题复播/续讲但 Redis 无正文时，改走流式生成（避免同步生成或空内容提示）。
	if svc.Config().StreamEnabled && (params.Action == story.ActionReplay || params.Action == story.ActionResume) {
		if theme := story.NormalizeThemeKey(params.Theme); theme != "" {
			if _, err := svc.Store().FindLatestByTheme(ctx, c.DeviceID, theme, true); err != nil {
				params.Action = story.ActionGenerate
			}
		}
	}

	if params.Action == story.ActionGenerate && svc.Config().StreamEnabled {
		if err := c.streamGenerateChildStory(ctx, params); err != nil {
			return nil, err
		}
		return &story.ToolResult{Status: story.StatusStreaming, Message: "故事生成中"}, nil
	}

	req := c.buildStoryToolRequest(params)

	if params.Action == story.ActionResume && svc.Config().StreamEnabled {
		rec, resolveErr := svc.ResolveResumeRecord(ctx, req)
		if resolveErr == nil && rec != nil && story.HasStoryContent(rec) && !story.IsGenerationComplete(rec) {
			if err := c.streamContinueChildStory(ctx, params, rec); err != nil {
				return nil, err
			}
			return &story.ToolResult{Status: story.StatusStreaming, StoryID: rec.StoryID, Message: "故事续写中"}, nil
		}
	}

	result, err := svc.Handle(ctx, req)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (c *ChatManager) LocalMcpUpdateStoryProgress(ctx context.Context, storyID string, pos story.PlayPosition, interrupted, completed bool) error {
	if c == nil {
		return fmt.Errorf("chat manager 不可用")
	}
	return c.getStoryService().UpdatePlaybackProgress(ctx, c.DeviceID, storyID, pos, interrupted, completed)
}
