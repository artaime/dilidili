package intent_test

import (
	"testing"

	"dili-esp32-server-golang/internal/domain/chat/intent"
)

func TestParseRouterResponse(t *testing.T) {
	resp, err := intent.ParseRouterResponse(`说明：{"intent":"msg_inquiry","confidence":"0.95","data":{"action":"list"}}`)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if resp.Intent != intent.IntentMsgInquiry {
		t.Fatalf("intent=%s", resp.Intent)
	}
	conf, err := intent.ParseConfidence(resp.Confidence)
	if err != nil || conf != 0.95 {
		t.Fatalf("confidence=%v err=%v", conf, err)
	}
}

func TestParseRouterResponseNeedsDialogue(t *testing.T) {
	resp, err := intent.ParseRouterResponse(`{"intent":"general","confidence":"0.9","needs_dialogue":true,"data":{}}`)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if !resp.NeedsDialogue {
		t.Fatal("expected needs_dialogue=true")
	}
	if resp.Intent != intent.IntentGeneral {
		t.Fatalf("intent=%s", resp.Intent)
	}
}
