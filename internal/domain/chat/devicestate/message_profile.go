package devicestate

import "time"

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

func (p *DeviceMessageProfile) LastPlayedRef() (PlayedMessageRef, bool) {
	if p == nil || len(p.PlayedHistory) == 0 {
		return PlayedMessageRef{}, false
	}
	return p.PlayedHistory[len(p.PlayedHistory)-1], true
}
