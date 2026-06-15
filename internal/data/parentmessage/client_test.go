package parentmessage

import (
	"encoding/json"
	"testing"
	"time"
)

func TestUnmarshalPendingMessagesResponseArray(t *testing.T) {
	body := []byte(`{"data":[{"id":1,"user_id":10,"device_id":2,"title":"语音留言","text_content":"","source_type":"voice","status":"pending","family_role":"妈妈","audio_url":"/api/internal/parent-messages/1/audio","created_at":"2026-06-12T15:30:00+08:00"}]}`)
	messages, err := parsePendingMessagesResponse(body)
	if err != nil {
		t.Fatalf("parse array: %v", err)
	}
	if len(messages) != 1 || messages[0].ID != 1 {
		t.Fatalf("unexpected data: %+v", messages)
	}
}

func TestUnmarshalPendingMessagesResponseLegacyObject(t *testing.T) {
	body := []byte(`{"data":{"id":2,"user_id":10,"device_id":2,"text_content":"你好","source_type":"text","status":"pending"}}`)
	messages, err := parsePendingMessagesResponse(body)
	if err != nil {
		t.Fatalf("parse legacy object: %v", err)
	}
	if len(messages) != 1 || messages[0].ID != 2 || messages[0].TextContent != "你好" {
		t.Fatalf("unexpected data: %+v", messages)
	}
}

func TestUnmarshalPendingMessagesResponseNull(t *testing.T) {
	body := []byte(`{"data":null}`)
	messages, err := parsePendingMessagesResponse(body)
	if err != nil {
		t.Fatalf("parse null: %v", err)
	}
	if len(messages) != 0 {
		t.Fatalf("expected empty, got %+v", messages)
	}
}

func TestUnmarshalPendingMessagesResponseHTMLFails(t *testing.T) {
	body := []byte(`<!DOCTYPE html><html><body>index</body></html>`)
	if _, err := parsePendingMessagesResponse(body); err == nil {
		t.Fatal("expected error for HTML body")
	}
}

func TestPendingMessageCreatedAtFlexible(t *testing.T) {
	type flex struct {
		CreatedAt time.Time `json:"created_at"`
	}
	for _, raw := range []string{
		`{"created_at":"2026-06-12T15:30:00+08:00"}`,
		`{"created_at":"2026-06-12T15:30:00Z"}`,
	} {
		var item flex
		if err := json.Unmarshal([]byte(raw), &item); err != nil {
			t.Fatalf("unmarshal %s: %v", raw, err)
		}
	}
}
