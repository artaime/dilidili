package story

import "strings"

// ResolveStoryTitle 优先用题材生成标题，避免被正文首句覆盖。
func ResolveStoryTitle(theme, fullText, existingTitle string) string {
	theme = strings.TrimSpace(theme)
	if theme != "" {
		return TitleFromTheme(theme)
	}
	existingTitle = strings.TrimSpace(existingTitle)
	if existingTitle != "" && !looksLikeStoryOpening(existingTitle) {
		return existingTitle
	}
	if t := ExtractTitle(fullText); t != "" && t != "..." && !looksLikeStoryOpening(t) {
		return t
	}
	return "儿童故事"
}

func themeFromRecord(rec *StoryRecord, plan *GeneratePlan) string {
	if plan != nil {
		if t := strings.TrimSpace(plan.Params.Theme); t != "" {
			return t
		}
	}
	if rec != nil && rec.ParamsSnapshot != nil {
		if v, ok := rec.ParamsSnapshot["theme"].(string); ok {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func looksLikeStoryOpening(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}
	for _, prefix := range []string{
		"很久很久以前", "在很久很久以前", "从前", "有一天",
		"好的，我来", "让我来", "我来给你讲", "接下来",
	} {
		if strings.HasPrefix(s, prefix) {
			return true
		}
	}
	return len([]rune(s)) > 20
}
