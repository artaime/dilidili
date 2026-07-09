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

const storyStreamDedupeWindow = 8 * time.Second

type storyStreamGuard struct {
	mu        sync.Mutex
	theme     string
	startedAt time.Time
}

func (c *ChatManager) tryBeginStoryStream(theme string) bool {
	if c == nil {
		return false
	}
	c.storyStreamGuard.mu.Lock()
	defer c.storyStreamGuard.mu.Unlock()
	key := story.NormalizeThemeKey(theme)
	now := time.Now()
	if key != "" && key == c.storyStreamGuard.theme && now.Sub(c.storyStreamGuard.startedAt) < storyStreamDedupeWindow {
		return false
	}
	c.storyStreamGuard.theme = key
	c.storyStreamGuard.startedAt = now
	return true
}

func (c *ChatManager) rememberRecentStory(rec *story.StoryRecord) {
	if c == nil || c.clientState == nil || rec == nil {
		return
	}
	brief := story.BuildStoryContextBrief(rec)
	if brief == "" {
		return
	}
	c.clientState.SetRecentStoryContext(brief)
	log.Infof("设备 %s 更新最近故事上下文 title=%q", c.DeviceID, rec.Title)
}

func (c *ChatManager) RememberStoryForFollowUp(ctx context.Context, session *ChatSession, storyID, spokenText string, completed bool) {
	heardSomething := strings.TrimSpace(spokenText) != ""
	if storyID != "" && (completed || heardSomething) {
		c.rememberRecentStoryByID(ctx, storyID)
	}
	if heardSomething {
		c.ensureStoryAssistantMessage(ctx, session, spokenText)
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

func (c *ChatManager) ensureStoryAssistantMessage(ctx context.Context, session *ChatSession, text string) {
	if c == nil || session == nil || session.llmManager == nil {
		return
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return
	}
	msgs := c.clientState.GetMessages(3)
	for i := len(msgs) - 1; i >= 0; i-- {
		msg := msgs[i]
		if msg == nil || msg.Role != schema.Assistant {
			continue
		}
		prev := strings.TrimSpace(msg.Content)
		if prev == text || strings.Contains(prev, text) || strings.Contains(text, prev) {
			return
		}
		break
	}
	if ctx == nil {
		ctx = c.clientState.Ctx
	}
	if err := session.llmManager.AddLlmMessage(ctx, schema.AssistantMessage(text, nil)); err != nil {
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

	fullRunes := utf8.RuneCountInString(rec.FullText)
	playedAll := fullRunes > 0 && pos.CharOffset >= fullRunes
	completed := playbackOK && llmComplete && playedAll
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
