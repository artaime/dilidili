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

func TestFallbackClassify(t *testing.T) {
	resp, ok := intent.FallbackClassify("我还有留言吗")
	if !ok || resp.Intent != intent.IntentMsgInquiry {
		t.Fatalf("unexpected inquiry fallback: %+v ok=%v", resp, ok)
	}
	resp, ok = intent.FallbackClassify("继续播放留言")
	if !ok || resp.Intent != intent.IntentMsgPlay {
		t.Fatalf("unexpected play fallback: %+v ok=%v", resp, ok)
	}
	data, err := intent.ParseData[intent.MsgPlayData](resp.Data)
	if err != nil || data.Action != intent.MsgPlayActionPending {
		t.Fatalf("play data=%+v err=%v", data, err)
	}

	resp, ok = intent.FallbackClassify("播放最近一条留言")
	if !ok || resp.Intent != intent.IntentMsgPlay {
		t.Fatalf("unexpected latest fallback: %+v", resp)
	}
	data, err = intent.ParseData[intent.MsgPlayData](resp.Data)
	if err != nil || data.Action != intent.MsgPlayActionLatest || data.FamilyRole != "" {
		t.Fatalf("latest data=%+v err=%v", data, err)
	}

	resp, ok = intent.FallbackClassify("播放妈妈最近的留言")
	if !ok || resp.Intent != intent.IntentMsgPlay {
		t.Fatalf("unexpected latest+role fallback: %+v", resp)
	}
	data, err = intent.ParseData[intent.MsgPlayData](resp.Data)
	if err != nil || data.Action != intent.MsgPlayActionLatest || data.FamilyRole != "妈妈" {
		t.Fatalf("latest+role data=%+v err=%v", data, err)
	}

	resp, ok = intent.FallbackClassify("播放妈妈昨天早上的留言")
	if !ok || resp.Intent != intent.IntentMsgPlay {
		t.Fatalf("unexpected select fallback: %+v", resp)
	}
	data, err = intent.ParseData[intent.MsgPlayData](resp.Data)
	if err != nil || data.Action != intent.MsgPlayActionSelect || data.FamilyRole != "妈妈" {
		t.Fatalf("select data=%+v err=%v", data, err)
	}
	if data.Start == "" || data.End == "" {
		t.Fatalf("select should include start/end: %+v", data)
	}

	resp, ok = intent.FallbackClassify("播放爸爸下午的留言")
	if !ok {
		t.Fatal("expected play select fallback")
	}
	data, err = intent.ParseData[intent.MsgPlayData](resp.Data)
	if err != nil || data.Action != intent.MsgPlayActionSelect || data.FamilyRole != "爸爸" {
		t.Fatalf("dad afternoon data=%+v err=%v", data, err)
	}
	if data.Start == "" || data.End == "" {
		t.Fatalf("select should include start/end: %+v", data)
	}

	resp, ok = intent.FallbackClassify("播放下午的留言")
	if !ok {
		t.Fatal("expected afternoon-only select fallback")
	}
	data, err = intent.ParseData[intent.MsgPlayData](resp.Data)
	if err != nil || data.Action != intent.MsgPlayActionSelect {
		t.Fatalf("afternoon-only data=%+v err=%v", data, err)
	}
	if data.FamilyRole != "" {
		t.Fatalf("afternoon-only should not set family_role: %+v", data)
	}
	if data.Start == "" || data.End == "" {
		t.Fatalf("afternoon-only should include start/end: %+v", data)
	}
}
