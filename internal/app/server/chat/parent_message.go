package chat

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	parentmsg "xiaozhi-esp32-server-golang/internal/data/parentmessage"
	log "xiaozhi-esp32-server-golang/logger"
)

type parentMessageItem = parentmsg.PendingMessage

const (
	parentMessageMaxRetry       = 1
	parentMessageNotifyAttempts = 10
)

type parentMessageFlowState struct {
	messages   []parentMessageItem
	index      int
	retryCount int
	intentCh   chan parentMessageIntent
}

func (c *ChatManager) SetParentMessageClient(client *parentmsg.Client) {
	if c == nil {
		return
	}
	c.parentMessageClient = client
}

func (c *ChatManager) NotifyPendingParentMessages() {
	if c == nil || c.parentMessageClient == nil {
		return
	}
	if !c.parentMessageNotifyOnce.CompareAndSwap(false, true) {
		return
	}

	go func() {
		for i := 0; i < parentMessageNotifyAttempts; i++ {
			time.Sleep(time.Second)
			if c.managerClosing.Load() {
				return
			}
			if err := c.tryStartParentMessageFlow(); err != nil {
				if strings.Contains(err.Error(), "not ready") {
					continue
				}
				log.Warnf("设备 %s 家长留言通知失败: %v", c.DeviceID, err)
				return
			}
			return
		}
	}()
}

func (c *ChatManager) tryStartParentMessageFlow() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	messages, err := c.parentMessageClient.ListPendingMessages(ctx, c.DeviceID)
	if err != nil {
		return err
	}
	if len(messages) == 0 {
		return nil
	}
	if !c.hasPlayableParentMessage(messages[0]) {
		return nil
	}

	c.parentMessageMu.Lock()
	if c.parentMessageState != nil {
		c.parentMessageMu.Unlock()
		return nil
	}
	state := &parentMessageFlowState{
		messages: messages,
		intentCh: make(chan parentMessageIntent, 1),
	}
	c.parentMessageState = state
	c.parentMessageMu.Unlock()

	go c.runParentMessagePlayback(state)
	return nil
}

func (c *ChatManager) hasPlayableParentMessage(msg parentMessageItem) bool {
	if msg.SourceType == "voice" {
		return strings.TrimSpace(msg.AudioURL) != ""
	}
	return strings.TrimSpace(msg.TextContent) != ""
}

func (c *ChatManager) runParentMessagePlayback(state *parentMessageFlowState) {
	defer c.clearParentMessageFlow("")

	ctx := context.Background()
	if c.ctx != nil {
		ctx = c.ctx
	}

	for state.index < len(state.messages) {
		msg := state.messages[state.index]
		if !c.hasPlayableParentMessage(msg) {
			c.updateParentMessageStatus(ctx, msg.ID, "skipped")
			state.index++
			continue
		}

		now := time.Now()
		if parentMessageNeedsAsk(state.index, state.messages) {
			if err := c.askAndWaitParentMessageIntent(ctx, state, msg); err != nil {
				log.Warnf("设备 %s 家长留言询问失败: %v", c.DeviceID, err)
				c.updateParentMessageStatus(ctx, msg.ID, "skipped")
				c.finishParentMessageSession(state.index)
				return
			}
			intent := <-state.intentCh
			if c.clientState != nil {
				c.clientState.OnAsrResultInterceptor = nil
			}
			if intent != parentMessageIntentAffirmative {
				c.updateParentMessageStatus(ctx, msg.ID, "skipped")
				c.finishParentMessageSession(state.index)
				return
			}
		} else {
			transition := buildTransitionPrompt(msg.FamilyRole, msg.CreatedAt, now)
			if err := c.InjectMessage(transition, true, false); err != nil {
				log.Warnf("设备 %s 家长留言过渡语失败: %v", c.DeviceID, err)
				c.finishParentMessageSession(state.index)
				return
			}
			c.waitInjectedSpeechApprox(transition)
		}

		if err := c.playParentMessage(ctx, msg); err != nil {
			log.Warnf("设备 %s 播放家长留言失败 message_id=%d: %v", c.DeviceID, msg.ID, err)
			c.updateParentMessageStatus(ctx, msg.ID, "skipped")
			c.finishParentMessageSession(state.index)
			return
		}
		c.updateParentMessageStatus(ctx, msg.ID, "played")
		state.index++
	}

	c.finishParentMessageSession(len(state.messages))
}

func (c *ChatManager) askAndWaitParentMessageIntent(ctx context.Context, state *parentMessageFlowState, msg parentMessageItem) error {
	c.updateParentMessageStatus(ctx, msg.ID, "notified")
	state.retryCount = 0
	c.clientState.OnAsrResultInterceptor = c.handleParentMessageASR

	prompt := c.generateParentMessageAskPrompt(ctx, msg.FamilyRole, msg.CreatedAt)
	if err := c.InjectMessage(prompt, true, true); err != nil {
		return err
	}
	return nil
}

func (c *ChatManager) handleParentMessageASR(text string) (bool, error) {
	c.parentMessageMu.Lock()
	state := c.parentMessageState
	c.parentMessageMu.Unlock()
	if state == nil || state.index >= len(state.messages) {
		return false, nil
	}

	ctx := context.Background()
	if c.ctx != nil {
		ctx = c.ctx
	}
	intent := c.classifyParentMessageIntentWithLLM(ctx, text)
	switch intent {
	case parentMessageIntentAffirmative:
		select {
		case state.intentCh <- parentMessageIntentAffirmative:
		default:
		}
		return true, nil
	case parentMessageIntentNegative:
		select {
		case state.intentCh <- parentMessageIntentNegative:
		default:
		}
		return true, nil
	default:
		if state.retryCount >= parentMessageMaxRetry {
			select {
			case state.intentCh <- parentMessageIntentNegative:
			default:
			}
			return true, nil
		}
		state.retryCount++
		retryPrompt := "没听清呢，如果想听留言就说「要」，不想听就说「不要」。"
		return true, c.InjectMessage(retryPrompt, true, true)
	}
}

func (c *ChatManager) playParentMessage(ctx context.Context, msg parentMessageItem) error {
	if msg.SourceType == "voice" {
		return c.playParentMessageVoice(ctx, msg)
	}
	content := strings.TrimSpace(msg.TextContent)
	if content == "" {
		return fmt.Errorf("文字留言内容为空")
	}
	return c.InjectMessage(content, true, false)
}

func (c *ChatManager) playParentMessageVoice(ctx context.Context, msg parentMessageItem) error {
	session, err := c.ensureSession()
	if err != nil {
		return err
	}
	if session.mediaPlayer == nil {
		return fmt.Errorf("media player 未初始化")
	}

	audioPath, cleanup, err := c.downloadParentMessageAudio(ctx, msg)
	if err != nil {
		return err
	}
	if cleanup != nil {
		defer cleanup()
	}

	title := strings.TrimSpace(msg.Title)
	if title == "" {
		title = "家长留言"
	}

	source := MediaSourceDescriptor{
		Title:      title,
		MIMEType:   "audio/mpeg",
		SourceType: MediaSourceTypeLocalFile,
		Local:      &LocalMediaSource{Path: audioPath},
	}
	handle, err := session.mediaPlayer.PlaySourceWithHandle(ctx, source)
	if err != nil {
		return err
	}
	return handle.Wait(ctx)
}

func (c *ChatManager) downloadParentMessageAudio(ctx context.Context, msg parentMessageItem) (string, func(), error) {
	if c.parentMessageClient == nil {
		return "", nil, fmt.Errorf("parent message client 未配置")
	}
	data, err := c.parentMessageClient.DownloadMessageAudio(ctx, msg.ID)
	if err != nil {
		return "", nil, err
	}
	tmpFile, err := os.CreateTemp("", fmt.Sprintf("parent-message-%d-*.mp3", msg.ID))
	if err != nil {
		return "", nil, err
	}
	path := tmpFile.Name()
	if _, err := tmpFile.Write(data); err != nil {
		tmpFile.Close()
		os.Remove(path)
		return "", nil, err
	}
	if err := tmpFile.Close(); err != nil {
		os.Remove(path)
		return "", nil, err
	}
	cleanup := func() {
		_ = os.Remove(path)
	}
	return filepath.Clean(path), cleanup, nil
}

func (c *ChatManager) updateParentMessageStatus(ctx context.Context, messageID uint, status string) {
	if c.parentMessageClient == nil {
		return
	}
	if err := c.parentMessageClient.UpdateStatus(ctx, messageID, status); err != nil {
		log.Warnf("设备 %s 更新留言状态 %s 失败: %v", c.DeviceID, status, err)
	}
}

func (c *ChatManager) waitInjectedSpeechApprox(text string) {
	delay := time.Duration(len([]rune(text))) * 120 * time.Millisecond
	if delay < 1500*time.Millisecond {
		delay = 1500 * time.Millisecond
	}
	if delay > 8*time.Second {
		delay = 8 * time.Second
	}
	time.Sleep(delay)
}

func (c *ChatManager) finishParentMessageSession(processedIndex int) {
	c.parentMessageMu.Lock()
	state := c.parentMessageState
	c.parentMessageState = nil
	c.parentMessageMu.Unlock()

	if c.clientState != nil {
		c.clientState.OnAsrResultInterceptor = nil
	}
	if state != nil && state.intentCh != nil {
		close(state.intentCh)
		state.intentCh = nil
	}
	log.Infof("设备 %s 家长留言流程结束, processed=%d", c.DeviceID, processedIndex)
}

func (c *ChatManager) clearParentMessageFlow(status string) {
	c.parentMessageMu.Lock()
	state := c.parentMessageState
	if state != nil {
		c.parentMessageState = nil
	}
	c.parentMessageMu.Unlock()

	if c.clientState != nil {
		c.clientState.OnAsrResultInterceptor = nil
	}
	if state == nil {
		return
	}
	if state.intentCh != nil {
		close(state.intentCh)
		state.intentCh = nil
	}
	if status != "" && state.index < len(state.messages) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		c.updateParentMessageStatus(ctx, state.messages[state.index].ID, status)
	}
}
