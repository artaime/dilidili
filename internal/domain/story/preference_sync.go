package story

import "context"

// PreferenceSync 预留：将播放偏好异步同步至 Memobase（首期 noop，后续实现）。
type PreferenceSync interface {
	SyncPlayPreference(ctx context.Context, deviceID string, record StoryRecord) error
}

type noopPreferenceSync struct{}

func (noopPreferenceSync) SyncPlayPreference(context.Context, string, StoryRecord) error {
	return nil
}

func DefaultPreferenceSync() PreferenceSync {
	return noopPreferenceSync{}
}
