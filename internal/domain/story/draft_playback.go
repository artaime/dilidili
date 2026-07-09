package story

import "strings"

// SpokenTextPrefix 返回听众实际已听过的正文前缀（整句边界，不含未播草稿）。
func SpokenTextPrefix(rec *StoryRecord) string {
	if rec == nil {
		return ""
	}
	full := strings.TrimSpace(rec.FullText)
	if full == "" {
		return ""
	}
	sentences := SplitSentences(full)
	if len(sentences) == 0 {
		return full
	}
	idx := rec.LastPosition.LastSentenceIndex
	if idx >= 0 && idx < len(sentences) {
		return strings.Join(sentences[:idx+1], "")
	}
	return spokenPrefixByOffset(full, sentences, rec.LastPosition.CharOffset)
}

func spokenPrefixByOffset(fullText string, sentences []string, charOffset int) string {
	if charOffset <= 0 {
		return ""
	}
	acc := 0
	lastComplete := -1
	for i, s := range sentences {
		acc += len([]rune(s))
		if acc <= charOffset {
			lastComplete = i
		}
	}
	if lastComplete >= 0 {
		return strings.Join(sentences[:lastComplete+1], "")
	}
	return ""
}

// FirstUnspokenSentenceIndex 返回第一条尚未播报的完整句索引（整句边界）。
func FirstUnspokenSentenceIndex(rec *StoryRecord) int {
	if rec == nil {
		return 0
	}
	full := strings.TrimSpace(rec.FullText)
	if full == "" {
		return 0
	}
	sentences := SplitSentences(full)
	if len(sentences) == 0 {
		return 0
	}
	if rec.LastPosition.LastSentenceIndex >= 0 {
		return rec.LastPosition.LastSentenceIndex + 1
	}
	if rec.LastPosition.CharOffset <= 0 {
		return 0
	}
	acc := 0
	for i, s := range sentences {
		if acc >= rec.LastPosition.CharOffset {
			return i
		}
		acc += len([]rune(s))
	}
	return len(sentences)
}

// DraftPlaybackSentences 返回待补播的草稿句（从第一条未播完整句到全文末）。
func DraftPlaybackSentences(rec *StoryRecord) []string {
	if rec == nil {
		return nil
	}
	full := strings.TrimSpace(rec.FullText)
	if full == "" {
		return nil
	}
	sentences := SplitSentences(full)
	start := FirstUnspokenSentenceIndex(rec)
	if start >= len(sentences) {
		return nil
	}
	return append([]string(nil), sentences[start:]...)
}

// MergeHeardStoryText 合并续讲前已听前缀与本会话新播报正文。
func MergeHeardStoryText(baselinePrefix, sessionSpoken string) string {
	baselinePrefix = strings.TrimSpace(baselinePrefix)
	sessionSpoken = strings.TrimSpace(sessionSpoken)
	switch {
	case baselinePrefix != "" && sessionSpoken != "":
		return baselinePrefix + sessionSpoken
	case sessionSpoken != "":
		return sessionSpoken
	default:
		return baselinePrefix
	}
}
