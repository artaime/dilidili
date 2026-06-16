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
			PlayedAt:  now,
		}, 2)
	}
	if len(profile.PlayedHistory) != 2 || profile.LastPlayedMessageID != 3 {
		t.Fatalf("unexpected history: %+v", profile.PlayedHistory)
	}
}
