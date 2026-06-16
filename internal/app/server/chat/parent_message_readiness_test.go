package chat

import (
	"fmt"
	"testing"

	"github.com/spf13/viper"

	types_conn "dili-esp32-server-golang/internal/app/server/types"
	. "dili-esp32-server-golang/internal/data/client"
)

func TestCollectParentMessageReadinessHelloPending(t *testing.T) {
	manager := &ChatManager{
		DeviceID: "00:50:47:ba:b8:e8",
		clientState: &ClientState{
			DeviceID: "00:50:47:ba:b8:e8",
		},
	}
	report := manager.collectParentMessageReadiness()
	if report.Ready {
		t.Fatal("expected not ready before hello")
	}
	if report.BlockingReason != "hello_not_inited" {
		t.Fatalf("unexpected reason: %s", report.BlockingReason)
	}
}

func TestCollectParentMessageReadinessUDPPending(t *testing.T) {
	viper.Set("chat.speak_request_enabled", false)
	t.Cleanup(func() { viper.Set("chat.speak_request_enabled", nil) })

	manager, _ := newSpeakRequestTestManager(types_conn.TransportTypeMqttUdp)
	manager.helloInited = true
	manager.clientState.SessionID = "sess-1"
	manager.session = &ChatSession{}

	report := manager.collectParentMessageReadiness()
	if report.Ready {
		t.Fatal("expected not ready without udp binding")
	}
	if report.BlockingReason != "udp_binding_pending" {
		t.Fatalf("unexpected reason: %s", report.BlockingReason)
	}
	if report.TransportType != types_conn.TransportTypeMqttUdp {
		t.Fatalf("unexpected transport: %s", report.TransportType)
	}
}

func TestIsParentMessageRetryableErrorIncludesReadinessReasons(t *testing.T) {
	cases := []string{
		"parent message not ready: hello_not_inited",
		"parent message not ready: udp_binding_pending",
		"parent message not ready: chat_session_nil",
	}
	for _, msg := range cases {
		if !isParentMessageRetryableError(fmt.Errorf("%s", msg)) {
			t.Fatalf("expected retryable: %s", msg)
		}
	}
}

func TestShouldRestartParentMessageNotify(t *testing.T) {
	manager := &ChatManager{}
	if !manager.shouldRestartParentMessageNotify(ParentMessageNotifyFromHello) {
		t.Fatal("hello should restart notify window")
	}
	if manager.shouldRestartParentMessageNotify(ParentMessageNotifyFromManager) {
		t.Fatal("manager_created should not restart notify window")
	}
}
