package chat

import (
	"context"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"dili-esp32-server-golang/internal/domain/story"
	log "dili-esp32-server-golang/logger"

	"github.com/cloudwego/eino/schema"
)

const (
	storyStreamDedupeWindow = 8 * time.Second
	// storyStreamEmptyThemeKey：「讲一个故事」等无主题请求的去重键。
	storyStreamEmptyThemeKey = "_empty_"
	// storyProgressPersistMinInterval 播报中写入进度的最小间隔。
	storyProgressPersistMinInterval = 2 * time.Second
)

type storyStreamGuard struct {
	mu        sync.Mutex
	theme     string
	startedAt time.Time
	active    bool // 是否有进行中的故事流（含播报）
}

func (c *ChatManager) tryBeginStoryStream(theme string) bool {
	if c == nil {
		return false
	}
	c.storyStreamGuard.mu.Lock()
	defer c.storyStreamGuard.mu.Unlock()
	key := story.NormalizeThemeKey(theme)
	if key == "" {
		key = storyStreamEmptyThemeKey
	}
	now := time.Now()
	// 任意进行中的故事流，或窗口内同主题（含空主题）重复请求，均拒绝再开流。
	if c.storyStreamGuard.active {
		return false
	}
	if key == c.storyStreamGuard.theme && now.Sub(c.storyStreamGuard.startedAt) < storyStreamDedupeWindow {
		return false
	}
	c.storyStreamGuard.theme = key
	c.storyStreamGuard.startedAt = now
	c.storyStreamGuard.active = true
	return true
}

func (c *ChatManager) endStoryStreamGuard() {
	if c == nil {
		return
	}
	c.storyStreamGuard.mu.Lock()
	c.storyStreamGuard.active = false
	c.storyStreamGuard.mu.Unlock()
}

func (c *ChatManager) isStoryStreamGuarded() bool {
	if c == nil {
		return false
	}
	c.storyStreamGuard.mu.Lock()
	defer c.storyStreamGuard.mu.Unlock()
	return c.storyStreamGuard.active
}

// shouldRejectStoryGenerateWhileActive 播报/流式生成中拒绝再开 generate，避免串播。
func (c *ChatManager) shouldRejectStoryGenerateWhileActive() bool {
	if c == nil {
		return false
	}
	if c.isStoryStreamGuarded() {
		return true
	}
	c.sessionMu.RLock()
	session := c.session
	c.sessionMu.RUnlock()
	return session != nil && session.IsStoryPlaybackActive()
}

func (c *ChatManager) rememberRecentStory(rec *story.StoryRecord) {
	if c == nil || c.clientState == nil || rec == nil {
		return
	}
	if strings.TrimSpace(rec.StoryID) == "" {
		return
	}
	theme := ""
	if rec.ParamsSnapshot != nil {
		if t, ok := rec.ParamsSnapshot["theme"].(string); ok {
			theme = strings.TrimSpace(t)
		}
	}
	c.clientState.SetRecentStoryPointer(rec.StoryID, rec.Title, theme)
	log.Infof("设备 %s 更新最近故事指针 story_id=%s title=%q", c.DeviceID, rec.StoryID, rec.Title)
}

	func (c *ChatManager) RememberStoryForFollowUp(ctx context.Context, session *ChatSession, storyID, spokenText string, completed bool) {
		heardSomething := strings.TrimSpace(spokenText) != ""
		var title string
		var genComplete bool
		if storyID != "" && (completed || heardSomething) {
			c.rememberRecentStoryByID(ctx, storyID)
			if rec, err := c.getStoryService().Store().Get(context.Background(), c.DeviceID, storyID); err == nil && rec != nil {
				title = rec.Title
				genComplete = story.IsGenerationComplete(rec)
			}
		}
		if heardSomething || completed {
			c.ensureStoryAssistantMessage(ctx, session, storyID, title, genComplete)
		}
	}

func (c *ChatManager) rememberRecentStoryByID(_ context.Context, storyID string) {
	if c == nil || storyID == "" {
		return
	}
	rec, err := c.getStoryService().Store().Get(context.Background(), c.DeviceID, storyID)
	if err != nil || rec == nil {
		return
	}
	c.rememberRecentStory(rec)
}

	func (c *ChatManager) ensureStoryAssistantMessage(ctx context.Context, session *ChatSession, storyID, title string, generationComplete bool) {
		if c == nil || session == nil || session.llmManager == nil {
			return
		}
		card := story.StoryCardContent(title)
		msgs := c.clientState.GetMessages(3)
		for i := len(msgs) - 1; i >= 0; i-- {
			msg := msgs[i]
			if msg == nil || msg.Role != schema.Assistant {
				continue
			}
			prev := strings.TrimSpace(msg.Content)
			if prev == card {
				return
			}
			if msg.Extra != nil {
				if id, _ := msg.Extra[story.ExtraKeyStoryID].(string); id != "" && id == storyID {
					return
				}
			}
			break
		}
		if ctx == nil {
			ctx = c.clientState.Ctx
		}
		msg := schema.AssistantMessage(card, nil)
		msg.Extra = story.StoryCardExtra(storyID, title, generationComplete)
		if err := session.llmManager.AddLlmMessage(ctx, msg); err != nil {
			log.Warnf("设备 %s 补写故事对话历史失败: %v", c.DeviceID, err)
		}
	}

func (c *ChatManager) syncStoryStreamProgress(svc *story.Service, deviceID, storyID, spokenStoryText string, storySentChars int, lastSentence string, lastSentenceIndex int, segments []string, llmComplete, playbackOK bool) {
	if svc == nil || storyID == "" {
		return
	}
	rec, err := svc.Store().Get(context.Background(), deviceID, storyID)
	if err != nil || rec == nil {
		return
	}
	if len(segments) == 0 {
		segments = rec.Segments
	}

	spokenStoryText = strings.TrimSpace(spokenStoryText)
	var pos story.PlayPosition
	switch {
	case rec.FullText != "" && spokenStoryText != "":
		pos = story.ComputePlayPosition(rec.FullText, spokenStoryText)
	case storySentChars > 0:
		pos = story.PlayPosition{
			CharOffset:        storySentChars,
			LastSentence:      lastSentence,
			LastSentenceIndex: lastSentenceIndex,
		}
		if len(segments) > 0 {
			pos.SegmentIndex = story.SegmentIndexForCharOffset(segments, storySentChars)
		}
	case rec.LastPosition.CharOffset > 0:
		pos = rec.LastPosition
	default:
		if len(segments) > 0 {
			pos.SegmentIndex = rec.LastPosition.SegmentIndex
		}
	}
	// 句缓冲长度偶发略短于全文时，以已播字数抬升断点。
	if storySentChars > pos.CharOffset {
		pos.CharOffset = storySentChars
		if len(segments) > 0 {
			pos.SegmentIndex = story.SegmentIndexForCharOffset(segments, storySentChars)
		}
	}

	fullRunes := utf8.RuneCountInString(rec.FullText)
	// TTS 正常排空且正文已完整生成 → 视为听完（句界导致的字数差不再误标打断）。
	completed := playbackOK && llmComplete
	if completed && fullRunes > 0 {
		pos.CharOffset = fullRunes
		if len(segments) > 0 {
			pos.SegmentIndex = len(segments) - 1
		}
	}
	interrupted := !completed
	_ = svc.UpdatePlaybackProgress(context.Background(), deviceID, storyID, pos, interrupted, completed)
}

func storySpokenText(filler, body string) string {
	filler = strings.TrimSpace(filler)
	body = strings.TrimSpace(body)
	switch {
	case filler != "" && body != "":
		return filler + body
	case body != "":
		return body
	default:
		return filler
	}
}
