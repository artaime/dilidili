package server

import (
	"testing"
	"time"

	"dili-esp32-server-golang/internal/app/server/chat"

	"github.com/spf13/viper"
)

func TestOnlineDeviceIDsFromManagers(t *testing.T) {
	ids := onlineDeviceIDsFromManagers(map[string]*chat.ChatManager{
		"dev-a": {},
		"":      {},
		"  ":    {},
		"dev-b": nil,
		"dev-c": {},
	})
	if len(ids) != 2 {
		t.Fatalf("ids=%v want 2 entries", ids)
	}
	got := map[string]bool{}
	for _, id := range ids {
		got[id] = true
	}
	if !got["dev-a"] || !got["dev-c"] {
		t.Fatalf("ids=%v missing expected devices", ids)
	}
}

func TestResolveDeviceOnlineHeartbeatInterval(t *testing.T) {
	viper.Reset()
	if got := resolveDeviceOnlineHeartbeatInterval(); got != defaultDeviceOnlineHeartbeatInterval {
		t.Fatalf("default interval=%v want %v", got, defaultDeviceOnlineHeartbeatInterval)
	}

	viper.Set("device_online.heartbeat_interval", 90*time.Second)
	if got := resolveDeviceOnlineHeartbeatInterval(); got != 90*time.Second {
		t.Fatalf("custom interval=%v want 90s", got)
	}
}
