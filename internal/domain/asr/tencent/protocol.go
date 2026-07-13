package tencent

const (
	sliceTypeStart    = 0
	sliceTypePartial  = 1
	sliceTypeSentence = 2
)

type serverMessage struct {
	Code      int     `json:"code"`
	Message   string  `json:"message"`
	VoiceID   string  `json:"voice_id"`
	MessageID string  `json:"message_id"`
	Final     int     `json:"final"`
	Result    *result `json:"result"`
}

type result struct {
	SliceType    int    `json:"slice_type"`
	Index        int    `json:"index"`
	StartTime    int    `json:"start_time"`
	EndTime      int    `json:"end_time"`
	VoiceTextStr string `json:"voice_text_str"`
	WordSize     int    `json:"word_size"`
}

type endMessage struct {
	Type string `json:"type"`
}

func (m serverMessage) isFinal() bool {
	return m.Final == 1
}

func (m serverMessage) sentenceText() string {
	if m.Result == nil {
		return ""
	}
	return m.Result.VoiceTextStr
}

func (m serverMessage) isStableSentence() bool {
	if m.Result == nil {
		return false
	}
	return m.Result.SliceType == sliceTypeSentence
}

func (m serverMessage) isPartial() bool {
	if m.Result == nil {
		return false
	}
	return m.Result.SliceType == sliceTypePartial
}
