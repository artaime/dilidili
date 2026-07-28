package chat

import (
	"context"
	"fmt"
	"strings"
	"time"

	. "dili-esp32-server-golang/internal/data/client"
	"dili-esp32-server-golang/internal/data/storypersist"
	"dili-esp32-server-golang/internal/domain/story"
	log "dili-esp32-server-golang/logger"
)

func (c *ChatManager) storyFollowupEnabled() bool {
	if c == nil {
		return false
	}
	return c.getStoryService().Config().FollowupEnabled
}

func (c *ChatManager) followupTTL() time.Duration {
	mins := c.getStoryService().Config().FollowupTTLMinutes
	if mins <= 0 {
		mins = 30
	}
	return time.Duration(mins) * time.Minute
}

func (c *ChatManager) clearFollowupClarify() {
	if c == nil {
		return
	}
	c.followupClarifyMu.Lock()
	defer c.followupClarifyMu.Unlock()
	c.followupClarifyRound = 0
	c.followupClarifyTheme = ""
	c.followupClarifyQ = ""
}

func (c *ChatManager) isFollowupClarifying() bool {
	if c == nil {
		return false
	}
	c.followupClarifyMu.Lock()
	defer c.followupClarifyMu.Unlock()
	return c.followupClarifyRound > 0
}

// handleStoryFollowup 处理故事情节追问。handled=false 表示应交主对话（无近期故事可答等）。
func (c *ChatManager) handleStoryFollowup(ctx context.Context, userText string, intent story.IntentResult) (handled bool, err error) {
	if c == nil {
		return true, fmt.Errorf("会话不可用")
	}
	userText = strings.TrimSpace(userText)
	theme, _ := story.ResolveIntentTheme(intent)

	rec := c.resolveFollowupStory(ctx, theme)
	if rec != nil && story.HasStoryContent(rec) {
		c.clearFollowupClarify()
		return true, c.answerStoryFollowupWithBody(ctx, userText, rec)
	}

	if theme != "" && story.IsClassicFollowupTarget(intent) {
		c.clearFollowupClarify()
		return true, c.answerClassicStoryFollowup(ctx, userText, theme)
	}

	if theme == "" {
		// 无名追问且无最近故事指针 → 放行主对话，避免「刚才好像还没讲故事」误伤
		c.clearFollowupClarify()
		log.Infof("设备 %s 故事追问无可用正文，交主对话 text=%q", c.DeviceID, userText)
		return false, nil
	}

	return true, c.askFollowupClarify(userText, theme)
}

func (c *ChatManager) handleFollowupClarifyTurn(ctx context.Context, userText string, intent story.IntentResult, classified bool) error {
	userText = strings.TrimSpace(userText)
	theme, _ := story.ResolveIntentTheme(intent)
	c.followupClarifyMu.Lock()
	if theme == "" {
		theme = c.followupClarifyTheme
	}
	origQ := c.followupClarifyQ
	c.followupClarifyMu.Unlock()
	if origQ == "" {
		origQ = userText
	}
	question := origQ
	if userText != "" && userText != origQ {
		question = origQ + "（补充：" + userText + "）"
	}

	rec := c.resolveFollowupStory(ctx, theme)
	if rec != nil && story.HasStoryContent(rec) {
		c.clearFollowupClarify()
		return c.answerStoryFollowupWithBody(ctx, question, rec)
	}
	if classified && theme != "" && story.IsClassicFollowupTarget(intent) {
		c.clearFollowupClarify()
		return c.answerClassicStoryFollowup(ctx, question, theme)
	}

	maxRounds := c.getStoryService().Config().FollowupClarifyMaxRounds
	if maxRounds <= 0 {
		maxRounds = 2
	}
	c.followupClarifyMu.Lock()
	round := c.followupClarifyRound
	if theme != "" {
		c.followupClarifyTheme = theme
	}
	c.followupClarifyMu.Unlock()
	if round >= maxRounds {
		c.clearFollowupClarify()
		return c.InjectMessage("这个故事我还不太清楚，等你想到更多细节再问我哦。", true, true)
	}
	return c.askFollowupClarify(origQ, theme)
}

func (c *ChatManager) askFollowupClarify(originalQ, theme string) error {
	maxRounds := c.getStoryService().Config().FollowupClarifyMaxRounds
	if maxRounds <= 0 {
		maxRounds = 2
	}
	c.followupClarifyMu.Lock()
	if c.followupClarifyRound >= maxRounds {
		c.followupClarifyMu.Unlock()
		c.clearFollowupClarify()
		return c.InjectMessage("这个故事我还不太清楚，等你想到更多细节再问我哦。", true, true)
	}
	c.followupClarifyRound++
	if theme != "" {
		c.followupClarifyTheme = theme
	}
	if c.followupClarifyQ == "" {
		c.followupClarifyQ = strings.TrimSpace(originalQ)
	}
	c.followupClarifyMu.Unlock()

	msg := "这个故事我不太熟，能再说说有哪些角色，或者大概讲了什么吗？"
	if theme != "" {
		msg = fmt.Sprintf("《%s》我这边还不熟，能再说说有哪些角色，或者大概讲了什么吗？", theme)
	}
	return c.InjectMessage(msg, true, true)
}

func (c *ChatManager) resolveFollowupStory(ctx context.Context, theme string) *story.StoryRecord {
	if c == nil || c.clientState == nil {
		return nil
	}
	store := c.getStoryService().Store()
	deviceID := c.DeviceID
	ptr, ok := c.clientState.RecentStoryPointer(c.followupTTL())
	themeKey := story.NormalizeThemeKey(theme)

	if themeKey != "" {
		if ok && recentPointerMatchesTheme(ptr, themeKey) {
			if rec := c.loadStoryBody(ctx, ptr.StoryID); rec != nil {
				return rec
			}
		}
		if rec, err := store.FindLatestByTheme(ctx, deviceID, themeKey, true); err == nil && rec != nil && story.HasStoryContent(rec) {
			return rec
		}
		return nil
	}

	if !ok {
		return nil
	}
	return c.loadStoryBody(ctx, ptr.StoryID)
}

func recentPointerMatchesTheme(ptr RecentStoryPointer, themeKey string) bool {
	if themeKey == "" {
		return false
	}
	if story.NormalizeThemeKey(ptr.Theme) == themeKey {
		return true
	}
	titleKey := story.NormalizeThemeKey(ptr.Title)
	return titleKey == themeKey || strings.Contains(titleKey, themeKey) || strings.Contains(themeKey, titleKey)
}

func (c *ChatManager) loadStoryBody(ctx context.Context, storyID string) *story.StoryRecord {
	storyID = strings.TrimSpace(storyID)
	if c == nil || storyID == "" {
		return nil
	}
	if rec, err := c.getStoryService().Store().Get(ctx, c.DeviceID, storyID); err == nil && rec != nil && story.HasStoryContent(rec) {
		return rec
	}
	if c.storyPersistClient == nil {
		c.storyPersistClient = storypersist.NewFromViper()
	}
	if c.storyPersistClient == nil || !c.storyPersistClient.Enabled() {
		return nil
	}
	rec, err := c.storyPersistClient.GetAsset(ctx, storyID)
	if err != nil || rec == nil || !story.HasStoryContent(rec) {
		return nil
	}
	log.Infof("设备 %s 追问从 MySQL 取故事正文 story_id=%s", c.DeviceID, storyID)
	return rec
}

func (c *ChatManager) answerStoryFollowupWithBody(ctx context.Context, userText string, rec *story.StoryRecord) error {
	cfg := c.getStoryService().Config()
	brief := story.BuildStoryFollowupBrief(rec, cfg.FollowupMaxRunes)
	if brief == "" {
		return c.InjectMessage("这个故事内容我暂时找不到，换个问题问问我吧。", true, true)
	}
	title := strings.TrimSpace(rec.Title)
	if title == "" {
		title = "刚才那个故事"
	}
	system := "你是儿童语音助手。用户在追问刚才听过的故事。仅根据提供的原文回答，两三句口语，勿复述全文，勿说不知道刚讲了什么。"
	user := fmt.Sprintf("故事《%s》原文：\n%s\n\n用户问：%s", title, brief, userText)
	reply, err := c.callLLMSyncText(ctx, system, user)
	if err != nil {
		log.Warnf("设备 %s 故事追问 LLM 失败: %v", c.DeviceID, err)
		return c.InjectMessage("这个问题我晚一点再回答你哦。", true, true)
	}
	reply = strings.TrimSpace(reply)
	if reply == "" {
		return c.InjectMessage("这个问题我还没想清楚，换个问法试试吧。", true, true)
	}
	return c.InjectMessage(reply, true, true)
}

func (c *ChatManager) answerClassicStoryFollowup(ctx context.Context, userText, theme string) error {
	system := fmt.Sprintf(
		"你是儿童语音助手。用户在问经典故事「%s」的情节。请按广为流传的通行说法，用两三句口语回答。" +
			"不要说「刚才给你讲过」或「我们刚听过」。不要整篇复述故事。",
		theme,
	)
	user := "用户问：" + userText
	reply, err := c.callLLMSyncText(ctx, system, user)
	if err != nil {
		log.Warnf("设备 %s 经典故事追问 LLM 失败: %v", c.DeviceID, err)
		return c.InjectMessage("这个问题我晚一点再回答你哦。", true, true)
	}
	reply = strings.TrimSpace(reply)
	if reply == "" {
		return c.InjectMessage("这个问题我还没想清楚，换个问法试试吧。", true, true)
	}
	return c.InjectMessage(reply, true, true)
}
