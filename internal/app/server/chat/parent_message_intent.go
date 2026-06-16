package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"xiaozhi-esp32-server-golang/internal/domain/llm"
	"xiaozhi-esp32-server-golang/internal/pool"
	log "xiaozhi-esp32-server-golang/logger"

	"github.com/cloudwego/eino/schema"
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

var parentMessageIntentJSONPattern = regexp.MustCompile(`\{[^}]*"intent"\s*:\s*"(play|unknown)"[^}]*\}`)

func classifyParentMessageIntent(text string) parentMessageIntent {
	normalized := strings.ToLower(strings.TrimSpace(text))
	if normalized == "" {
		return parentMessageIntentUnknown
	}

	negativeKeywords := []string{"不要", "不用", "别", "不听", "不想", "算了", "否", "不需要", "不听啦", "不想听", "稍后", "等等", "先不"}
	for _, kw := range negativeKeywords {
		if strings.Contains(normalized, kw) {
			return parentMessageIntentNegative
		}
	}

	affirmativeKeywords := []string{"要", "好的", "好", "听", "是", "嗯", "行", "可以", "想听", "听听", "播", "读", "播放"}
	for _, kw := range affirmativeKeywords {
		if strings.Contains(normalized, kw) {
			return parentMessageIntentAffirmative
		}
	}
	return parentMessageIntentUnknown
}

func (c *ChatManager) classifyParentMessageIntentWithLLM(ctx context.Context, asrText string) parentMessageIntent {
	if c == nil || c.clientState == nil {
		return classifyParentMessageIntent(asrText)
	}
	intent, err := c.callLLMSyncText(ctx,
		"你是意图分类器。用户是儿童，正在决定是否收听家长留言。只输出 JSON，格式为 {\"intent\":\"play\"} 或 {\"intent\":\"unknown\"}，不要输出其它内容。",
		"用户说："+strings.TrimSpace(asrText),
	)
	if err != nil {
		log.Warnf("设备 %s 家长留言 LLM 意图识别失败，降级关键词: %v", c.DeviceID, err)
		return classifyParentMessageIntent(asrText)
	}
	if parsed := parseParentMessageIntentJSON(intent); parsed != parentMessageIntentUnknown {
		return parsed
	}
	return classifyParentMessageIntent(asrText)
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

func (c *ChatManager) generateParentMessageAskPrompt(ctx context.Context, familyRole string, createdAt time.Time) string {
	now := time.Now()
	fallback := buildAskPromptFallback(familyRole, createdAt, now)
	if c == nil || c.clientState == nil {
		return fallback
	}
	systemPrompt := strings.TrimSpace(c.clientState.SystemPrompt)
	prompt, err := c.callLLMSyncText(ctx,
		systemPrompt+"\n请用一句温柔、简短、适合儿童的话，询问孩子是否要收听家长留言。只输出这一句话，不要解释。",
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
