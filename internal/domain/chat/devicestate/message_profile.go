package devicestate

import (
	"sort"
	"time"
)

const DefaultPlayedHistoryLimit = 20

type PlayedMessageRef struct {
	MessageID   uint      `json:"message_id"`
	FamilyRole  string    `json:"family_role"`
	Title       string    `json:"title"`
	SourceType  string    `json:"source_type"`
	TextContent string    `json:"text_content,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	PlayedAt    time.Time `json:"played_at"`
}

type DeviceMessageProfile struct {
	DeviceID            string             `json:"device_id"`
	HasNewMessages      bool               `json:"has_new_messages"`
	PendingCount        int                `json:"pending_count"`
	AllCaughtUp         bool               `json:"all_caught_up"`
	LastSyncedAt        time.Time          `json:"last_synced_at"`
	LastPlayedMessageID uint               `json:"last_played_message_id"`
	PlayedHistory       []PlayedMessageRef `json:"played_history"`
}

func NewDeviceMessageProfile(deviceID string) *DeviceMessageProfile {
	return &DeviceMessageProfile{
		DeviceID:      deviceID,
		AllCaughtUp:   true,
		PlayedHistory: make([]PlayedMessageRef, 0),
	}
}

func ApplyPendingSync(profile *DeviceMessageProfile, pendingCount int) *DeviceMessageProfile {
	if profile == nil {
		return nil
	}
	profile.PendingCount = pendingCount
	profile.HasNewMessages = pendingCount > 0
	profile.AllCaughtUp = pendingCount == 0
	profile.LastSyncedAt = time.Now()
	return profile
}

func AppendPlayedHistory(profile *DeviceMessageProfile, ref PlayedMessageRef, limit int) *DeviceMessageProfile {
	if profile == nil {
		return nil
	}
	if limit <= 0 {
		limit = DefaultPlayedHistoryLimit
	}
	profile.LastPlayedMessageID = ref.MessageID
	profile.PlayedHistory = append(profile.PlayedHistory, ref)
	if len(profile.PlayedHistory) > limit {
		profile.PlayedHistory = profile.PlayedHistory[len(profile.PlayedHistory)-limit:]
	}
	return profile
}

// ApplyPlayedHistorySync 用已播列表覆盖档案。输入可为任意顺序（如 API 的 played_at DESC）；
// 内部规范为 PlayedAt 升序，与 AppendPlayedHistory 约定一致，保证 LastPlayedRef 指向最近一次播放。
func ApplyPlayedHistorySync(profile *DeviceMessageProfile, refs []PlayedMessageRef, limit int) *DeviceMessageProfile {
	if profile == nil {
		return nil
	}
	if limit <= 0 {
		limit = DefaultPlayedHistoryLimit
	}
	if len(refs) == 0 {
		profile.PlayedHistory = make([]PlayedMessageRef, 0)
		return profile
	}
	sorted := make([]PlayedMessageRef, len(refs))
	copy(sorted, refs)
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].PlayedAt.Equal(sorted[j].PlayedAt) {
			return sorted[i].MessageID < sorted[j].MessageID
		}
		return sorted[i].PlayedAt.Before(sorted[j].PlayedAt)
	})
	if len(sorted) > limit {
		sorted = sorted[len(sorted)-limit:]
	}
	profile.PlayedHistory = sorted
	profile.LastPlayedMessageID = sorted[len(sorted)-1].MessageID
	return profile
}

// LastPlayedRef 返回最近一次播放的留言（按 PlayedAt，同时间按 MessageID）。
func (p *DeviceMessageProfile) LastPlayedRef() (PlayedMessageRef, bool) {
	if p == nil || len(p.PlayedHistory) == 0 {
		return PlayedMessageRef{}, false
	}
	best := p.PlayedHistory[0]
	for i := 1; i < len(p.PlayedHistory); i++ {
		ref := p.PlayedHistory[i]
		if ref.PlayedAt.After(best.PlayedAt) || (ref.PlayedAt.Equal(best.PlayedAt) && ref.MessageID > best.MessageID) {
			best = ref
		}
	}
	return best, true
}
