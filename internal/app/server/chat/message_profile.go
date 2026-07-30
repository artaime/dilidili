package chat

import (
	"context"
	"time"

	"github.com/spf13/viper"

	"dili-esp32-server-golang/internal/domain/chat/devicestate"
	log "dili-esp32-server-golang/logger"
)

func playedHistoryLimit() int {
	if viper.IsSet("chat.device_message_profile.played_history_limit") {
		if n := viper.GetInt("chat.device_message_profile.played_history_limit"); n > 0 {
			return n
		}
	}
	return devicestate.DefaultPlayedHistoryLimit
}

func (c *ChatManager) ensureMessageProfileStore() devicestate.MessageProfileStore {
	if c == nil {
		return nil
	}
	if c.messageProfileStore == nil {
		c.messageProfileStore = devicestate.NewMemoryStore()
	}
	return c.messageProfileStore
}

func (c *ChatManager) syncDeviceMessageProfile(ctx context.Context) (*devicestate.DeviceMessageProfile, error) {
	store := c.ensureMessageProfileStore()
	if store == nil || c.parentMessageClient == nil {
		return devicestate.NewDeviceMessageProfile(c.DeviceID), nil
	}
	if ctx == nil {
		ctx = context.Background()
	}

	pending, err := c.parentMessageClient.ListPendingMessages(ctx, c.DeviceID)
	if err != nil {
		return nil, err
	}
	played, err := c.parentMessageClient.ListPlayedMessages(ctx, c.DeviceID, playedHistoryLimit())
	if err != nil {
		log.Warnf("设备 %s 拉取已播留言失败: %v", c.DeviceID, err)
		played = nil
	}

	profile := store.Upsert(c.DeviceID, func(p *devicestate.DeviceMessageProfile) *devicestate.DeviceMessageProfile {
		devicestate.ApplyPendingSync(p, len(pending))
		if len(played) > 0 {
			refs := make([]devicestate.PlayedMessageRef, 0, len(played))
			for _, msg := range played {
				ref := devicestate.PlayedMessageRef{
					MessageID:   msg.ID,
					FamilyRole:  msg.FamilyRole,
					Title:       msg.Title,
					SourceType:  msg.SourceType,
					TextContent: msg.TextContent,
					CreatedAt:   msg.CreatedAt,
				}
				if msg.PlayedAt != nil {
					ref.PlayedAt = *msg.PlayedAt
				}
				refs = append(refs, ref)
			}
			// ListPlayedMessages 为 played_at DESC；此处规范为升序，避免 LastPlayedRef 误取最早一条
			devicestate.ApplyPlayedHistorySync(p, refs, playedHistoryLimit())
		}
		return p
	})
	return profile, nil
}

func (c *ChatManager) recordPlayedMessageProfile(msg parentMessageItem) {
	store := c.ensureMessageProfileStore()
	if store == nil {
		return
	}
	now := time.Now()
	ref := devicestate.PlayedMessageRef{
		MessageID:   msg.ID,
		FamilyRole:  msg.FamilyRole,
		Title:       msg.Title,
		SourceType:  msg.SourceType,
		TextContent: msg.TextContent,
		CreatedAt:   msg.CreatedAt,
		PlayedAt:    now,
	}
	store.Upsert(c.DeviceID, func(p *devicestate.DeviceMessageProfile) *devicestate.DeviceMessageProfile {
		devicestate.AppendPlayedHistory(p, ref, playedHistoryLimit())
		if p.PendingCount > 0 {
			p.PendingCount--
		}
		p.HasNewMessages = p.PendingCount > 0
		p.AllCaughtUp = p.PendingCount == 0
		p.LastSyncedAt = now
		return p
	})
}

func (c *ChatManager) startParentMessagePlayback(messages []parentMessageItem, skipAsk bool) error {
	if len(messages) == 0 {
		return nil
	}
	c.parentMessageMu.Lock()
	if c.parentMessageState != nil {
		c.parentMessageMu.Unlock()
		return nil
	}
	state := &parentMessageFlowState{
		messages: messages,
		intentCh: make(chan parentMessageIntent, 1),
		skipAsk:  skipAsk,
	}
	c.parentMessageState = state
	c.parentMessageMu.Unlock()
	go c.runParentMessagePlayback(state)
	return nil
}
