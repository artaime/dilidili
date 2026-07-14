package story

import (
	"strings"
	"unicode/utf8"
)

// StoryGenres 故事题材（类型）枚举，供 LLM 与展示使用。
var StoryGenres = []string{
	"童话", "历史", "神话", "寓言", "冒险", "侦探", "科幻", "生活",
}

const metaMarkerPrefix = "[[meta:"

// StoryMeta LLM 在正文前输出的故事元信息（不播报）。
type StoryMeta struct {
	Title      string   `json:"title,omitempty"`
	Genre      string   `json:"genre,omitempty"`
	Theme      string   `json:"theme,omitempty"` // 具体故事主题/名称（可与 title 相同）
	Canonical  string   `json:"canonical,omitempty"`
	Aliases    []string `json:"aliases,omitempty"`
}

// MetaStreamFilter 从流式首行剥离 meta 行，避免送入 TTS。
type MetaStreamFilter struct {
	buf  strings.Builder
	done bool
	Meta *StoryMeta
}

func (f *MetaStreamFilter) Feed(chunk string) string {
	if f == nil || f.done {
		return chunk
	}
	if chunk == "" {
		return ""
	}
	f.buf.WriteString(chunk)
	s := f.buf.String()
	closeIdx := strings.Index(s, "]]")
	if closeIdx < 0 {
		if utf8.RuneCountInString(s) > 256 {
			f.done = true
			out := s
			f.buf.Reset()
			return out
		}
		return ""
	}
	head := strings.TrimSpace(s[:closeIdx+2])
	rest := s[closeIdx+2:]
	if strings.HasPrefix(head, metaMarkerPrefix) {
		if meta, ok := ParseMetaLine(head); ok {
			f.Meta = &meta
		}
	}
	f.done = true
	f.buf.Reset()
	rest = strings.TrimPrefix(rest, "\n")
	rest = strings.TrimPrefix(rest, "\r\n")
	return rest
}

// ParseMetaLine 解析 [[meta:title=xxx|genre=神话|theme=xxx]]
func ParseMetaLine(line string) (StoryMeta, bool) {
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, metaMarkerPrefix) || !strings.HasSuffix(line, "]]") {
		return StoryMeta{}, false
	}
	inner := strings.TrimSuffix(strings.TrimPrefix(line, metaMarkerPrefix), "]]")
	var meta StoryMeta
	for _, part := range strings.Split(inner, "|") {
		part = strings.TrimSpace(part)
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			continue
		}
		key := strings.TrimSpace(kv[0])
		val := strings.TrimSpace(kv[1])
		switch key {
		case "title", "名称":
			meta.Title = val
		case "genre", "题材", "type":
			meta.Genre = NormalizeGenre(val)
		case "theme", "主题":
			meta.Theme = val
		case "canonical", "规范名", "标准名":
			meta.Canonical = val
		case "aliases", "别名":
			for _, a := range strings.Split(val, ",") {
				a = strings.TrimSpace(a)
				if a != "" {
					meta.Aliases = append(meta.Aliases, a)
				}
			}
		}
	}
	if meta.Title == "" && meta.Theme != "" {
		meta.Title = meta.Theme
	}
	if meta.Theme == "" && meta.Title != "" {
		meta.Theme = meta.Title
	}
	return meta, meta.Title != "" || meta.Genre != "" || meta.Theme != "" || meta.Canonical != "" || len(meta.Aliases) > 0
}

// StripLeadingMeta 从完整文本剥离首行 meta，返回元信息与纯正文。
func StripLeadingMeta(fullText string) (StoryMeta, string) {
	fullText = strings.TrimSpace(fullText)
	if fullText == "" {
		return StoryMeta{}, fullText
	}
	firstLineEnd := strings.IndexAny(fullText, "\n\r")
	var firstLine, rest string
	if firstLineEnd < 0 {
		firstLine = fullText
		rest = ""
	} else {
		firstLine = fullText[:firstLineEnd]
		rest = strings.TrimLeft(fullText[firstLineEnd:], "\n\r")
	}
	firstLine = strings.TrimSpace(firstLine)
	if strings.HasPrefix(firstLine, metaMarkerPrefix) {
		if meta, ok := ParseMetaLine(firstLine); ok {
			return meta, rest
		}
	}
	return StoryMeta{}, fullText
}

// NormalizeGenre 规范题材名。
func NormalizeGenre(genre string) string {
	genre = strings.TrimSpace(genre)
	if genre == "" {
		return ""
	}
	for _, g := range StoryGenres {
		if genre == g || strings.Contains(genre, g) {
			return g
		}
	}
	return genre
}

// InferGenreFromParams 从请求类型推断默认题材。
func InferGenreFromParams(params StoryParams) string {
	p := params
	NormalizeStoryParams(&p)
	switch p.RequestType {
	case StoryModeMyth:
		return "神话"
	case StoryModeClassic, StoryModeFairy:
		return "童话"
	case StoryModeFable:
		return "寓言"
	case StoryModeBedtime:
		return "生活"
	case StoryModeOriginal:
		if strings.Contains(p.Style, "科幻") {
			return "科幻"
		}
		if strings.Contains(p.Style, "侦探") {
			return "侦探"
		}
		return "冒险"
	default:
		return ""
	}
}

func applyStoryMeta(rec *StoryRecord, meta StoryMeta, params StoryParams) {
	if rec == nil {
		return
	}
	if rec.ParamsSnapshot == nil {
		rec.ParamsSnapshot = map[string]any{}
	}
	title := strings.TrimSpace(meta.Title)
	if title == "" {
		title = strings.TrimSpace(meta.Theme)
	}
	genre := NormalizeGenre(meta.Genre)
	if genre == "" {
		genre = InferGenreFromParams(params)
	}
	theme := strings.TrimSpace(params.Theme)
	if theme == "" {
		theme = strings.TrimSpace(meta.Theme)
	}
	if title != "" && !looksLikeStoryOpening(title) {
		rec.ParamsSnapshot["story_title"] = title
		rec.Title = title
	} else if theme != "" {
		rec.Title = ResolveStoryTitle(theme, rec.FullText, rec.Title)
	} else if rec.Title == "" || looksLikeStoryOpening(rec.Title) {
		rec.Title = ResolveStoryTitle("", rec.FullText, rec.Title)
	}
	if genre != "" {
		rec.ParamsSnapshot["genre"] = genre
	}
	if theme != "" {
		rec.ParamsSnapshot["theme"] = theme
	}
}

func mergeStoryMeta(planMeta, parsed StoryMeta) StoryMeta {
	out := planMeta
	if strings.TrimSpace(parsed.Title) != "" {
		out.Title = parsed.Title
	}
	if strings.TrimSpace(parsed.Genre) != "" {
		out.Genre = parsed.Genre
	}
	if strings.TrimSpace(parsed.Theme) != "" {
		out.Theme = parsed.Theme
	}
	if strings.TrimSpace(parsed.Canonical) != "" {
		out.Canonical = parsed.Canonical
	}
	if len(parsed.Aliases) > 0 {
		out.Aliases = append(out.Aliases, parsed.Aliases...)
	}
	return out
}
