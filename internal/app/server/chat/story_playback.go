package chat

import (
	"strings"
	"sync"

	"dili-esp32-server-golang/internal/domain/story"
)

// StoryPlaybackTracker 跟踪当前会话的故事播报进度。
type StoryPlaybackTracker struct {
	mu sync.Mutex

	active            bool
	storyID           string
	deviceID          string
	segments          []string
	startSegment      int
	totalChars        int
	sentChars         int
	storySentChars    int
	lastSentence      string
	lastSentenceIndex int
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

func (s *ChatSession) ClearStoryPlayback() {
	if s == nil {
		return
	}
	tracker := s.storyPlaybackTracker()
	tracker.mu.Lock()
	tracker.active = false
	tracker.storyID = ""
	tracker.segments = nil
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
	defer tracker.mu.Unlock()
	if !tracker.active {
		return
	}
	tracker.storySentChars += len([]rune(sentence))
	tracker.lastSentence = sentence
	tracker.lastSentenceIndex++
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
