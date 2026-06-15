package controllers

import (
	"testing"
	"time"

	"xiaozhi/manager/backend/models"
)

func TestSanitizeMessageText(t *testing.T) {
	input := "宝贝你好😊!!!@#$，今天开心吗？"
	got := sanitizeMessageText(input)
	want := "宝贝你好，今天开心吗？"
	if got != want {
		t.Fatalf("sanitizeMessageText() = %q, want %q", got, want)
	}
}

func TestAutoGenerateTitle(t *testing.T) {
	at := time.Date(2026, 6, 12, 15, 30, 0, 0, time.Local)
	got := autoGenerateTitle("voice", at)
	if got != "6月12日 15:30 语音留言" {
		t.Fatalf("autoGenerateTitle() = %q", got)
	}
}

func TestResolveMessageTitleUsesCustomTitle(t *testing.T) {
	at := time.Date(2026, 6, 12, 15, 30, 0, 0, time.Local)
	got := resolveMessageTitle("给宝贝的留言", "voice", at)
	if got != "给宝贝的留言" {
		t.Fatalf("resolveMessageTitle() = %q", got)
	}
}

func TestNormalizeFamilyRole(t *testing.T) {
	if normalizeFamilyRole("妈妈") != "妈妈" {
		t.Fatal("expected 妈妈")
	}
	if normalizeFamilyRole("invalid") != "其他" {
		t.Fatal("expected 其他 for invalid role")
	}
}

func TestEnrichParentMessageVoiceAudioURL(t *testing.T) {
	msg := models.ParentMessage{
		ID:         42,
		SourceType: "voice",
		AudioPath:  "/tmp/test.mp3",
		CreatedAt:  time.Now(),
	}
	item := enrichParentMessage(msg, models.Device{})
	if item["audio_url"] != "/api/mp/messages/42/audio" {
		t.Fatalf("audio_url = %v, want /api/mp/messages/42/audio", item["audio_url"])
	}
}
