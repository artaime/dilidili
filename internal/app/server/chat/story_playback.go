package chat

import (
	"strings"
	"sync"
	"time"

	. "dili-esp32-server-golang/internal/data/client"
	"dili-esp32-server-golang/internal/domain/story"
	log "dili-esp32-server-golang/logger"
)

// StoryPlaybackTracker 跟踪当前会话的故事播报进度。
type StoryPlaybackTracker struct {
	mu sync.Mutex

	active            bool
	storyID           string
	title             string
	deviceID          string
	segments          []string
	startSegment      int
	totalChars        int
	sentChars         int
	storySentChars    int
	lastSentence      string
	lastSentenceIndex int
	lastProgressAt    time.Time
}

func (s *ChatSession) storyPlaybackTracker() *StoryPlaybackTracker {
	if s == nil {
		return nil
	}
	s.storyPlaybackMu.Lock()
	defer s.storyPlaybackMu.Unlock()
	if s.storyPlayback == nil {
		s.storyPlayback = &StoryPlaybackTracker{}
	}
	return s.storyPlayback
}

func (s *ChatSession) ActivateStoryPlayback(result *story.ToolResult) {
	if s == nil || result == nil {
		return
	}
	if result.StoryID == "" {
		return
	}
	if result.Status != story.StatusReady && result.Status != story.StatusReplay && result.Status != story.StatusResume {
		return
	}

	tracker := s.storyPlaybackTracker()
	tracker.mu.Lock()
	defer tracker.mu.Unlock()

	tracker.active = true
	tracker.storyID = result.StoryID
	tracker.title = strings.TrimSpace(result.Title)
	tracker.deviceID = s.clientState.DeviceID
	tracker.segments = result.Segments
	tracker.startSegment = result.StartSegment
	tracker.totalChars = len([]rune(result.TextToSpeak))
	if len(result.Segments) > result.StartSegment {
		total := 0
		for _, seg := range result.Segments[result.StartSegment:] {
			total += len([]rune(seg))
		}
		tracker.totalChars = total
	}
	tracker.sentChars = 0
	tracker.storySentChars = 0
	tracker.lastSentence = ""
	tracker.lastSentenceIndex = -1
	tracker.lastProgressAt = time.Time{}
	s.storyPlaybackActive.Store(true)
}

func (s *ChatSession) UpdateStoryPlaybackFromResult(result *story.ToolResult) {
	if s == nil || result == nil {
		return
	}
	tracker := s.storyPlaybackTracker()
	tracker.mu.Lock()
	defer tracker.mu.Unlock()
	if !tracker.active {
		return
	}
	if result.StoryID != "" {
		tracker.storyID = result.StoryID
	}
	if t := strings.TrimSpace(result.Title); t != "" {
		tracker.title = t
	}
	if len(result.Segments) > 0 {
		tracker.segments = result.Segments
		total := 0
		start := tracker.startSegment
		if start < 0 {
			start = 0
		}
		for _, seg := range result.Segments[start:] {
			total += len([]rune(seg))
		}
		tracker.totalChars = total
	}
}

func (s *ChatSession) IsStoryPlaybackActive() bool {
	if s == nil {
		return false
	}
	return s.storyPlaybackActive.Load()
}

// StoryPlaybackIdentity 返回当前播报故事的 ID 与标题（供对话短卡片落库）。
func (s *ChatSession) StoryPlaybackIdentity() (storyID, title string) {
	if s == nil || !s.IsStoryPlaybackActive() {
		return "", ""
	}
	tracker := s.storyPlaybackTracker()
	tracker.mu.Lock()
	defer tracker.mu.Unlock()
	if !tracker.active {
		return "", ""
	}
	return tracker.storyID, tracker.title
}

// isStoryPlaybackAudioGateActive 故事播报期间忽略扬声器回声触发的 ASR/VAD 打断，避免 TTS 与 LLM 流式生成被误终止。
func (s *ChatSession) isStoryPlaybackAudioGateActive() bool {
	return s != nil && s.IsStoryPlaybackActive()
}

// isAssistantOutputAudioGateActive 助手 LLM/TTS 输出期间启用回声门控，避免扬声器回声误触发打断。
func (s *ChatSession) isAssistantOutputAudioGateActive() bool {
	if s == nil || s.clientState == nil {
		return false
	}
	if s.isStoryPlaybackAudioGateActive() {
		return true
	}
	if s.clientState.GetTtsStart() {
		return true
	}
	switch s.clientState.GetStatus() {
	case ClientStatusLLMStart, ClientStatusTTSStart:
		return true
	default:
		return false
	}
}

// shouldIgnoreASRDuringStoryPlayback 故事播报中丢弃误触发的 ASR 文本（不进入 STT/LLM，也不触发 OnVoiceSilence）。
func (s *ChatSession) shouldIgnoreASRDuringStoryPlayback(text string) bool {
	if !s.isStoryPlaybackAudioGateActive() {
		return false
	}
	log.Infof("设备 %s 故事播报门控忽略 ASR 文本: %q", s.clientState.DeviceID, strings.TrimSpace(text))
	return true
}

// shouldIgnoreASRDuringAssistantOutput auto 模式下助手输出期间丢弃扬声器回声触发的 ASR，避免 TTS 播到一半被新轮次打断。
// realtime 模式保留用户主动插话打断能力。
func (s *ChatSession) shouldIgnoreASRDuringAssistantOutput(text string) bool {
	if s.shouldIgnoreASRDuringStoryPlayback(text) {
		return true
	}
	if s.clientState == nil || s.clientState.IsRealTime() {
		return false
	}
	if !s.isAssistantOutputAudioGateActive() {
		return false
	}
	log.Infof("设备 %s auto 模式助手输出门控忽略 ASR 回声: %q", s.clientState.DeviceID, strings.TrimSpace(text))
	return true
}

func (s *ChatSession) ClearStoryPlayback() {
	if s == nil {
		return
	}
	tracker := s.storyPlaybackTracker()
	tracker.mu.Lock()
	tracker.active = false
	tracker.storyID = ""
	tracker.title = ""
	tracker.segments = nil
	tracker.lastProgressAt = time.Time{}
	tracker.mu.Unlock()
	s.storyPlaybackActive.Store(false)
}

// StoryPlaybackSnapshot 返回当前故事播报进度快照（须在 ClearStoryPlayback 之前调用）。
func (s *ChatSession) StoryPlaybackSnapshot() (storySentChars int, lastSentence string, lastSentenceIndex int, segments []string, startSegment int, ok bool) {
	if s == nil || !s.IsStoryPlaybackActive() {
		return 0, "", -1, nil, 0, false
	}
	tracker := s.storyPlaybackTracker()
	tracker.mu.Lock()
	defer tracker.mu.Unlock()
	if !tracker.active {
		return 0, "", -1, nil, 0, false
	}
	segs := append([]string(nil), tracker.segments...)
	return tracker.storySentChars, tracker.lastSentence, tracker.lastSentenceIndex, segs, tracker.startSegment, true
}

func (s *ChatSession) OnStoryTextSent(delta string) {
	if s == nil || !s.IsStoryPlaybackActive() {
		return
	}
	tracker := s.storyPlaybackTracker()
	tracker.mu.Lock()
	defer tracker.mu.Unlock()
	if !tracker.active {
		return
	}
	tracker.sentChars += len([]rune(delta))
}

func (s *ChatSession) OnStorySentenceSent(sentence string) {
	if s == nil || !s.IsStoryPlaybackActive() {
		return
	}
	sentence = strings.TrimSpace(sentence)
	if sentence == "" {
		return
	}
	tracker := s.storyPlaybackTracker()
	tracker.mu.Lock()
	if !tracker.active {
		tracker.mu.Unlock()
		return
	}
	tracker.storySentChars += len([]rune(sentence))
	tracker.lastSentence = sentence
	tracker.lastSentenceIndex++
	storyID := tracker.storyID
	charOffset := tracker.storySentChars
	lastSent := tracker.lastSentence
	lastIdx := tracker.lastSentenceIndex
	segments := tracker.segments
	startSeg := tracker.startSegment
	shouldPersist := time.Since(tracker.lastProgressAt) >= storyProgressPersistMinInterval
	if shouldPersist {
		tracker.lastProgressAt = time.Now()
	}
	updater := s.storyProgressUpdater
	tracker.mu.Unlock()

	if !shouldPersist || updater == nil || storyID == "" {
		return
	}
	pos := story.PlayPosition{
		SegmentIndex:      startSeg,
		CharOffset:        charOffset,
		LastSentence:      lastSent,
		LastSentenceIndex: lastIdx,
	}
	if len(segments) > 0 && charOffset > 0 {
		pos.SegmentIndex = story.SegmentIndexForCharOffset(segments, charOffset)
		if pos.SegmentIndex < startSeg {
			pos.SegmentIndex = startSeg
		}
	}
	_ = updater.LocalMcpUpdateStoryProgress(s.ctx, storyID, pos, false, false)
}

func (s *ChatSession) OnStoryPlaybackFinished(completed bool) {
	if s == nil {
		return
	}
	tracker := s.storyPlaybackTracker()
	tracker.mu.Lock()
	if !tracker.active || tracker.storyID == "" {
		tracker.mu.Unlock()
		s.ClearStoryPlayback()
		return
	}
	storyID := tracker.storyID
	charOffset := tracker.storySentChars
	if charOffset == 0 {
		charOffset = tracker.sentChars
	}
	pos := story.PlayPosition{
		SegmentIndex:      tracker.startSegment,
		CharOffset:        charOffset,
		LastSentence:      tracker.lastSentence,
		LastSentenceIndex: tracker.lastSentenceIndex,
	}
	if len(tracker.segments) > 0 && charOffset > 0 {
		pos.SegmentIndex = story.SegmentIndexForCharOffset(tracker.segments, charOffset)
		if pos.SegmentIndex < tracker.startSegment {
			pos.SegmentIndex = tracker.startSegment
		}
	}
	updater := s.storyProgressUpdater
	hasSegments := len(tracker.segments) > 0
	tracker.mu.Unlock()

	if updater != nil && (completed || hasSegments || pos.CharOffset > 0) {
		_ = updater.LocalMcpUpdateStoryProgress(s.ctx, storyID, pos, !completed, completed)
	}
	s.ClearStoryPlayback()
}

func (s *ChatSession) OnStoryPlaybackInterrupted() {
	s.OnStoryPlaybackFinished(false)
}

// LogTTSSynthesizedDebug 在单句 TTS 合成并入队发送完成后打印调试日志（需 chat.debug_log_tts_only=true）。
func (s *ChatSession) LogTTSSynthesizedDebug(sentence string) {
	if !storyDebugLogTTSSynthesized() || s == nil || s.clientState == nil {
		return
	}
	sentence = strings.TrimSpace(sentence)
	if sentence == "" {
		return
	}
	deviceID := s.clientState.DeviceID
	if s.IsStoryPlaybackActive() {
		storyID := ""
		if tracker := s.storyPlaybackTracker(); tracker != nil {
			tracker.mu.Lock()
			storyID = tracker.storyID
			tracker.mu.Unlock()
		}
		log.Infof("[TTS-%s] story_id=%s: %s", deviceID, storyID, sentence)
		return
	}
	log.Infof("[TTS-%s]: %s", deviceID, sentence)
}

// LogStoryTTSSynthesized 故事播报路径的兼容别名。
func (s *ChatSession) LogStoryTTSSynthesized(sentence string) {
	s.LogTTSSynthesizedDebug(sentence)
}
