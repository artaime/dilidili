package controllers

import (
	"fmt"
	"strings"
	"time"
	"unicode"

	"xiaozhi/manager/backend/models"
)

var allowedFamilyRoles = map[string]struct{}{
	"爸爸": {}, "妈妈": {}, "爷爷": {}, "奶奶": {}, "外公": {}, "外婆": {}, "其他": {},
}

func normalizeFamilyRole(role string) string {
	role = strings.TrimSpace(role)
	if role == "" {
		return "其他"
	}
	if _, ok := allowedFamilyRoles[role]; ok {
		return role
	}
	return "其他"
}

func isValidFamilyRole(role string) bool {
	_, ok := allowedFamilyRoles[strings.TrimSpace(role)]
	return ok
}

func sanitizeMessageText(text string) string {
	var b strings.Builder
	for _, r := range text {
		if r == '\n' || r == '\r' || r == '\t' {
			b.WriteRune(' ')
			continue
		}
		if unicode.IsControl(r) {
			continue
		}
		if isAllowedMessageRune(r) {
			b.WriteRune(r)
		}
	}
	return strings.TrimSpace(strings.Join(strings.Fields(b.String()), " "))
}

func isAllowedMessageRune(r rune) bool {
	if unicode.Is(unicode.Han, r) {
		return true
	}
	if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' {
		return true
	}
	switch r {
	case '，', '。', '！', '？', '、', '…', ' ', '·':
		return true
	}
	return false
}

func autoGenerateTitle(sourceType string, createdAt time.Time) string {
	label := "文字留言"
	if sourceType == "voice" {
		label = "语音留言"
	}
	return fmt.Sprintf("%d月%d日 %02d:%02d %s", int(createdAt.Month()), createdAt.Day(), createdAt.Hour(), createdAt.Minute(), label)
}

func resolveMessageTitle(title, sourceType string, createdAt time.Time) string {
	title = strings.TrimSpace(title)
	if title != "" {
		if len([]rune(title)) > 50 {
			title = string([]rune(title)[:50])
		}
		return title
	}
	return autoGenerateTitle(sourceType, createdAt)
}

func sourceTypeLabel(sourceType string) string {
	if sourceType == "voice" {
		return "语音"
	}
	return "文字"
}

func formatCreatedAtDisplay(t time.Time) string {
	return fmt.Sprintf("%d月%d日 %02d:%02d", int(t.Month()), t.Day(), t.Hour(), t.Minute())
}

func formatAudioDuration(sec int) string {
	if sec <= 0 {
		return ""
	}
	m := sec / 60
	s := sec % 60
	return fmt.Sprintf("%d:%02d", m, s)
}

func enrichParentMessage(msg models.ParentMessage, device models.Device) map[string]interface{} {
	statusText := map[string]string{
		"pending":  "待播放",
		"notified": "已通知",
		"played":   "已播放",
		"skipped":  "已跳过",
		"expired":  "已过期",
	}
	title := strings.TrimSpace(msg.Title)
	if title == "" {
		title = autoGenerateTitle(msg.SourceType, msg.CreatedAt)
	}
	item := map[string]interface{}{
		"id":                 msg.ID,
		"device_id":          msg.DeviceID,
		"device_name":        device.DeviceName,
		"device_nick":        device.NickName,
		"title":              title,
		"text_content":       msg.TextContent,
		"source_type":        msg.SourceType,
		"source_type_label":  sourceTypeLabel(msg.SourceType),
		"status":             msg.Status,
		"status_text":        statusText[msg.Status],
		"has_audio":          msg.AudioPath != "",
		"audio_duration_sec": msg.AudioDurationSec,
		"audio_duration":     formatAudioDuration(msg.AudioDurationSec),
		"created_at":         msg.CreatedAt,
		"created_at_display": formatCreatedAtDisplay(msg.CreatedAt),
		"played_at":          msg.PlayedAt,
	}
	if msg.SourceType == "voice" && strings.TrimSpace(msg.AudioPath) != "" {
		item["audio_url"] = fmt.Sprintf("/api/mp/messages/%d/audio", msg.ID)
	}
	return item
}
