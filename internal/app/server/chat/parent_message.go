package chat

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	types_conn "xiaozhi-esp32-server-golang/internal/app/server/types"
	parentmsg "xiaozhi-esp32-server-golang/internal/data/parentmessage"
	log "xiaozhi-esp32-server-golang/logger"
)

type parentMessageItem = parentmsg.PendingMessage

const (
	parentMessageMaxRetry       = 1
	parentMessageNotifyAttempts = 60
	parentMessageRetryDelay     = 3 * time.Second
)

// 家长留言通知触发来源（用于日志与重试窗口重置）。
const (
	ParentMessageNotifyFromManager   = "manager_created"
	ParentMessageNotifyFromHello     = "hello"
	ParentMessageNotifyFromTransport = "mqtt_transport_ready"
	ParentMessageNotifyFromUDP       = "udp_binding"
	ParentMessageNotifyFromRetry     = "scheduled_retry"
)

type parentMessageFlowState struct {
	messages   []parentMessageItem
	index      int
	retryCount int
	intentCh   chan parentMessageIntent
}

type parentMessageReadiness struct {
	Ready                bool
	BlockingReason       string
	ManagerClosing       bool
	HasClientState       bool
	HelloInited          bool
	TransportType        string
	HasChatSession       bool
	SpeakRequestEnabled  bool
	UDPBindingActive     bool
	SessionID            string
	UDPLastActiveMs      int64
	ParentMessageEnabled bool
}

func (r parentMessageReadiness) String() string {
	return fmt.Sprintf(
		"ready=%v reason=%q closing=%v client_state=%v hello=%v transport=%s session=%v session_id=%s speak_request=%v udp_binding=%v udp_last_active_ms=%d parent_client=%v",
		r.Ready, r.BlockingReason, r.ManagerClosing, r.HasClientState, r.HelloInited,
		r.TransportType, r.HasChatSession, r.SessionID, r.SpeakRequestEnabled,
		r.UDPBindingActive, r.UDPLastActiveMs, r.ParentMessageEnabled,
	)
}

func (c *ChatManager) collectParentMessageReadiness() parentMessageReadiness {
	report := parentMessageReadiness{
		ParentMessageEnabled: c != nil && c.parentMessageClient != nil,
	}
	if c == nil {
		report.BlockingReason = "chat_manager_nil"
		return report
	}
	report.ManagerClosing = c.managerClosing.Load()
	if report.ManagerClosing {
		report.BlockingReason = "manager_closing"
		return report
	}
	report.HasClientState = c.clientState != nil
	if !report.HasClientState {
		report.BlockingReason = "client_state_nil"
		return report
	}
	report.HelloInited = c.helloInited
	if !report.HelloInited {
		report.BlockingReason = "hello_not_inited"
		return report
	}
	if c.clientState != nil {
		report.SessionID = strings.TrimSpace(c.clientState.SessionID)
	}
	if c.serverTransport != nil {
		report.TransportType = c.serverTransport.GetTransportType()
	} else {
		report.TransportType = "unknown"
	}
	if c.serverTransport == nil || c.serverTransport.GetTransportType() != types_conn.TransportTypeMqttUdp {
		report.Ready = true
		report.BlockingReason = ""
		return report
	}
	report.HasChatSession = c.GetSession() != nil
	if !report.HasChatSession {
		report.BlockingReason = "chat_session_nil"
		return report
	}
	report.SpeakRequestEnabled = speakRequestEnabled()
	if !report.SpeakRequestEnabled {
		report.UDPBindingActive = c.serverTransport.HasActiveUDPBinding()
		if c.serverTransport != nil {
			report.UDPLastActiveMs = c.serverTransport.GetUDPLastActiveTs()
		}
		if !report.UDPBindingActive {
			report.BlockingReason = "udp_binding_pending"
			return report
		}
	}
	report.Ready = true
	report.BlockingReason = ""
	return report
}

func (c *ChatManager) isReadyForParentMessagePlayback() bool {
	return c.collectParentMessageReadiness().Ready
}

func (c *ChatManager) parentMessageNotReadyError() error {
	report := c.collectParentMessageReadiness()
	if report.Ready {
		return nil
	}
	return fmt.Errorf("parent message not ready: %s", report.BlockingReason)
}

func (c *ChatManager) SetParentMessageClient(client *parentmsg.Client) {
	if c == nil {
		return
	}
	c.parentMessageClient = client
}

func (c *ChatManager) shouldRestartParentMessageNotify(trigger string) bool {
	switch trigger {
	case ParentMessageNotifyFromHello, ParentMessageNotifyFromTransport, ParentMessageNotifyFromUDP:
		return true
	default:
		return false
	}
}

func (c *ChatManager) NotifyPendingParentMessages(trigger string) {
	if c == nil || c.parentMessageClient == nil {
		return
	}
	if trigger == "" {
		trigger = ParentMessageNotifyFromManager
	}
	if c.shouldRestartParentMessageNotify(trigger) {
		c.parentMessageNotifyGen.Add(1)
		c.parentMessageNotifyOnce.Store(false)
	}
	if !c.parentMessageNotifyOnce.CompareAndSwap(false, true) {
		log.Infof("设备 %s 家长留言通知已在进行，跳过重复触发 trigger=%s", c.DeviceID, trigger)
		return
	}

	gen := c.parentMessageNotifyGen.Load()
	log.Infof("设备 %s 开始家长留言就绪检测 trigger=%s gen=%d（协议要求：hello → session → UDP 绑定）",
		c.DeviceID, trigger, gen)

	go func() {
		defer func() {
			c.parentMessageMu.Lock()
			if c.parentMessageState == nil {
				c.parentMessageNotifyOnce.Store(false)
			}
			c.parentMessageMu.Unlock()
		}()

		var lastReport parentMessageReadiness
		var lastPendingCount = -1

		for i := 0; i < parentMessageNotifyAttempts; i++ {
			if c.parentMessageNotifyGen.Load() != gen {
				log.Infof("设备 %s 家长留言通知被新触发取代，结束本轮 trigger=%s gen=%d", c.DeviceID, trigger, gen)
				return
			}
			if i > 0 {
				time.Sleep(time.Second)
			}
			if c.managerClosing.Load() {
				log.Infof("设备 %s 家长留言通知中止：ChatManager 正在关闭", c.DeviceID)
				return
			}

			started, err := c.tryStartParentMessageFlow(i+1, &lastReport, &lastPendingCount)
			if err != nil {
				if isParentMessageRetryableError(err) {
					if (i+1)%5 == 0 || i == 0 {
						log.Infof("设备 %s 家长留言第 %d/%d 次检测未就绪: %v | %s",
							c.DeviceID, i+1, parentMessageNotifyAttempts, err, lastReport.String())
					} else {
						log.Debugf("设备 %s 家长留言第 %d/%d 次检测未就绪: %v | %s",
							c.DeviceID, i+1, parentMessageNotifyAttempts, err, lastReport.String())
					}
					continue
				}
				log.Warnf("设备 %s 家长留言通知失败 trigger=%s attempt=%d: %v | %s",
					c.DeviceID, trigger, i+1, err, lastReport.String())
				return
			}
			if started {
				log.Infof("设备 %s 家长留言流程已启动 trigger=%s attempt=%d | %s",
					c.DeviceID, trigger, i+1, lastReport.String())
				return
			}
			log.Infof("设备 %s 家长留言通知结束：无待播留言 trigger=%s attempt=%d pending=%d | %s",
				c.DeviceID, trigger, i+1, lastPendingCount, lastReport.String())
			return
		}
		log.Warnf("设备 %s 家长留言通知超时，未能在 %d 次重试内启动 trigger=%s | %s（对照 AI 玩具协议：需设备完成 hello 响应并建立 UDP 音频通道）",
			c.DeviceID, parentMessageNotifyAttempts, trigger, lastReport.String())
	}()
}

func (c *ChatManager) tryStartParentMessageFlow(attempt int, lastReport *parentMessageReadiness, lastPendingCount *int) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if c.parentMessageClient == nil {
		return false, fmt.Errorf("parent message client 未配置")
	}

	messages, err := c.parentMessageClient.ListPendingMessages(ctx, c.DeviceID)
	if err != nil {
		log.Warnf("设备 %s 家长留言第 %d 次拉取 pending 失败: %v", c.DeviceID, attempt, err)
		return false, err
	}
	*lastPendingCount = len(messages)
	if len(messages) == 0 {
		log.Debugf("设备 %s 家长留言第 %d 次检测：无 pending 留言", c.DeviceID, attempt)
		return false, nil
	}
	if !c.hasPlayableParentMessage(messages[0]) {
		first := messages[0]
		log.Warnf("设备 %s 待播放留言不可播放 message_id=%d source_type=%s audio_url=%q text_len=%d",
			c.DeviceID, first.ID, first.SourceType, first.AudioURL, len(strings.TrimSpace(first.TextContent)))
		return false, nil
	}

	report := c.collectParentMessageReadiness()
	*lastReport = report
	if !report.Ready {
		log.Debugf("设备 %s 家长留言第 %d 次检测：pending=%d 首条 id=%d 未就绪 %s",
			c.DeviceID, attempt, len(messages), messages[0].ID, report.String())
		return false, c.parentMessageNotReadyError()
	}

	c.parentMessageMu.Lock()
	if c.parentMessageState != nil {
		c.parentMessageMu.Unlock()
		log.Infof("设备 %s 家长留言流程已在进行，跳过重复启动", c.DeviceID)
		return true, nil
	}
	state := &parentMessageFlowState{
		messages: messages,
		intentCh: make(chan parentMessageIntent, 1),
	}
	c.parentMessageState = state
	c.parentMessageMu.Unlock()

	log.Infof("设备 %s 开始家长留言流程，待播 %d 条 | %s", c.DeviceID, len(messages), report.String())
	go c.runParentMessagePlayback(state)
	return true, nil
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
				if c.handleParentMessagePlaybackError(ctx, msg, state, err, "询问") {
					return
				}
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
				if c.handleParentMessagePlaybackError(ctx, msg, state, err, "过渡语") {
					return
				}
			}
			c.waitInjectedSpeechApprox(transition)
		}

		if err := c.playParentMessage(ctx, msg); err != nil {
			if c.handleParentMessagePlaybackError(ctx, msg, state, err, "播放") {
				return
			}
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
	log.Infof("设备 %s 家长留言询问语注入 message_id=%d | %s",
		c.DeviceID, msg.ID, c.collectParentMessageReadiness().String())
	if err := c.InjectMessage(prompt, true, true); err != nil {
		log.Warnf("设备 %s 家长留言询问语注入失败 message_id=%d: %v", c.DeviceID, msg.ID, err)
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

func isParentMessageRetryableError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "等待 speak_ready 超时") ||
		strings.Contains(msg, "等待 ChatSession 建立超时") ||
		strings.Contains(msg, "hello尚未初始化") ||
		strings.Contains(msg, "重新发送hello") ||
		strings.Contains(msg, "parent message not ready")
}

func (c *ChatManager) handleParentMessagePlaybackError(ctx context.Context, msg parentMessageItem, state *parentMessageFlowState, err error, stage string) bool {
	if isParentMessageRetryableError(err) {
		log.Warnf("设备 %s 家长留言%s失败，稍后重试: %v | %s", c.DeviceID, stage, err, c.collectParentMessageReadiness().String())
		c.updateParentMessageStatus(ctx, msg.ID, "pending")
		c.finishParentMessageSession(state.index)
		c.scheduleParentMessageRetry()
		return true
	}
	log.Warnf("设备 %s 家长留言%s失败: %v | %s", c.DeviceID, stage, err, c.collectParentMessageReadiness().String())
	c.updateParentMessageStatus(ctx, msg.ID, "skipped")
	c.finishParentMessageSession(state.index)
	return true
}

func (c *ChatManager) scheduleParentMessageRetry() {
	go func() {
		time.Sleep(parentMessageRetryDelay)
		if c.managerClosing.Load() {
			return
		}
		c.parentMessageNotifyOnce.Store(false)
		c.NotifyPendingParentMessages(ParentMessageNotifyFromRetry)
	}()
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
