package story

import (
	"strings"
	"unicode/utf8"
)

const StoryContextMaxRunes = 1500

// BuildStoryContextBrief 构建供 LLM 追问用的最近故事摘要（默认上限 StoryContextMaxRunes）。
func BuildStoryContextBrief(rec *StoryRecord) string {
	return BuildStoryFollowupBrief(rec, StoryContextMaxRunes)
}

// BuildStoryFollowupBrief 按 maxRunes 截断正文构建追问上下文；maxRunes<=0 时用 StoryContextMaxRunes。
func BuildStoryFollowupBrief(rec *StoryRecord, maxRunes int) string {
	if rec == nil {
		return ""
	}
	if maxRunes <= 0 {
		maxRunes = StoryContextMaxRunes
	}
	body := strings.TrimSpace(rec.FullText)
	if body == "" && len(rec.Segments) > 0 {
		body = strings.TrimSpace(strings.Join(rec.Segments, ""))
	}
	if body == "" {
		return ""
	}
	if utf8.RuneCountInString(body) > maxRunes {
		body = trimRunes(body, maxRunes) + "…"
	}
	var b strings.Builder
	b.WriteString("标题：" + strings.TrimSpace(rec.Title))
	if rec.ParamsSnapshot != nil {
		if t, ok := rec.ParamsSnapshot["story_title"].(string); ok && strings.TrimSpace(t) != "" {
			b.WriteString("\n故事名：" + strings.TrimSpace(t))
		}
		if g, ok := rec.ParamsSnapshot["genre"].(string); ok && strings.TrimSpace(g) != "" {
			b.WriteString("\n题材：" + strings.TrimSpace(g))
		}
		if t, ok := rec.ParamsSnapshot["theme"].(string); ok && strings.TrimSpace(t) != "" {
			b.WriteString("\n主题：" + strings.TrimSpace(t))
		}
	}
	b.WriteString("\n正文：\n")
	b.WriteString(body)
	return b.String()
}

func trimRunes(s string, max int) string {
	if max <= 0 {
		return ""
	}
	n := 0
	for i := range s {
		if n == max {
			return s[:i]
		}
		n++
	}
	return s
}
