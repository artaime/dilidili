package devicestate

import "sync"

type MessageProfileStore interface {
	Get(deviceID string) (*DeviceMessageProfile, bool)
	Upsert(deviceID string, update func(*DeviceMessageProfile) *DeviceMessageProfile) *DeviceMessageProfile
}

type MemoryStore struct {
	mu   sync.RWMutex
	data map[string]*DeviceMessageProfile
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{data: make(map[string]*DeviceMessageProfile)}
}

func (s *MemoryStore) Get(deviceID string) (*DeviceMessageProfile, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	profile, ok := s.data[deviceID]
	if !ok || profile == nil {
		return nil, false
	}
	copyProfile := *profile
	copyHistory := make([]PlayedMessageRef, len(profile.PlayedHistory))
	copy(copyHistory, profile.PlayedHistory)
	copyProfile.PlayedHistory = copyHistory
	return &copyProfile, true
}

func (s *MemoryStore) Upsert(deviceID string, update func(*DeviceMessageProfile) *DeviceMessageProfile) *DeviceMessageProfile {
	s.mu.Lock()
	defer s.mu.Unlock()
	profile, ok := s.data[deviceID]
	if !ok || profile == nil {
		profile = NewDeviceMessageProfile(deviceID)
		s.data[deviceID] = profile
	}
	if update != nil {
		profile = update(profile)
		if profile != nil {
			s.data[deviceID] = profile
		}
	}
	return profile
}
