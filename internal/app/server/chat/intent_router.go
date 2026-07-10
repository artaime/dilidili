package chat

import (
	"context"
	"fmt"
	"strings"
	"time"

	"dili-esp32-server-golang/internal/domain/chat/intent"
	"dili-esp32-server-golang/internal/domain/speaker"
	log "dili-esp32-server-golang/logger"
)

func (c *ChatManager) RouteUserIntent(ctx context.Context, text string, speakerResult *speaker.IdentifyResult) (bool, error) {
	_ = speakerResult
	if c == nil {
		return false, nil
	}
	if handled, err := c.tryHandleChildStoryRequest(ctx, text); handled {
		return true, err
	}
	if !intent.IntentRouterEnabled() {
		return false, nil
	}
	if c.parentMessageState != nil {
		return false, nil
	}

	resp, confidence, err := c.classifyUserIntent(ctx, text)
	if err != nil {
		log.Debugf("设备 %s 意图路由降级: %v", c.DeviceID, err)
		return false, nil
	}
	if confidence < intent.MinConfidence() {
		log.Infof("设备 %s 意图置信度不足 intent=%s confidence=%.2f text=%q", c.DeviceID, resp.Intent, confidence, text)
		return false, nil
	}

	log.Infof("设备 %s 意图路由 intent=%s confidence=%.2f text=%q", c.DeviceID, resp.Intent, confidence, text)
	switch resp.Intent {
	case intent.IntentMsgInquiry:
		data, _ := intent.ParseData[intent.MsgInquiryData](resp.Data)
		return true, c.HandleMsgInquiry(ctx, data)
	case intent.IntentMsgPlay:
		data, _ := intent.ParseData[intent.MsgPlayData](resp.Data)
		return true, c.HandleMsgPlay(ctx, data)
	case intent.IntentGeneral:
		data, err := intent.ParseData[intent.GeneralData](resp.Data)
		if err != nil || strings.TrimSpace(data.Reply) == "" {
			return false, nil
		}
		return true, c.HandleGeneralIntent(ctx, data)
	default:
		return false, nil
	}
}

func (c *ChatManager) classifyUserIntent(ctx context.Context, text string) (intent.RouterResponse, float64, error) {
	if c == nil || c.clientState == nil {
		return intent.RouterResponse{}, 0, fmt.Errorf("client state unavailable")
	}
	systemPrompt := intent.BuildClassifierSystemPrompt(c.clientState.SystemPrompt)
	raw, err := c.callLLMSyncText(ctx, systemPrompt, "用户说："+strings.TrimSpace(text))
	if err != nil {
		if fallback, ok := intent.FallbackClassify(text); ok {
			conf, confErr := intent.ParseConfidence(fallback.Confidence)
			if confErr != nil {
				conf = 0.75
			}
			return fallback, conf, nil
		}
		return intent.RouterResponse{}, 0, err
	}
	resp, err := intent.ParseRouterResponse(raw)
	if err != nil {
		if fallback, ok := intent.FallbackClassify(text); ok {
			conf, confErr := intent.ParseConfidence(fallback.Confidence)
			if confErr != nil {
				conf = 0.75
			}
			return fallback, conf, nil
		}
		return intent.RouterResponse{}, 0, err
	}
	confidence, err := intent.ParseConfidence(resp.Confidence)
	if err != nil {
		confidence = 0.5
	}
	return resp, confidence, nil
}

func (c *ChatManager) HandleGeneralIntent(ctx context.Context, data intent.GeneralData) error {
	reply := strings.TrimSpace(data.Reply)
	if reply == "" {
		return fmt.Errorf("general intent missing reply")
	}
	return c.InjectMessage(reply, true, true)
}

func (c *ChatManager) buildParentMessageSummary(messages []parentMessageItem) string {
	if len(messages) == 0 {
		return "现在没有新留言哦。"
	}
	now := time.Now()
	parts := make([]string, 0, len(messages))
	limit := 3
	for i, msg := range messages {
		if i >= limit {
			break
		}
		role := normalizeFamilyRoleLabel(msg.FamilyRole)
		parts = append(parts, fmt.Sprintf("%s%s", role, formatChildFriendlyTime(msg.CreatedAt, now)))
	}
	summary := strings.Join(parts, "、")
	if len(messages) > limit {
		return fmt.Sprintf("你有%d条新留言，比如%s等；想听可以说播放留言。", len(messages), summary)
	}
	if len(messages) == 1 {
		return fmt.Sprintf("你有1条新留言，%s；想听可以说播放留言。", summary)
	}
	return fmt.Sprintf("你有%d条新留言，%s；想听可以说播放留言。", len(messages), summary)
}
