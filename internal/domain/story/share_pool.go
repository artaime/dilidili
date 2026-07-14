package story

import (
	"strings"
)

const (
	SharePoolNamed   = "named"
	SharePoolOpen    = "open"
	SharePoolBedtime = "bedtime"

	DefaultShareExcludeDays = 7
	DefaultSharePickTopK    = 5

		SnapshotKeyPoolKind     = "pool_kind"
		SnapshotKeyCanonicalKey = "canonical_key"
		SnapshotKeyAliases      = "aliases"
		SnapshotKeyThemeRaw     = "theme_raw"
	)

// 泛称主题：视为无点名剧情，进开放/睡前池。
var genericThemeKeys = map[string]struct{}{
	"": {}, "故事": {}, "一个故事": {}, "睡前故事": {}, "睡前": {},
	"童话": {}, "神话": {}, "寓言": {}, "随便": {}, "都可以": {},
}

// 口语填充片段：用于规范 key（轻量，不做向量）。
var canonicalFillerPhrases = []string{
	"三太子", "的传说", "传说", "故事", "神话", "寓言", "童话",
}

// ClassifyShareIntent 判断 generate 请求应查/入哪个共享池；空串表示不共享。
func ClassifyShareIntent(params StoryParams) string {
	p := params
	NormalizeStoryParams(&p)
	theme := NormalizeThemeKey(p.Theme)
	bedtime := p.IsBedtime != nil && *p.IsBedtime

	if ShouldTellCanonical(p) && theme != "" && !isGenericThemeKey(theme) {
		return SharePoolNamed
	}
	if bedtime && (theme == "" || isGenericThemeKey(theme) || isBedtimeGenericTheme(theme)) {
		return SharePoolBedtime
	}
	if p.UserSaidCasual || theme == "" || isGenericThemeKey(theme) {
		return SharePoolOpen
	}
	return ""
}

func isGenericThemeKey(theme string) bool {
	_, ok := genericThemeKeys[NormalizeThemeKey(theme)]
	return ok
}

func isBedtimeGenericTheme(theme string) bool {
	t := NormalizeThemeKey(theme)
	return t == "睡前故事" || t == "睡前" || strings.Contains(t, "睡前")
}

// NormalizeCanonicalKey 将口语主题收成规范比较键。
func NormalizeCanonicalKey(theme string) string {
	key := NormalizeThemeKey(theme)
	if key == "" {
		return ""
	}
	// 固定纠正常见异写（轻量词典，兜底 LLM 意图纠错）
	replacements := []struct{ old, neu string }{
		{"脑海", "闹海"},
		{"那吒", "哪吒"},
		{"三太子闹海", "闹海"},
		{"哪吒三太子闹海", "哪吒闹海"},
		{"三太子哪吒闹海", "哪吒闹海"},
		{"后裔", "后羿"},
		{"射太阳", "射日"},
		{"女哇", "女娲"},
		{"夸父追日", "夸父逐日"},
	}
	for _, r := range replacements {
		key = strings.ReplaceAll(key, r.old, r.neu)
	}
	for _, filler := range canonicalFillerPhrases {
		if filler == "" || key == filler {
			continue
		}
		key = strings.ReplaceAll(key, filler, "")
	}
	key = strings.TrimSpace(key)
	// 若剥太狠，回退 NormalizeThemeKey 结果
	if key == "" {
		return NormalizeThemeKey(theme)
	}
	// 哪吒+闹海 被剥成「哪吒闹海」类：确保关键实体保留
	orig := NormalizeThemeKey(theme)
	if strings.Contains(orig, "哪吒") && !strings.Contains(key, "哪吒") {
		key = "哪吒" + key
	}
	if strings.Contains(orig, "闹海") && !strings.Contains(key, "闹海") {
		key = key + "闹海"
	}
	return strings.TrimSpace(key)
}

// BuildShareLookupKeys 组装 named 池查询用的多键（规范名+口语+词典归一）。
func BuildShareLookupKeys(themes ...string) []string {
	seen := map[string]struct{}{}
	var out []string
	add := func(raw string) {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			return
		}
		for _, k := range []string{NormalizeThemeKey(raw), NormalizeCanonicalKey(raw)} {
			k = strings.TrimSpace(k)
			if k == "" || isGenericThemeKey(k) {
				continue
			}
			if _, ok := seen[k]; ok {
				continue
			}
			seen[k] = struct{}{}
			out = append(out, k)
		}
	}
	for _, t := range themes {
		add(t)
	}
	return out
}

// BuildAliasKeys 合并规范名与别名，去重规范化。
func BuildAliasKeys(canonical, title, theme string, extras []string) []string {
	seen := map[string]struct{}{}
	var out []string
	add := func(raw string) {
		for _, k := range []string{NormalizeThemeKey(raw), NormalizeCanonicalKey(raw)} {
			k = strings.TrimSpace(k)
			if k == "" || isGenericThemeKey(k) {
				continue
			}
			if _, ok := seen[k]; ok {
				continue
			}
			seen[k] = struct{}{}
			out = append(out, k)
		}
	}
	add(canonical)
	add(title)
	add(theme)
	for _, e := range extras {
		add(e)
	}
	return out
}

// ApplyShareEnrollment 将共享池元数据写入 ParamsSnapshot（完整生成后调用）。
func ApplyShareEnrollment(rec *StoryRecord, params StoryParams, meta StoryMeta) {
	if rec == nil {
		return
	}
	if rec.ParamsSnapshot == nil {
		rec.ParamsSnapshot = map[string]any{}
	}
	clearShare := func() {
		delete(rec.ParamsSnapshot, SnapshotKeyPoolKind)
		delete(rec.ParamsSnapshot, SnapshotKeyCanonicalKey)
		delete(rec.ParamsSnapshot, SnapshotKeyAliases)
	}
	// 未完整生成或无正文的故事不入共享池。
	if !IsGenerationComplete(rec) || !hasStoryContent(rec) {
		clearShare()
		return
	}
	pool := ClassifyShareIntent(params)
	if pool == "" {
		clearShare()
		return
	}
	rec.ParamsSnapshot[SnapshotKeyPoolKind] = pool

		theme := strings.TrimSpace(params.Theme)
		if meta.Theme != "" {
			theme = meta.Theme
		}
		themeRaw := strings.TrimSpace(params.ThemeRaw)
		if themeRaw == "" {
			themeRaw = strings.TrimSpace(fmtString(rec.ParamsSnapshot[SnapshotKeyThemeRaw]))
		}
		title := rec.Title
		if meta.Title != "" {
			title = meta.Title
		}
		canonical := strings.TrimSpace(meta.Canonical)
		if canonical == "" {
			canonical = NormalizeCanonicalKey(theme)
		} else {
			canonical = NormalizeCanonicalKey(canonical)
		}
		if pool == SharePoolNamed && canonical != "" {
			rec.ParamsSnapshot[SnapshotKeyCanonicalKey] = canonical
		}
		extras := append([]string{}, meta.Aliases...)
		if themeRaw != "" {
			extras = append(extras, themeRaw)
			rec.ParamsSnapshot[SnapshotKeyThemeRaw] = themeRaw
		}
		aliases := BuildAliasKeys(canonical, title, theme, extras)
		if len(aliases) > 0 {
			rec.ParamsSnapshot[SnapshotKeyAliases] = aliases
		}
		if _, ok := rec.ParamsSnapshot["theme"]; !ok || strings.TrimSpace(fmtString(rec.ParamsSnapshot["theme"])) == "" {
			if theme != "" {
				rec.ParamsSnapshot["theme"] = theme
			}
		}
	}

func fmtString(v any) string {
	s, _ := v.(string)
	return s
}

// ShareEnrollmentFromRecord 供 persist 客户端读取池字段。
func ShareEnrollmentFromRecord(rec *StoryRecord) (poolKind, canonicalKey string, aliases []string) {
	if rec == nil || rec.ParamsSnapshot == nil {
		return "", "", nil
	}
	poolKind, _ = rec.ParamsSnapshot[SnapshotKeyPoolKind].(string)
	canonicalKey, _ = rec.ParamsSnapshot[SnapshotKeyCanonicalKey].(string)
	switch a := rec.ParamsSnapshot[SnapshotKeyAliases].(type) {
	case []string:
		aliases = a
	case []any:
		for _, x := range a {
			if s, ok := x.(string); ok && strings.TrimSpace(s) != "" {
				aliases = append(aliases, s)
			}
		}
	}
	return strings.TrimSpace(poolKind), strings.TrimSpace(canonicalKey), aliases
}
