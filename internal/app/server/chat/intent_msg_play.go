package chat

import (
	"context"
	"strings"
	"time"

	"xiaozhi-esp32-server-golang/internal/domain/chat/intent"
	log "xiaozhi-esp32-server-golang/logger"
)

func (c *ChatManager) HandleMsgPlay(ctx context.Context, data intent.MsgPlayData) error {
	if !c.collectParentMessageReadiness().Ready {
		return c.InjectMessage("需要先连接好再帮你播放留言哦。", true, true)
	}
	action := strings.TrimSpace(strings.ToLower(data.Action))
	if action == "" {
		action = intent.MsgPlayActionPending
	}
	ack := strings.TrimSpace(data.Reply)
	if ack == "" {
		switch action {
		case intent.MsgPlayActionReplayLast, intent.MsgPlayActionReplayID:
			ack = "好的，再给你播一遍。"
		case intent.MsgPlayActionLatest:
			ack = "好的，播放最近一条留言。"
		case intent.MsgPlayActionSelect:
			ack = "好的，帮你找这条留言。"
		default:
			ack = "好的，开始播放留言。"
		}
	}
	if err := c.injectSpeechSegment(ack, true, ttsTurnEndPolicyNone); err != nil {
		return err
	}
	c.waitInjectedSpeechSettled(ctx, ack)

	switch action {
	case intent.MsgPlayActionReplayLast:
		return c.replayLastParentMessage(ctx)
	case intent.MsgPlayActionReplayID:
		if data.MessageID == 0 {
			return c.InjectMessage("没有找到要重播的留言。", true, true)
		}
		return c.replayParentMessageByID(ctx, data.MessageID)
	case intent.MsgPlayActionLatest:
		return c.playLatestParentMessage(ctx)
	case intent.MsgPlayActionSelect:
		return c.playSelectedParentMessage(ctx, data)
	default:
		return c.playPendingParentMessages(ctx)
	}
}

func (c *ChatManager) playPendingParentMessages(ctx context.Context) error {
	messages, err := c.parentMessageClient.ListPendingMessages(ctx, c.DeviceID)
	if err != nil {
		log.Warnf("设备 %s 拉取待播留言失败: %v", c.DeviceID, err)
		return c.InjectMessage("播放留言失败了，稍后再试试吧。", true, true)
	}
	if len(messages) == 0 {
		_, _ = c.syncDeviceMessageProfile(ctx)
		return c.InjectMessage("现在没有待播放的留言哦。", true, true)
	}
	_, _ = c.syncDeviceMessageProfile(ctx)
	log.Infof("设备 %s msg_play pending count=%d", c.DeviceID, len(messages))
	return c.startParentMessagePlayback(messages, true)
}

func (c *ChatManager) replayLastParentMessage(ctx context.Context) error {
	store := c.ensureMessageProfileStore()
	var messageID uint
	if store != nil {
		if profile, ok := store.Get(c.DeviceID); ok && profile != nil {
			if ref, ok := profile.LastPlayedRef(); ok {
				messageID = ref.MessageID
			} else if profile.LastPlayedMessageID > 0 {
				messageID = profile.LastPlayedMessageID
			}
		}
	}
	if messageID == 0 {
		_, _ = c.syncDeviceMessageProfile(ctx)
		if store != nil {
			if profile, ok := store.Get(c.DeviceID); ok && profile != nil {
				if ref, ok := profile.LastPlayedRef(); ok {
					messageID = ref.MessageID
				}
			}
		}
	}
	if messageID == 0 {
		return c.InjectMessage("还没有播放过留言，暂时没法重播哦。", true, true)
	}
	return c.replayParentMessageByID(ctx, messageID)
}

func (c *ChatManager) replayParentMessageByID(ctx context.Context, messageID uint) error {
	msg, err := c.parentMessageClient.GetMessage(ctx, messageID)
	if err != nil || msg == nil {
		log.Warnf("设备 %s 获取留言详情失败 id=%d: %v", c.DeviceID, messageID, err)
		return c.InjectMessage("没有找到那条留言。", true, true)
	}
	item := parentMessageItem(*msg)
	if !c.hasPlayableParentMessage(item) {
		return c.InjectMessage("这条留言暂时无法播放。", true, true)
	}
	transition := buildTransitionPrompt(item.FamilyRole, item.CreatedAt, time.Now())
	if err := c.injectSpeechSegment(transition, true, ttsTurnEndPolicyNone); err != nil {
		return err
	}
	c.waitInjectedSpeechSettled(ctx, transition)
	if err := c.playParentMessage(ctx, item); err != nil {
		log.Warnf("设备 %s 重播留言失败 id=%d: %v", c.DeviceID, messageID, err)
		return c.InjectMessage("播放留言失败了，稍后再试试吧。", true, true)
	}
	store := c.ensureMessageProfileStore()
	if store != nil {
		c.recordPlayedMessageProfile(item)
	}
	return nil
}
