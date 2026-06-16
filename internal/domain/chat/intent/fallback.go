package intent

import (
	"encoding/json"
	"strings"
	"time"
)

var familyRoleKeywords = []struct {
	role string
	keys []string
}{
	{role: "爸爸", keys: []string{"爸爸", "父亲", "老爸"}},
	{role: "妈妈", keys: []string{"妈妈", "母亲", "老妈"}},
	{role: "爷爷", keys: []string{"爷爷"}},
	{role: "奶奶", keys: []string{"奶奶"}},
	{role: "外公", keys: []string{"外公", "姥爷"}},
	{role: "外婆", keys: []string{"外婆", "姥姥"}},
}

var dayLabelKeywords = []string{"今天", "昨天", "前天"}
var dayPeriodKeywords = []string{"早上", "上午", "中午", "下午", "傍晚", "晚上"}

func FallbackClassify(userText string) (RouterResponse, bool) {
	normalized := strings.ToLower(strings.TrimSpace(userText))
	if normalized == "" {
		return RouterResponse{}, false
	}

	if data, ok := inferMsgPlayData(userText); ok {
		raw, _ := json.Marshal(data)
		return RouterResponse{
			Intent:     IntentMsgPlay,
			Confidence: "0.75",
			Data:       raw,
		}, true
	}
	if isInquiryKeyword(normalized) {
		data, _ := json.Marshal(MsgInquiryData{Action: "list"})
		return RouterResponse{
			Intent:     IntentMsgInquiry,
			Confidence: "0.75",
			Data:       data,
		}, true
	}
	return RouterResponse{}, false
}

func inferMsgPlayData(text string) (MsgPlayData, bool) {
	normalized := strings.TrimSpace(text)
	if normalized == "" {
		return MsgPlayData{}, false
	}
	if isReplayKeyword(normalized) {
		return MsgPlayData{Action: MsgPlayActionReplayLast}, true
	}
	if !containsPlayIntent(normalized) {
		return MsgPlayData{}, false
	}
	if strings.Contains(normalized, "最近") || strings.Contains(normalized, "最新") {
		return MsgPlayData{Action: MsgPlayActionLatest}, true
	}

	role, hasRole := extractFamilyRole(normalized)
	dayLabel, hasDay := extractDayLabel(normalized)
	period, hasPeriod := extractDayPeriod(normalized)
	if hasRole || hasDay || hasPeriod {
		if !hasDay && hasPeriod {
			dayLabel = "今天"
		}
		start, end := BuildSelectTimeRange(dayLabel, period, time.Now())
		return MsgPlayData{
			Action:     MsgPlayActionSelect,
			FamilyRole: role,
			Start:      start,
			End:        end,
		}, true
	}
	if isPlayKeyword(normalized) {
		return MsgPlayData{Action: MsgPlayActionPending}, true
	}
	return MsgPlayData{}, false
}

func containsPlayIntent(text string) bool {
	return isPlayKeyword(text) || isReplayKeyword(text) ||
		strings.Contains(text, "最近") || strings.Contains(text, "最新")
}

func extractFamilyRole(text string) (string, bool) {
	for _, item := range familyRoleKeywords {
		for _, key := range item.keys {
			if strings.Contains(text, key) {
				return item.role, true
			}
		}
	}
	return "", false
}

func extractDayLabel(text string) (string, bool) {
	for _, label := range dayLabelKeywords {
		if strings.Contains(text, label) {
			return label, true
		}
	}
	return "", false
}

func extractDayPeriod(text string) (string, bool) {
	for _, period := range dayPeriodKeywords {
		if strings.Contains(text, period) {
			return period, true
		}
	}
	return "", false
}

func isInquiryKeyword(text string) bool {
	keywords := []string{"有留言", "有没有留言", "还有留言", "留言吗", "几条留言", "谁留言", "留言了"}
	for _, kw := range keywords {
		if strings.Contains(text, kw) {
			return true
		}
	}
	return false
}

func isPlayKeyword(text string) bool {
	keywords := []string{"播放留言", "继续播放", "播留言", "听留言", "放留言", "播放下一条", "播放吧", "播放", "播一下", "听一下"}
	for _, kw := range keywords {
		if strings.Contains(text, kw) {
			return true
		}
	}
	return false
}

func isReplayKeyword(text string) bool {
	keywords := []string{"再播", "重播", "播一遍", "再放一遍", "上一条", "刚才那条"}
	for _, kw := range keywords {
		if strings.Contains(text, kw) {
			return true
		}
	}
	return false
}
