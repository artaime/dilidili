package story

import "strings"

// SentenceBuffer 从流式文本中切分完整句子，便于 TTS 逐句入队。
type SentenceBuffer struct {
	pending strings.Builder
}

func (b *SentenceBuffer) Append(chunk string) []string {
	if chunk == "" {
		return nil
	}
	b.pending.WriteString(chunk)

	content := b.pending.String()
	runes := []rune(content)
	var sentences []string
	lastEnd := 0
	for i, r := range runes {
		if !isSentenceEnd(r) {
			continue
		}
		sentence := strings.TrimSpace(string(runes[lastEnd : i+1]))
		lastEnd = i + 1
		if sentence != "" {
			sentences = append(sentences, sentence)
		}
	}
	if lastEnd > 0 {
		rest := string(runes[lastEnd:])
		b.pending.Reset()
		b.pending.WriteString(rest)
	}
	return sentences
}

func (b *SentenceBuffer) Flush() string {
	out := strings.TrimSpace(b.pending.String())
	b.pending.Reset()
	return out
}
