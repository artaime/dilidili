package story

import (
	"strings"
	"unicode"
)

// SplitSentences 将正文按句号等切分为完整句子。
func SplitSentences(fullText string) []string {
	fullText = strings.TrimSpace(fullText)
	if fullText == "" {
		return nil
	}

	runes := []rune(fullText)
	var sentences []string
	var buf strings.Builder
	for i, r := range runes {
		buf.WriteRune(r)
		if isSentenceEnd(r) && i < len(runes)-1 {
			s := strings.TrimSpace(buf.String())
			if s != "" {
				sentences = append(sentences, s)
			}
			buf.Reset()
		}
	}
	if tail := strings.TrimSpace(buf.String()); tail != "" {
		sentences = append(sentences, tail)
	}
	if len(sentences) == 0 {
		return []string{fullText}
	}
	return sentences
}

// SegmentText 将故事正文切分为 TTS 友好段落（按句号等，合并至每段约 1~3 句）。
func SegmentText(fullText string) []string {
	sentences := SplitSentences(fullText)
	if len(sentences) == 0 {
		return nil
	}

	const maxSentencesPerSegment = 3
	var segments []string
	var group strings.Builder
	count := 0
	for _, sent := range sentences {
		if count > 0 {
			group.WriteString("")
		}
		group.WriteString(sent)
		count++
		if count >= maxSentencesPerSegment {
			segments = append(segments, group.String())
			group.Reset()
			count = 0
		}
	}
	if group.Len() > 0 {
		segments = append(segments, group.String())
	}
	return segments
}

func isSentenceEnd(r rune) bool {
	switch r {
	case '。', '！', '？', '!', '?', '…':
		return true
	default:
		return false
	}
}

// ExtractTitle 从正文提取短标题。
func ExtractTitle(fullText string) string {
	fullText = strings.TrimSpace(fullText)
	if fullText == "" {
		return "未命名故事"
	}
	runes := []rune(fullText)
	end := 0
	for end < len(runes) && end < 24 {
		if isSentenceEnd(runes[end]) {
			break
		}
		end++
	}
	if end == 0 {
		end = minInt(16, len(runes))
	}
	title := strings.TrimSpace(string(runes[:end]))
	title = strings.TrimFunc(title, func(r rune) bool {
		return unicode.IsPunct(r) || unicode.IsSpace(r)
	})
	if title == "" {
		return "未命名故事"
	}
	if len([]rune(title)) > 20 {
		return string([]rune(title)[:20]) + "…"
	}
	return title
}

func TextFromSegmentIndex(segments []string, startIndex int) string {
	if startIndex < 0 || startIndex >= len(segments) {
		return ""
	}
	return strings.Join(segments[startIndex:], "")
}

func SegmentIndexForCharOffset(segments []string, charOffset int) int {
	if charOffset <= 0 || len(segments) == 0 {
		return 0
	}
	acc := 0
	for i, seg := range segments {
		acc += len([]rune(seg))
		if acc >= charOffset {
			return i
		}
	}
	return len(segments) - 1
}

// CharOffsetForSegment 返回 segmentIndex 之前的累计字符数。
func CharOffsetForSegment(segments []string, segmentIndex int) int {
	offset := 0
	for i := 0; i < segmentIndex && i < len(segments); i++ {
		offset += len([]rune(segments[i]))
	}
	return offset
}

// ComputePlayPosition 根据全文与已成功播报的正文，计算播放断点。
func ComputePlayPosition(fullText, spokenStoryText string) PlayPosition {
	spokenStoryText = strings.TrimSpace(spokenStoryText)
	spokenRunes := len([]rune(spokenStoryText))
	if spokenRunes <= 0 {
		return PlayPosition{}
	}
	segments := SegmentText(fullText)
	sentences := SplitSentences(fullText)

	lastIdx := -1
	lastSent := ""
	acc := 0
	for i, s := range sentences {
		acc += len([]rune(s))
		if acc <= spokenRunes {
			lastIdx = i
			lastSent = s
		}
	}
	if lastIdx < 0 && len(sentences) > 0 {
		lastIdx = 0
		lastSent = sentences[0]
	}

	return PlayPosition{
		SegmentIndex:      SegmentIndexForCharOffset(segments, spokenRunes),
		CharOffset:        spokenRunes,
		LastSentenceIndex: lastIdx,
		LastSentence:      lastSent,
	}
}

// SentenceStartOffset 返回 sentence 在 fullText 中的 rune 起始偏移。
func SentenceStartOffset(fullText, sentence string) int {
	sentence = strings.TrimSpace(sentence)
	if sentence == "" || fullText == "" {
		return -1
	}
	idx := strings.Index(fullText, sentence)
	if idx < 0 {
		return -1
	}
	return len([]rune(fullText[:idx]))
}

// TextFromResumePosition 根据断点返回续讲正文（不含过渡语）。
func TextFromResumePosition(rec *StoryRecord) string {
	if rec == nil {
		return ""
	}
	full := strings.TrimSpace(rec.FullText)
	if full == "" {
		return TextFromSegmentIndex(rec.Segments, ResumeStartSegment(rec))
	}
	return textFromCharOffsetForResume(full, rec.LastPosition, rec.LastPlayStatus)
}

func textFromCharOffsetForResume(fullText string, pos PlayPosition, status string) string {
	runes := []rune(fullText)
	if len(runes) == 0 {
		return ""
	}
	offset := pos.CharOffset
	if offset <= 0 {
		return fullText
	}
	if offset > len(runes) {
		return ""
	}
	// 打断续讲：回退到上一句完整句开头衔接；首句已播完则直接从 CharOffset 往后，避免重播全文。
	if status == PlayStatusInterrupted && pos.LastSentence != "" {
		if sentStart := SentenceStartOffset(fullText, pos.LastSentence); sentStart >= 0 && sentStart < offset {
			if pos.LastSentenceIndex > 0 {
				offset = sentStart
			}
		}
	}
	if offset >= len(runes) {
		return ""
	}
	return string(runes[offset:])
}

// BuildResumePrefixFromSentence 根据最后播完的整句生成续讲过渡语。
func BuildResumePrefixFromSentence(lastSentence string) string {
	excerpt := strings.TrimSpace(lastSentence)
	if excerpt == "" {
		return ""
	}
	runes := []rune(excerpt)
	if len(runes) > 36 {
		excerpt = string(runes[:36]) + "…"
	}
	return "上次讲到" + excerpt + "，我们接着往下讲——"
}

// ResumeSpeakPlan 计算续讲起始段、过渡语与正文。
func ResumeSpeakPlan(rec *StoryRecord) (startSeg int, prefix, body string) {
	if rec == nil {
		return 0, "上次讲到这儿，我们继续——", ""
	}
	body = TextFromResumePosition(rec)
	if body == "" {
		startSeg = ResumeStartSegment(rec)
		prefix = BuildResumePrefix(rec, startSeg)
		body = TextFromSegmentIndex(rec.Segments, startSeg)
		return startSeg, prefix, body
	}
	if rec.LastPosition.LastSentence != "" {
		prefix = BuildResumePrefixFromSentence(rec.LastPosition.LastSentence)
	}
	if prefix == "" {
		startSeg = ResumeStartSegment(rec)
		prefix = BuildResumePrefix(rec, startSeg)
	} else {
		offset := rec.LastPosition.CharOffset
		if rec.LastPlayStatus == PlayStatusInterrupted && rec.LastPosition.LastSentence != "" {
			if sentStart := SentenceStartOffset(rec.FullText, rec.LastPosition.LastSentence); sentStart >= 0 {
				if rec.LastPosition.LastSentenceIndex > 0 && sentStart < offset {
					offset = sentStart
				}
			}
		}
		segments := rec.Segments
		if len(segments) == 0 {
			segments = SegmentText(rec.FullText)
		}
		startSeg = SegmentIndexForCharOffset(segments, offset)
	}
	return startSeg, prefix, body
}

// ResumeStartSegment 计算续讲起始段：打断后回退一段以便衔接，已播完则从头。
func ResumeStartSegment(rec *StoryRecord) int {
	if rec == nil || len(rec.Segments) == 0 {
		return 0
	}
	if rec.LastPlayStatus == PlayStatusCompleted {
		return 0
	}
	if rec.LastPosition.CharOffset > 0 && strings.TrimSpace(rec.FullText) != "" {
		offset := rec.LastPosition.CharOffset
		if rec.LastPosition.LastSentence != "" {
			if sentStart := SentenceStartOffset(rec.FullText, rec.LastPosition.LastSentence); sentStart >= 0 {
				if rec.LastPosition.LastSentenceIndex > 0 && sentStart < offset {
					offset = sentStart
				}
			}
		}
		seg := SegmentIndexForCharOffset(rec.Segments, offset)
		if seg >= len(rec.Segments) {
			return len(rec.Segments) - 1
		}
		return seg
	}
	lastIdx := rec.LastPosition.SegmentIndex
	if lastIdx < 0 {
		lastIdx = 0
	}
	if lastIdx >= len(rec.Segments) {
		return 0
	}
	if rec.LastPlayStatus == PlayStatusCompleted {
		return 0
	}
	// 未播完却被标到最后一段，属于进度异常，从头续讲。
	if rec.LastPlayStatus == PlayStatusInterrupted && lastIdx >= len(rec.Segments)-1 && rec.CompleteCount == 0 {
		return 0
	}
	if rec.LastPlayStatus == PlayStatusInterrupted && lastIdx > 0 {
		return lastIdx - 1
	}
	return lastIdx
}

// BuildResumePrefix 生成续讲过渡语，简要提示上次讲到的内容。
func BuildResumePrefix(rec *StoryRecord, startSeg int) string {
	if rec == nil {
		return "上次讲到这儿，我们继续——"
	}
	if rec.LastPosition.LastSentence != "" {
		if prefix := BuildResumePrefixFromSentence(rec.LastPosition.LastSentence); prefix != "" {
			return prefix
		}
	}
	if len(rec.Segments) == 0 {
		return "上次讲到这儿，我们继续——"
	}
	lastIdx := rec.LastPosition.SegmentIndex
	if lastIdx < 0 || lastIdx >= len(rec.Segments) {
		return "上次讲到这儿，我们继续——"
	}
	excerpt := strings.TrimSpace(rec.Segments[lastIdx])
	if excerpt == "" {
		return "上次讲到这儿，我们继续——"
	}
	runes := []rune(excerpt)
	if len(runes) > 36 {
		excerpt = string(runes[:36]) + "…"
	}
	if startSeg <= 0 && lastIdx == 0 {
		return "故事才刚开始，我们接着说——"
	}
	return "上次讲到" + excerpt + "，我们接着往下讲——"
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
