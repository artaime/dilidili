package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"dili-esp32-server-golang/internal/domain/llm"
	"dili-esp32-server-golang/internal/pool"
	log "dili-esp32-server-golang/logger"

	"github.com/cloudwego/eino/schema"
	"github.com/google/uuid"
	"github.com/spf13/viper"
)

type parentMessageIntent int

const (
	parentMessageIntentUnknown parentMessageIntent = iota
	parentMessageIntentAffirmative
	parentMessageIntentNegative
)

type parentMessageIntentJSON struct {
	Intent string `json:"intent"`
}

var parentMessageIntentJSONPattern = regexp.MustCompile(`\{[^}]*"intent"\s*:\s*"(play|skip|unknown)"[^}]*\}`)

func (c *ChatManager) classifyParentMessageIntentWithLLM(ctx context.Context, asrText string) parentMessageIntent {
	if c == nil || c.clientState == nil {
		return parentMessageIntentUnknown
	}
	asrText = strings.TrimSpace(asrText)
	if asrText == "" {
		return parentMessageIntentUnknown
	}
	intent, err := c.callLLMSyncText(ctx,
		"你是意图分类器。用户是儿童，正在决定是否收听家长留言。根据用户的话判断意愿，只输出 JSON，不要输出其它内容。\n"+
			"- 明确同意收听（如：好、要、听、播放、可以、嗯）：{\"intent\":\"play\"}\n"+
			"- 明确拒绝收听（如：不要、不想、不用、算了、不听）：{\"intent\":\"skip\"}\n"+
			"- 与留言无关、意图不清、在聊别的话题：{\"intent\":\"unknown\"}",
		"用户说："+asrText,
	)
	if err != nil {
		log.Warnf("设备 %s 家长留言 LLM 意图识别失败: %v", c.DeviceID, err)
		return parentMessageIntentUnknown
	}
	return parseParentMessageIntentJSON(intent)
}

func parseParentMessageIntentJSON(raw string) parentMessageIntent {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return parentMessageIntentUnknown
	}
	match := parentMessageIntentJSONPattern.FindString(raw)
	if match == "" {
		var payload parentMessageIntentJSON
		if err := json.Unmarshal([]byte(raw), &payload); err != nil {
			return parentMessageIntentUnknown
		}
		return mapIntentString(payload.Intent)
	}
	var payload parentMessageIntentJSON
	if err := json.Unmarshal([]byte(match), &payload); err != nil {
		return parentMessageIntentUnknown
	}
	return mapIntentString(payload.Intent)
}

func mapIntentString(intent string) parentMessageIntent {
	switch strings.ToLower(strings.TrimSpace(intent)) {
	case "play":
		return parentMessageIntentAffirmative
	case "skip":
		return parentMessageIntentNegative
	default:
		return parentMessageIntentUnknown
	}
}

func buildParentMessageIntentTransitionFallback() string {
	return "好的，等你准备好了再告诉我要不要听留言哦。"
}

func (c *ChatManager) generateParentMessageIntentTransition(ctx context.Context, userText, familyRole string, createdAt time.Time) string {
	fallback := buildParentMessageIntentTransitionFallback()
	if c == nil || c.clientState == nil {
		return fallback
	}
	now := time.Now()
	systemPrompt := strings.TrimSpace(c.clientState.SystemPrompt)
	systemPrompt += "\n孩子正在决定是否收听家长留言，但刚才的话没有明确表态。"
	systemPrompt += "用一两句温柔、简短、适合儿童的话自然回应用户刚说的话。"
	systemPrompt += "可以轻轻再问要不要听这条留言，但不要告诉孩子该说什么词或口令，不要替孩子做决定，不要播放留言内容。只输出这一句话。"

	userPrompt := fmt.Sprintf(
		"家长身份：%s。留言时间：%s。用户说：%s",
		normalizeFamilyRoleLabel(familyRole),
		formatChildFriendlyTime(createdAt, now),
		strings.TrimSpace(userText),
	)
	reply, err := c.callLLMSyncText(ctx, systemPrompt, userPrompt)
	if err != nil || strings.TrimSpace(reply) == "" {
		log.Warnf("设备 %s 家长留言过渡语生成失败，使用模板: %v", c.DeviceID, err)
		return fallback
	}
	return strings.TrimSpace(reply)
}

func (c *ChatManager) generateParentMessageAskPrompt(ctx context.Context, familyRole string, createdAt time.Time) string {
	now := time.Now()
	fallback := buildAskPromptFallback(familyRole, createdAt, now)
	if c == nil || c.clientState == nil {
		return fallback
	}
	systemPrompt := strings.TrimSpace(c.clientState.SystemPrompt)
	systemPrompt += "\n请用一句温柔、简短、适合儿童的话，告知孩子有家长留言，并自然询问是否要播放。"
	systemPrompt += "不要告诉孩子该说什么词或口令。只输出这一句话。"
	prompt, err := c.callLLMSyncText(ctx,
		systemPrompt,
		fmt.Sprintf("家长身份：%s。留言时间：%s。参考句式：%s", normalizeFamilyRoleLabel(familyRole), formatChildFriendlyTime(createdAt, now), fallback),
	)
	if err != nil || strings.TrimSpace(prompt) == "" {
		log.Warnf("设备 %s 生成家长留言询问语失败，使用模板: %v", c.DeviceID, err)
		return fallback
	}
	return strings.TrimSpace(prompt)
}

func (c *ChatManager) callLLMSyncText(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	if c.clientState == nil || c.clientState.DeviceConfig.Llm.Provider == "" {
		return "", fmt.Errorf("LLM 未配置")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	callCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()

	llmWrapper, err := pool.Acquire[llm.LLMProvider](
		"llm",
		c.clientState.DeviceConfig.Llm.Provider,
		c.clientState.DeviceConfig.Llm.Config,
	)
	if err != nil {
		return "", err
	}
	defer pool.Release(llmWrapper)

	dialogue := []*schema.Message{
		schema.SystemMessage(systemPrompt),
		schema.UserMessage(userPrompt),
	}
	msgChan := llmWrapper.GetProvider().ResponseWithContext(callCtx, c.clientState.SessionID, dialogue, nil)

	var builder strings.Builder
	for msg := range msgChan {
		if msg == nil {
			continue
		}
		builder.WriteString(msg.Content)
	}
	return strings.TrimSpace(builder.String()), nil
}

func (c *ChatManager) callLLMSyncTextForStory(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	var builder strings.Builder
	err := c.callLLMStreamForStory(ctx, systemPrompt, userPrompt, func(chunk string) error {
		builder.WriteString(chunk)
		return nil
	})
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(builder.String()), nil
}

func (c *ChatManager) callLLMStreamForStory(ctx context.Context, systemPrompt, userPrompt string, onChunk func(chunk string) error) error {
	if c.clientState == nil || c.clientState.DeviceConfig.Llm.Provider == "" {
		return fmt.Errorf("LLM 未配置")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if onChunk == nil {
		return fmt.Errorf("onChunk is nil")
	}
	callCtx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()

	llmConfig := cloneLLMConfigForStory(c.clientState.DeviceConfig.Llm.Config)

	llmWrapper, err := pool.Acquire[llm.LLMProvider](
		"llm",
		c.clientState.DeviceConfig.Llm.Provider,
		llmConfig,
	)
	if err != nil {
		return err
	}
	defer pool.Release(llmWrapper)

	dialogue := []*schema.Message{
		schema.SystemMessage(systemPrompt),
		schema.UserMessage(userPrompt),
	}
	storySessionID := fmt.Sprintf("%s:story-gen:%s", c.clientState.SessionID, uuid.NewString())
	msgChan := llmWrapper.GetProvider().ResponseWithContext(callCtx, storySessionID, dialogue, nil)

	chunkCount := 0
	for msg := range msgChan {
		if msg == nil || msg.Content == "" {
			continue
		}
		chunkCount++
		if err := onChunk(msg.Content); err != nil {
			log.Warnf("设备 %s 故事 LLM 流中断 session=%s chunks=%d err=%v",
				c.DeviceID, storySessionID, chunkCount, err)
			return err
		}
	}
	if chunkCount == 0 {
		log.Warnf("设备 %s 故事 LLM 流无内容 session=%s", c.DeviceID, storySessionID)
	}
	return nil
}

func cloneLLMConfigForStory(base map[string]interface{}) map[string]interface{} {
	cfg := make(map[string]interface{}, len(base)+1)
	for k, v := range base {
		cfg[k] = v
	}
	minStoryTokens := 2048
	if viper.IsSet("story.generate_max_tokens") {
		minStoryTokens = viper.GetInt("story.generate_max_tokens")
	}
	for _, key := range []string{"max_tokens", "max_token"} {
		if v, ok := cfg[key]; ok {
			switch n := v.(type) {
			case int:
				if n < minStoryTokens {
					cfg[key] = minStoryTokens
				}
			case float64:
				if int(n) < minStoryTokens {
					cfg[key] = minStoryTokens
				}
			}
			return cfg
		}
	}
	cfg["max_tokens"] = minStoryTokens
	return cfg
}
