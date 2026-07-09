package story

// HasStoryContent 故事是否有已落库正文。
func HasStoryContent(rec *StoryRecord) bool {
	return hasStoryContent(rec)
}

// IsGenerationComplete 故事正文是否已由 LLM 完整生成（可复播/按播放进度展示）。
func IsGenerationComplete(rec *StoryRecord) bool {
	if rec == nil {
		return false
	}
	if rec.GenerationComplete {
		return true
	}
	if rec.ParamsSnapshot != nil {
		if v, ok := rec.ParamsSnapshot["generation_complete"].(bool); ok {
			return v
		}
		if draft, ok := rec.ParamsSnapshot["draft"].(bool); ok && draft {
			return false
		}
	}
	// 兼容旧数据：有正文且非草稿占位，视为已完整生成。
	return hasStoryContent(rec)
}

func setGenerationComplete(rec *StoryRecord, complete bool) {
	if rec == nil {
		return
	}
	rec.GenerationComplete = complete
	if rec.ParamsSnapshot == nil {
		rec.ParamsSnapshot = map[string]any{}
	}
	rec.ParamsSnapshot["generation_complete"] = complete
	if complete {
		delete(rec.ParamsSnapshot, "draft")
	}
}

func paramsFromRecord(rec *StoryRecord) StoryParams {
	if rec == nil {
		return StoryParams{}
	}
	p := StoryParams{
		RequestType: rec.Mode,
		AgeBand:     rec.AgeBand,
	}
	if rec.ParamsSnapshot != nil {
		if v, ok := rec.ParamsSnapshot["theme"].(string); ok {
			p.Theme = v
		}
		if v, ok := rec.ParamsSnapshot["style"].(string); ok {
			p.Style = v
		}
	}
	return p
}

// ContinueFillerText 续写生成前的过渡语。
func ContinueFillerText(rec *StoryRecord) string {
	if rec == nil {
		return "上次讲到这儿，我们继续——"
	}
	if rec.LastPosition.LastSentence != "" {
		if prefix := BuildResumePrefixFromSentence(rec.LastPosition.LastSentence); prefix != "" {
			return prefix
		}
	}
	return "上次讲到这儿，我们继续——"
}

// SpokenPrefixForContinue 听众已听前缀（整句边界），供断点计算。
func SpokenPrefixForContinue(rec *StoryRecord) string {
	return SpokenTextPrefix(rec)
}
