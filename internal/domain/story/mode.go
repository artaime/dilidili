package story

import "strings"

const (
	StoryModeOriginal = "original"
	StoryModeClassic  = "classic"
	StoryModeMyth     = "myth"
	StoryModeFable    = "fable"
	StoryModeFairy    = "fairy_tale"
	StoryModeBedtime  = "bedtime"

	NarrationCanonical = "canonical"
	NarrationCreative  = "creative"
)

// NormalizeStoryParams 规范化 LLM 传入的故事参数，不做内置故事名匹配。
func NormalizeStoryParams(p *StoryParams) {
	if p == nil {
		return
	}
	p.RequestType = normalizeStoryType(p.RequestType)
	p.NarrationMode = normalizeNarrationMode(p.NarrationMode)
	if p.NarrationMode == "" {
		p.NarrationMode = defaultNarrationMode(*p)
	}
	if p.RequestType == "" {
		p.RequestType = StoryModeOriginal
	}
	if p.IsBedtime != nil && *p.IsBedtime && p.RequestType != StoryModeBedtime {
		// bedtime 标签保留，类型可并存
	}
}

func normalizeStoryType(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case StoryModeClassic, "classic_tale", "童话":
		return StoryModeClassic
	case StoryModeMyth, "神话", "legend":
		return StoryModeMyth
	case StoryModeFable, "寓言":
		return StoryModeFable
	case StoryModeFairy, "fairy":
		return StoryModeFairy
	case StoryModeBedtime, "睡前":
		return StoryModeBedtime
	case StoryModeOriginal, "原创", "interactive":
		return StoryModeOriginal
	case "":
		return ""
	default:
		return strings.ToLower(strings.TrimSpace(raw))
	}
}

func normalizeNarrationMode(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case NarrationCanonical, "正篇", "原著", "经典版", "canonical_retelling":
		return NarrationCanonical
	case NarrationCreative, "原创", "新编", "改编", "compose", "create":
		return NarrationCreative
	default:
		return strings.ToLower(strings.TrimSpace(raw))
	}
}

func defaultNarrationMode(p StoryParams) string {
	if p.UserSaidCasual || p.RequestType == StoryModeOriginal {
		return NarrationCreative
	}
	switch p.RequestType {
	case StoryModeClassic, StoryModeMyth, StoryModeFable, StoryModeFairy:
		return NarrationCanonical
	default:
		return NarrationCreative
	}
}

// ShouldTellCanonical 是否按广为流传的正篇/经典版讲述（非新编）。
func ShouldTellCanonical(p StoryParams) bool {
	NormalizeStoryParams(&p)
	if p.NarrationMode == NarrationCreative {
		return false
	}
	if p.NarrationMode == NarrationCanonical {
		return true
	}
	switch p.RequestType {
	case StoryModeClassic, StoryModeMyth, StoryModeFable, StoryModeFairy:
		return true
	default:
		return false
	}
}

// ThemeSpeakLabel 过渡语 TTS 口语化标题。
func ThemeSpeakLabel(theme, storyType string) string {
	theme = strings.TrimSpace(theme)
	if theme == "" {
		return "一个故事"
	}
	if strings.Contains(theme, "故事") || strings.Contains(theme, "童话") || strings.Contains(theme, "神话") {
		return theme
	}
	switch normalizeStoryType(storyType) {
	case StoryModeMyth:
		return theme + "的神话"
	case StoryModeFable:
		return theme + "的寓言"
	case StoryModeFairy, StoryModeClassic:
		return theme + "的故事"
	default:
		return theme + "的故事"
	}
}

// NormalizeThemeKey 用于主题匹配：去掉口语后缀，统一比较键。
func NormalizeThemeKey(theme string) string {
	theme = strings.TrimSpace(theme)
	for _, suffix := range []string{"的故事", "的神话", "的寓言", "的童话"} {
		theme = strings.TrimSuffix(theme, suffix)
	}
	return strings.TrimSpace(theme)
}

// ThemeMatchesRecord 判断用户主题是否与故事记录匹配。
func ThemeMatchesRecord(theme string, rec *StoryRecord) bool {
	if rec == nil {
		return false
	}
	key := NormalizeThemeKey(theme)
	if key == "" {
		return false
	}
	if rec.ParamsSnapshot != nil {
		if t, ok := rec.ParamsSnapshot["theme"].(string); ok {
			if NormalizeThemeKey(t) == key {
				return true
			}
		}
	}
	titleKey := NormalizeThemeKey(rec.Title)
	return titleKey == key || strings.Contains(titleKey, key) || strings.Contains(key, titleKey)
}

// TitleFromTheme 落库标题。
func TitleFromTheme(theme string) string {
	theme = strings.TrimSpace(theme)
	if theme == "" {
		return "儿童故事"
	}
	if strings.Contains(theme, "故事") {
		return theme
	}
	return theme + "的故事"
}
