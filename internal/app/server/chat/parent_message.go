package chat

import (
	"context"
	"strings"
	"time"

	parentmsg "xiaozhi-esp32-server-golang/internal/data/parentmessage"
	log "xiaozhi-esp32-server-golang/logger"
)

const (
	parentMessageAskPrompt      = "你有来自家长的留言，要听吗？说「要」或「不要」。"
	parentMessageRetryPrompt    = "没听清，请说「要」或「不要」。"
	parentMessageMaxRetry       = 1
	parentMessageNotifyAttempts = 10
)

type parentMessageFlowState struct {
	message    *parentmsg.PendingMessage
	retryCount int
}

type parentMessageIntent int

const (
	parentMessageIntentUnknown parentMessageIntent = iota
	parentMessageIntentAffirmative
	parentMessageIntentNegative
)

func classifyParentMessageIntent(text string) parentMessageIntent {
	normalized := strings.ToLower(strings.TrimSpace(text))
	if normalized == "" {
		return parentMessageIntentUnknown
	}

	negativeKeywords := []string{"不要", "不用", "别", "不听", "不想", "算了", "否", "不需要", "不听啦"}
	for _, kw := range negativeKeywords {
		if strings.Contains(normalized, kw) {
			return parentMessageIntentNegative
		}
	}

	affirmativeKeywords := []string{"要", "好的", "好", "听", "是", "嗯", "行", "可以", "想听", "听听", "播", "读"}
	for _, kw := range affirmativeKeywords {
		if strings.Contains(normalized, kw) {
			return parentMessageIntentAffirmative
		}
	}
	return parentMessageIntentUnknown
}

func (c *ChatManager) SetParentMessageClient(client *parentmsg.Client) {
	if c == nil {
		return
	}
	c.parentMessageClient = client
}

func (c *ChatManager) NotifyPendingParentMessages() {
	if c == nil || c.parentMessageClient == nil {
		return
	}
	if !c.parentMessageNotifyOnce.CompareAndSwap(false, true) {
		return
	}

	go func() {
		for i := 0; i < parentMessageNotifyAttempts; i++ {
			time.Sleep(time.Second)
			if c.managerClosing.Load() {
				return
			}
			if err := c.tryStartParentMessageFlow(); err != nil {
				if strings.Contains(err.Error(), "not ready") {
					continue
				}
				log.Warnf("设备 %s 家长留言通知失败: %v", c.DeviceID, err)
				return
			}
			return
		}
	}()
}

func (c *ChatManager) tryStartParentMessageFlow() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	msg, err := c.parentMessageClient.GetPendingMessage(ctx, c.DeviceID)
	if err != nil {
		return err
	}
	if msg == nil || strings.TrimSpace(msg.TextContent) == "" {
		return nil
	}

	c.parentMessageMu.Lock()
	if c.parentMessageState != nil {
		c.parentMessageMu.Unlock()
		return nil
	}
	c.parentMessageState = &parentMessageFlowState{message: msg}
	c.parentMessageMu.Unlock()

	if err := c.parentMessageClient.UpdateStatus(ctx, msg.ID, "notified"); err != nil {
		log.Warnf("设备 %s 更新留言状态 notified 失败: %v", c.DeviceID, err)
	}

	c.clientState.OnAsrResultInterceptor = c.handleParentMessageASR
	if err := c.InjectMessage(parentMessageAskPrompt, true, true); err != nil {
		c.clearParentMessageFlow("skipped")
		return err
	}
	log.Infof("设备 %s 已开始家长留言确认流程, message_id=%d", c.DeviceID, msg.ID)
	return nil
}

func (c *ChatManager) handleParentMessageASR(text string) (bool, error) {
	c.parentMessageMu.Lock()
	state := c.parentMessageState
	c.parentMessageMu.Unlock()
	if state == nil || state.message == nil {
		return false, nil
	}

	intent := classifyParentMessageIntent(text)
	switch intent {
	case parentMessageIntentAffirmative:
		content := strings.TrimSpace(state.message.TextContent)
		c.clearParentMessageFlow("played")
		if content == "" {
			return true, nil
		}
		return true, c.InjectMessage(content, true, false)
	case parentMessageIntentNegative:
		c.clearParentMessageFlow("skipped")
		return true, nil
	default:
		if state.retryCount >= parentMessageMaxRetry {
			c.clearParentMessageFlow("skipped")
			return true, nil
		}
		state.retryCount++
		return true, c.InjectMessage(parentMessageRetryPrompt, true, true)
	}
}

func (c *ChatManager) clearParentMessageFlow(status string) {
	c.parentMessageMu.Lock()
	state := c.parentMessageState
	c.parentMessageState = nil
	c.parentMessageMu.Unlock()

	if c.clientState != nil {
		c.clientState.OnAsrResultInterceptor = nil
	}
	if state == nil || state.message == nil || c.parentMessageClient == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := c.parentMessageClient.UpdateStatus(ctx, state.message.ID, status); err != nil {
		log.Warnf("设备 %s 更新留言状态 %s 失败: %v", c.DeviceID, status, err)
	}
}
