package chat

import (
	"context"
	"strings"

	"dili-esp32-server-golang/internal/domain/chat/intent"
	log "dili-esp32-server-golang/logger"
)

func (c *ChatManager) HandleMsgInquiry(ctx context.Context, data intent.MsgInquiryData) error {
	if !c.collectParentMessageReadiness().Ready {
		return c.InjectMessage("需要先连接好再帮你查留言哦。", true, true)
	}
	ack := strings.TrimSpace(data.Reply)
	if ack == "" {
		ack = "好的，我帮你看一下留言。"
	}
	if err := c.InjectMessage(ack, true, true); err != nil {
		return err
	}
	c.waitInjectedSpeechSettled(ctx, ack)

	profile, err := c.syncDeviceMessageProfile(ctx)
	if err != nil {
		log.Warnf("设备 %s 同步留言档案失败: %v", c.DeviceID, err)
		return c.InjectMessage("查询留言失败了，稍后再试试吧。", true, true)
	}
	if profile == nil || !profile.HasNewMessages {
		return c.InjectMessage("现在没有新留言哦。", true, true)
	}

	pending, err := c.parentMessageClient.ListPendingMessages(ctx, c.DeviceID)
	if err != nil {
		log.Warnf("设备 %s 拉取 pending 留言失败: %v", c.DeviceID, err)
		return c.InjectMessage("查询留言失败了，稍后再试试吧。", true, true)
	}
	summary := c.buildParentMessageSummary(pending)
	log.Infof("设备 %s msg_inquiry pending=%d has_new=%v", c.DeviceID, len(pending), profile.HasNewMessages)
	return c.InjectMessage(summary, true, true)
}
