package chat

import (
	"context"
	"fmt"
	"strings"
	"time"

	"dili-esp32-server-golang/internal/domain/chat/intent"
	llm_common "dili-esp32-server-golang/internal/domain/llm/common"
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
	if resp.NeedsDialogue {
		log.Infof("设备 %s 意图需主对话 needs_dialogue intent=%s text=%q", c.DeviceID, resp.Intent, text)
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
	case intent.IntentDevice, intent.IntentGeneral:
		// device / general：一律交主 LLM（带 short_context），禁止旁路单句编答。
		log.Infof("设备 %s intent=%s 交主对话处理 text=%q", c.DeviceID, resp.Intent, text)
		return false, nil
	default:
		return false, nil
	}
}

func (c *ChatManager) classifyUserIntent(ctx context.Context, text string) (intent.RouterResponse, float64, error) {
	if c == nil || c.clientState == nil {
		return intent.RouterResponse{}, 0, fmt.Errorf("client state unavailable")
	}
	systemPrompt := intent.BuildClassifierSystemPrompt(c.clientState.SystemPrompt)
	userPrompt := buildClassifierUserPrompt(c.clientState, text)
	raw, err := c.callLLMSyncText(ctx, systemPrompt, userPrompt)
	if err != nil {
		return intent.RouterResponse{}, 0, err
	}
	resp, err := intent.ParseRouterResponse(raw)
	if err != nil {
		return intent.RouterResponse{}, 0, err
	}
	confidence, err := intent.ParseConfidence(resp.Confidence)
	if err != nil {
		confidence = 0.5
	}
	return resp, confidence, nil
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

// HandleGeneralIntent 保留供测试/兼容；路由已不再截获 general。
func (c *ChatManager) HandleGeneralIntent(ctx context.Context, data intent.GeneralData) error {
	_ = ctx
	reply := strings.TrimSpace(data.Reply)
	if reply == "" {
		return fmt.Errorf("general intent missing reply")
	}
	if rewritten, ok := llm_common.MaybeRewriteUngroundedActionClaim(reply, false); ok {
		log.Infof("设备 %s general 意图回复含虚构操作，已改写", c.DeviceID)
		reply = rewritten
	} else if rewritten, ok := llm_common.MaybeRewriteUngroundedCapabilityOffer(reply, nil); ok {
		log.Infof("设备 %s general 意图回复含虚构能力推销，已改写", c.DeviceID)
		reply = rewritten
	}
	return c.InjectMessage(reply, true, true)
}
