package chat

import (
	"context"
	"time"

	"github.com/spf13/viper"

	log "dili-esp32-server-golang/logger"
)

const (
	ParentMessageNotifyFromPoll = "poll"
	defaultParentMessagePollSec = 30
)

func parentMessagePollInterval() time.Duration {
	if viper.IsSet("chat.parent_message.poll_interval_sec") {
		sec := viper.GetInt("chat.parent_message.poll_interval_sec")
		if sec > 0 {
			return time.Duration(sec) * time.Second
		}
	}
	return defaultParentMessagePollSec * time.Second
}

func parentMessagePollEnabled() bool {
	if !viper.IsSet("chat.parent_message.poll_enabled") {
		return true
	}
	return viper.GetBool("chat.parent_message.poll_enabled")
}

func (c *ChatManager) resetParentMessagePendingSnapshot() {
	c.parentMessageMu.Lock()
	c.parentMessagePendingSnapshot = make(map[uint]struct{})
	c.parentMessageMu.Unlock()
}

func (c *ChatManager) markParentMessagePendingSnapshot(messages []parentMessageItem) {
	if len(messages) == 0 {
		return
	}
	c.parentMessageMu.Lock()
	if c.parentMessagePendingSnapshot == nil {
		c.parentMessagePendingSnapshot = make(map[uint]struct{})
	}
	for _, msg := range messages {
		c.parentMessagePendingSnapshot[msg.ID] = struct{}{}
	}
	c.parentMessageMu.Unlock()
}

func (c *ChatManager) filterNewPendingMessages(messages []parentMessageItem) []parentMessageItem {
	c.parentMessageMu.Lock()
	defer c.parentMessageMu.Unlock()
	if c.parentMessagePendingSnapshot == nil {
		c.parentMessagePendingSnapshot = make(map[uint]struct{})
	}
	out := make([]parentMessageItem, 0)
	for _, msg := range messages {
		if _, seen := c.parentMessagePendingSnapshot[msg.ID]; !seen {
			out = append(out, msg)
		}
	}
	return out
}

func (c *ChatManager) filterPlayableParentMessages(messages []parentMessageItem) []parentMessageItem {
	out := make([]parentMessageItem, 0, len(messages))
	for _, msg := range messages {
		if c.hasPlayableParentMessage(msg) {
			out = append(out, msg)
		}
	}
	return out
}

// prepareParentMessageNotify 拉取 pending，按快照筛出尚未主动询问过的新留言。
func (c *ChatManager) prepareParentMessageNotify(ctx context.Context, trigger string) ([]parentMessageItem, []parentMessageItem, error) {
	if c.parentMessageClient == nil {
		return nil, nil, nil
	}
	_ = trigger

	allPending, err := c.parentMessageClient.ListPendingMessages(ctx, c.DeviceID)
	if err != nil {
		return nil, nil, err
	}
	newPending := c.filterNewPendingMessages(allPending)
	if len(newPending) == 0 {
		c.markParentMessagePendingSnapshot(allPending)
		return nil, allPending, nil
	}
	playable := c.filterPlayableParentMessages(newPending)
	return playable, allPending, nil
}

func (c *ChatManager) notifyParentMessagesOnce(trigger string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	newPending, allPending, err := c.prepareParentMessageNotify(ctx, trigger)
	if err != nil {
		return false, err
	}
	if len(newPending) == 0 {
		return false, nil
	}
	if !c.isReadyForParentMessagePlayback() {
		return false, c.parentMessageNotReadyError()
	}

	c.markParentMessagePendingSnapshot(allPending)
	log.Infof("设备 %s 检测到 %d 条新留言，主动询问是否播放 trigger=%s ids=%v",
		c.DeviceID, len(newPending), trigger, parentMessageIDs(newPending))
	return c.tryStartParentMessageFlowWithMessages(1, newPending, false)
}

func parentMessageIDs(messages []parentMessageItem) []uint {
	ids := make([]uint, 0, len(messages))
	for _, msg := range messages {
		ids = append(ids, msg.ID)
	}
	return ids
}

func (c *ChatManager) startParentMessagePoller() {
	if c == nil || !parentMessagePollEnabled() {
		return
	}
	if !c.parentMessagePollerStarted.CompareAndSwap(false, true) {
		return
	}
	go c.runParentMessagePoller()
}

func (c *ChatManager) runParentMessagePoller() {
	interval := parentMessagePollInterval()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	log.Infof("设备 %s 家长留言轮询已启动 interval=%s", c.DeviceID, interval)

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			if c.managerClosing.Load() {
				return
			}
			if !c.isReadyForParentMessagePlayback() {
				continue
			}
			c.parentMessageMu.Lock()
			inFlow := c.parentMessageState != nil
			c.parentMessageMu.Unlock()
			if inFlow {
				continue
			}
			if !c.parentMessageNotifyOnce.CompareAndSwap(false, true) {
				continue
			}
			started, err := c.notifyParentMessagesOnce(ParentMessageNotifyFromPoll)
			if !started {
				c.parentMessageNotifyOnce.Store(false)
			}
			if err != nil && !isParentMessageRetryableError(err) {
				log.Debugf("设备 %s 家长留言轮询检测: %v", c.DeviceID, err)
			}
		}
	}
}
