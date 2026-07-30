package devicestate_test

import (
	"testing"
	"time"

	"dili-esp32-server-golang/internal/domain/chat/devicestate"
)

func TestApplyPendingSync(t *testing.T) {
	profile := devicestate.NewDeviceMessageProfile("00:11:22:33:44:55")
	devicestate.ApplyPendingSync(profile, 2)
	if !profile.HasNewMessages || profile.AllCaughtUp || profile.PendingCount != 2 {
		t.Fatalf("unexpected profile after pending sync: %+v", profile)
	}
	devicestate.ApplyPendingSync(profile, 0)
	if profile.HasNewMessages || !profile.AllCaughtUp || profile.PendingCount != 0 {
		t.Fatalf("unexpected profile after empty pending: %+v", profile)
	}
}

func TestAppendPlayedHistory(t *testing.T) {
	profile := devicestate.NewDeviceMessageProfile("dev")
	now := time.Now()
	for i := 1; i <= 3; i++ {
		devicestate.AppendPlayedHistory(profile, devicestate.PlayedMessageRef{
			MessageID: uint(i),
			PlayedAt:  now.Add(time.Duration(i) * time.Second),
		}, 2)
	}
	if len(profile.PlayedHistory) != 2 || profile.LastPlayedMessageID != 3 {
		t.Fatalf("unexpected history: %+v", profile.PlayedHistory)
	}
	ref, ok := profile.LastPlayedRef()
	if !ok || ref.MessageID != 3 {
		t.Fatalf("LastPlayedRef want id=3, got %+v ok=%v", ref, ok)
	}
}

func TestApplyPlayedHistorySyncNormalizesDescOrder(t *testing.T) {
	profile := devicestate.NewDeviceMessageProfile("dev")
	now := time.Now()
	// 模拟 API played_at DESC：最新在前
	devicestate.ApplyPlayedHistorySync(profile, []devicestate.PlayedMessageRef{
		{MessageID: 30, PlayedAt: now.Add(2 * time.Minute)},
		{MessageID: 20, PlayedAt: now.Add(1 * time.Minute)},
		{MessageID: 10, PlayedAt: now},
	}, 20)

	if profile.LastPlayedMessageID != 30 {
		t.Fatalf("LastPlayedMessageID want 30, got %d", profile.LastPlayedMessageID)
	}
	if len(profile.PlayedHistory) != 3 || profile.PlayedHistory[0].MessageID != 10 || profile.PlayedHistory[2].MessageID != 30 {
		t.Fatalf("history should be ascending by PlayedAt: %+v", profile.PlayedHistory)
	}
	ref, ok := profile.LastPlayedRef()
	if !ok || ref.MessageID != 30 {
		t.Fatalf("LastPlayedRef after DESC sync want id=30, got %+v ok=%v", ref, ok)
	}
}

func TestLastPlayedRefPicksNewestEvenIfUnsorted(t *testing.T) {
	profile := &devicestate.DeviceMessageProfile{
		DeviceID: "dev",
		PlayedHistory: []devicestate.PlayedMessageRef{
			{MessageID: 1, PlayedAt: time.Unix(100, 0)},
			{MessageID: 3, PlayedAt: time.Unix(300, 0)},
			{MessageID: 2, PlayedAt: time.Unix(200, 0)},
		},
	}
	ref, ok := profile.LastPlayedRef()
	if !ok || ref.MessageID != 3 {
		t.Fatalf("want id=3, got %+v ok=%v", ref, ok)
	}
}
