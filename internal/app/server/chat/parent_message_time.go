package chat

import (
	"fmt"
	"time"
)

const parentMessageBatchGap = 3 * time.Hour

func formatChildFriendlyTime(t time.Time, now time.Time) string {
	if now.IsZero() {
		now = time.Now()
	}
	dayLabel := childFriendlyDayLabel(t, now)
	period := childFriendlyDayPeriod(t.Hour())
	minutePart := ""
	if t.Minute() > 0 {
		minutePart = fmt.Sprintf("%d分", t.Minute())
	}
	return fmt.Sprintf("%s%s%d点%s", dayLabel, period, t.Hour(), minutePart)
}

func childFriendlyDayLabel(t, now time.Time) string {
	tDate := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
	nowDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	switch nowDate.Sub(tDate) {
	case 0:
		return "今天"
	case 24 * time.Hour:
		return "昨天"
	case 48 * time.Hour:
		return "前天"
	default:
		return fmt.Sprintf("%d月%d日", int(t.Month()), t.Day())
	}
}

func childFriendlyDayPeriod(hour int) string {
	switch {
	case hour >= 5 && hour < 11:
		return "早上"
	case hour >= 11 && hour < 14:
		return "中午"
	case hour >= 14 && hour < 19:
		return "傍晚"
	default:
		return "晚上"
	}
}

func parentMessageNeedsAsk(index int, messages []parentMessageItem) bool {
	if index <= 0 {
		return true
	}
	prev := messages[index-1].CreatedAt
	curr := messages[index].CreatedAt
	return curr.Sub(prev) > parentMessageBatchGap
}

func buildTransitionPrompt(familyRole string, createdAt time.Time, now time.Time) string {
	role := normalizeFamilyRoleLabel(familyRole)
	timeDesc := formatChildFriendlyTime(createdAt, now)
	return fmt.Sprintf("接下来将播放%s%s的留言。", role, timeDesc)
}

func buildConfirmTransitionPrompt(familyRole string, createdAt time.Time, now time.Time) string {
	return "好的，" + buildTransitionPrompt(familyRole, createdAt, now)
}

func buildAskPromptFallback(familyRole string, createdAt time.Time, now time.Time) string {
	role := normalizeFamilyRoleLabel(familyRole)
	timeDesc := formatChildFriendlyTime(createdAt, now)
	return fmt.Sprintf("%s%s给你留言了，要播放吗？", role, timeDesc)
}

func normalizeFamilyRoleLabel(role string) string {
	role = trimSpace(role)
	switch role {
	case "", "其他":
		return "家长"
	default:
		return role
	}
}

func trimSpace(s string) string {
	start := 0
	for start < len(s) && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n') {
		start++
	}
	end := len(s)
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n') {
		end--
	}
	return s[start:end]
}
