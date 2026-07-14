package chat

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"dili-esp32-server-golang/internal/domain/story"
	llm_common "dili-esp32-server-golang/internal/domain/llm/common"
	log "dili-esp32-server-golang/logger"
)

func (c *ChatManager) buildStoryToolRequest(params *CreateChildStoryParams) story.ToolRequest {
	return story.ToolRequest{
		Action:   params.Action,
		StoryRef: params.StoryRef,
		StoryParams: story.StoryParams{
			RequestType:    params.RequestType,
			NarrationMode:  params.NarrationMode,
			Theme:          params.Theme,
			ThemeRaw:       params.ThemeRaw,
			Style:          params.Style,
			AgeBand:        params.AgeBand,
			AgeYears:       params.AgeYears,
			IsBedtime:      params.IsBedtime,
			DurationHint:   params.DurationHint,
			Interests:      params.Interests,
			MemoryHints:    params.MemoryHints,
			UserSaidCasual: params.UserSaidCasual,
		},
		FromBeginning: params.FromBeginning,
		DeviceID:      c.DeviceID,
		AgentID:       c.clientState.AgentID,
		MemoryContext: c.clientState.MemoryContext,
		Now:           time.Now(),
	}
}

// storyProtectedGenerationContext 与 SessionCtx 解耦，避免用户插话取消故事 LLM。
func (c *ChatManager) storyProtectedGenerationContext(parent context.Context) context.Context {
	base := context.Background()
	if parent != nil && parent.Err() == nil {
		// 仅继承 parent 的值（如 trace），不继承取消；用 WithoutCancel。
		base = context.WithoutCancel(parent)
	}
	return base
}

func (c *ChatManager) streamGenerateChildStory(ctx context.Context, params *CreateChildStoryParams) error {
	if c == nil || params == nil {
		return nil
	}
	svc := c.getStoryService()
	req := c.buildStoryToolRequest(params)

	needResult, plan, err := svc.PlanGenerate(ctx, req)
	if err != nil {
		return err
	}
	if needResult != nil {
		return c.deliverChildStoryResult(ctx, needResult)
	}

	if !c.tryBeginStoryStream(plan.Params.Theme) {
		log.Infof("设备 %s 跳过重复故事流式启动 theme=%q", c.DeviceID, plan.Params.Theme)
		return nil
	}

	if err := svc.SaveDraftStory(ctx, req, plan); err != nil {
		log.Warnf("设备 %s 故事草稿落库失败: %v", c.DeviceID, err)
	}

	filler := svc.FillerText(plan.Params)
	if err := c.runStoryStream(ctx, req, plan, filler); err != nil {
		c.endStoryStreamGuard()
		return err
	}
	return nil
}

func (c *ChatManager) streamContinueChildStory(ctx context.Context, params *CreateChildStoryParams, rec *story.StoryRecord) error {
	if c == nil || params == nil || rec == nil {
		return nil
	}
	svc := c.getStoryService()
	req := c.buildStoryToolRequest(params)

	plan, err := svc.PlanContinueGenerate(ctx, rec)
	if err != nil {
		return err
	}

	theme := story.NormalizeThemeKey(plan.Params.Theme)
	if !c.tryBeginStoryStream(theme + ":continue:" + rec.StoryID) {
		log.Infof("设备 %s 跳过重复故事续写启动 story_id=%s", c.DeviceID, rec.StoryID)
		return nil
	}

	filler := story.ContinueFillerText(rec)
	if err := c.runStoryStream(ctx, req, plan, filler); err != nil {
		c.endStoryStreamGuard()
		return err
	}
	return nil
}

// runStoryStream 流式生成/续写：已播字数≥阈值时停播不停写；否则 TTS 打断同步取消 LLM。
func (c *ChatManager) runStoryStream(ctx context.Context, req story.ToolRequest, plan *story.GeneratePlan, filler string) error {
	c.cancelRetainedSessionCleanup("child_story_stream")
	session, err := c.ensureSession()
	if err != nil {
		return err
	}
	svc := c.getStoryService()
	threshold := story.ProtectContinueThreshold(svc.Config())

	session.ActivateStoryPlayback(&story.ToolResult{
		Status:       story.StatusReady,
		StoryID:      plan.StoryID,
		Title:        story.TitleFromTheme(plan.Params.Theme),
		TextToSpeak:  filler,
		StartSegment: 0,
	})

	responseChan := make(chan llm_common.LLMResponseStruct, 16)
	sessionCtx := c.clientState.SessionCtx.Get(c.clientState.Ctx)
	ttsCtx := c.clientState.AfterAsrSessionCtx.Get(sessionCtx)
	genCtx, genCancel := context.WithTimeout(c.storyProtectedGenerationContext(ctx), 90*time.Second)

	var (
		fullBuilder        strings.Builder
		storySpokenBuilder strings.Builder
		sentenceBuf        story.SentenceBuffer
		metaFilter         story.MetaStreamFilter
		streamMu           sync.Mutex
		streamErr          error
		savedComplete      bool
		playbackStopped    bool // TTS 已停但保护续写中
	)

	heardRunes := func() int {
		heard := story.MergeHeardStoryText(plan.SpokenBaseline, storySpokenBuilder.String())
		return utf8.RuneCountInString(heard)
	}

	// TTS 打断：字数未达阈值则取消 LLM；达阈值则仅停播。
	go func() {
		select {
		case <-ttsCtx.Done():
			streamMu.Lock()
			heard := heardRunes()
			if story.ShouldCancelGenerationOnInterrupt(heard, threshold) {
				log.Infof("设备 %s 故事生成随打断取消 story_id=%s heard_runes=%d threshold=%d",
					c.DeviceID, plan.StoryID, heard, threshold)
				genCancel()
			} else {
				playbackStopped = true
				log.Infof("设备 %s 故事保护续写 story_id=%s heard_runes=%d threshold=%d",
					c.DeviceID, plan.StoryID, heard, threshold)
			}
			streamMu.Unlock()
		case <-genCtx.Done():
		}
	}()

	go func() {
		defer genCancel()
		defer close(responseChan)

		firstChunk := true
		send := func(resp llm_common.LLMResponseStruct, isStorySentence bool) bool {
			streamMu.Lock()
			stopped := playbackStopped
			streamMu.Unlock()
			if stopped {
				return false
			}
			select {
			case responseChan <- resp:
				if t := strings.TrimSpace(resp.Text); t != "" {
					if isStorySentence {
						storySpokenBuilder.WriteString(t)
						session.OnStorySentenceSent(t)
					} else {
						session.OnStoryTextSent(t)
					}
				}
				return true
			case <-ttsCtx.Done():
				streamMu.Lock()
				if streamErr == nil {
					streamErr = ttsCtx.Err()
				}
				heard := heardRunes()
				if !story.ShouldCancelGenerationOnInterrupt(heard, threshold) {
					playbackStopped = true
					streamMu.Unlock()
					return false
				}
				streamMu.Unlock()
			case <-genCtx.Done():
				streamMu.Lock()
				if streamErr == nil {
					streamErr = genCtx.Err()
				}
				streamMu.Unlock()
			}
			return false
		}
		finish := func() {
			streamMu.Lock()
			stopped := playbackStopped
			streamMu.Unlock()
			if stopped {
				return
			}
			if firstChunk {
				send(llm_common.LLMResponseStruct{IsStart: true, IsEnd: true}, false)
				return
			}
			send(llm_common.LLMResponseStruct{IsEnd: true}, false)
		}

		saveCheckpoint := func(interrupted bool) {
			partial := fullBuilder.String()
			sessionSpoken := storySpokenBuilder.String()
			heard := story.MergeHeardStoryText(plan.SpokenBaseline, sessionSpoken)
			if !plan.IsContinuation {
				if partial == "" && heard == "" {
					return
				}
			} else if partial == "" && heard == plan.SpokenBaseline {
				return
			}
			reason := storyFailureReason(interrupted, ttsCtx.Err(), streamErr)
			log.Warnf("设备 %s 故事生成断点 story_id=%s partial_runes=%d heard_runes=%d interrupted=%t reason=%s err=%v",
				c.DeviceID, plan.StoryID, utf8.RuneCountInString(partial), utf8.RuneCountInString(heard), interrupted, reason, streamErr)
			if err := svc.SaveGenerationCheckpoint(context.Background(), req, plan, partial, heard, interrupted); err != nil {
				log.Warnf("设备 %s 故事生成断点落库失败: %v", c.DeviceID, err)
			}
		}

		if filler != "" {
			send(llm_common.LLMResponseStruct{Text: filler, IsStart: true}, false)
			firstChunk = false
		}

		// B 方案：续写前先按整句补播未听草稿。
		for _, sent := range plan.DraftPlaybackSentences {
			streamMu.Lock()
			stopped := playbackStopped
			streamMu.Unlock()
			if stopped || ttsCtx.Err() != nil {
				break
			}
			resp := llm_common.LLMResponseStruct{Text: sent}
			if firstChunk {
				resp.IsStart = true
				firstChunk = false
			}
			if !send(resp, true) {
				break
			}
		}
		streamMu.Lock()
		stoppedAfterDraft := playbackStopped
		streamMu.Unlock()
		if ttsCtx.Err() != nil && !stoppedAfterDraft {
			if story.ShouldCancelGenerationOnInterrupt(heardRunes(), threshold) {
				saveCheckpoint(true)
				finish()
				return
			}
			streamMu.Lock()
			playbackStopped = true
			streamMu.Unlock()
		}

		genErr := c.callLLMStreamForStory(genCtx, plan.SystemPrompt, plan.UserPrompt, func(chunk string) error {
			if chunk == "" {
				return nil
			}
			if genCtx.Err() != nil {
				return genCtx.Err()
			}
			clean := metaFilter.Feed(chunk)
			if metaFilter.Meta != nil {
				streamMu.Lock()
				plan.StoryMeta = *metaFilter.Meta
				streamMu.Unlock()
			}
			if clean == "" {
				return nil
			}
			fullBuilder.WriteString(clean)
			for _, sent := range sentenceBuf.Append(clean) {
				streamMu.Lock()
				stopped := playbackStopped
				streamMu.Unlock()
				if stopped {
					continue
				}
				if ttsCtx.Err() != nil {
					heard := heardRunes()
					if story.ShouldCancelGenerationOnInterrupt(heard, threshold) {
						return ttsCtx.Err()
					}
					streamMu.Lock()
					playbackStopped = true
					streamMu.Unlock()
					continue
				}
				resp := llm_common.LLMResponseStruct{Text: sent}
				if firstChunk {
					resp.IsStart = true
					firstChunk = false
				}
				if !send(resp, true) {
					streamMu.Lock()
					stopped = playbackStopped
					streamMu.Unlock()
					if stopped {
						continue
					}
					if story.ShouldCancelGenerationOnInterrupt(heardRunes(), threshold) {
						return context.Canceled
					}
					streamMu.Lock()
					playbackStopped = true
					streamMu.Unlock()
				}
			}
			streamMu.Lock()
			if streamErr != nil && story.ShouldCancelGenerationOnInterrupt(heardRunes(), threshold) {
				err := streamErr
				streamMu.Unlock()
				return err
			}
			streamMu.Unlock()
			return nil
		})

		if genErr != nil {
			streamMu.Lock()
			streamErr = genErr
			interrupted := ttsCtx.Err() != nil || errors.Is(genErr, context.Canceled)
			streamMu.Unlock()
			reason := storyFailureReason(interrupted, ttsCtx.Err(), genErr)
			log.Errorf("设备 %s 故事流式生成失败 story_id=%s reason=%s partial_runes=%d err=%v",
				c.DeviceID, plan.StoryID, reason, utf8.RuneCountInString(fullBuilder.String()), genErr)
			saveCheckpoint(interrupted)
			finish()
			return
		}

		result, saveErr := svc.SaveGeneratedStory(context.Background(), req, plan, fullBuilder.String())
		if saveErr != nil {
			streamMu.Lock()
			streamErr = saveErr
			streamMu.Unlock()
			log.Errorf("设备 %s 故事落库失败: %v", c.DeviceID, saveErr)
			finish()
			return
		}
		streamMu.Lock()
		savedComplete = true
		stopped := playbackStopped
		streamMu.Unlock()
		session.UpdateStoryPlaybackFromResult(result)

		if !stopped {
			if tail := sentenceBuf.Flush(); tail != "" {
				resp := llm_common.LLMResponseStruct{Text: tail, IsEnd: true}
				if firstChunk {
					resp.IsStart = true
				}
				send(resp, true)
				return
			}
			finish()
			return
		}
		// 保护续写完成：正文已落库，无需再推 TTS。
		log.Infof("设备 %s 故事保护续写完成 story_id=%s total_runes=%d",
			c.DeviceID, plan.StoryID, utf8.RuneCountInString(fullBuilder.String()))
	}()

	log.Infof("设备 %s 开始流式故事 story_id=%s theme=%q continuation=%t filler=%t",
		c.DeviceID, plan.StoryID, plan.Params.Theme, plan.IsContinuation, filler != "")
	return session.llmManager.HandleLLMResponseChannelAsyncWithOptions(ttsCtx, nil, responseChan, llmResponseChannelOptions{
		ttsTurnEndPolicy:          ttsTurnEndPolicyNone,
		deferProtocolTtsStop:      true,
		waitTtsDrainWithoutCancel: true,
		onEndFunc: func(err error, args ...any) {
			defer c.endStoryStreamGuard()
			streamMu.Lock()
			genErr := streamErr
			spokenStory := storySpokenBuilder.String()
			complete := savedComplete
			streamMu.Unlock()
			if genErr != nil && err == nil && story.ShouldCancelGenerationOnInterrupt(
				utf8.RuneCountInString(story.MergeHeardStoryText(plan.SpokenBaseline, spokenStory)), threshold) {
				err = genErr
			}
			playbackOK := err == nil && ttsCtx.Err() == nil
			storySentChars, lastSent, lastSentIdx, segments, _, snapOK := session.StoryPlaybackSnapshot()
			if !snapOK && spokenStory != "" {
				storySentChars = len([]rune(spokenStory))
			}

			// 有听过内容或生成已完整时同步进度（含打断）。
			if complete || strings.TrimSpace(spokenStory) != "" || storySentChars > 0 {
				c.syncStoryStreamProgress(svc, c.DeviceID, plan.StoryID, spokenStory, storySentChars, lastSent, lastSentIdx, segments, complete, playbackOK)
			}
			session.ClearStoryPlayback()

			spokenText := storySpokenText(filler, spokenStory)
			if updater := session.storyProgressUpdater; updater != nil {
				remember := complete || strings.TrimSpace(spokenStory) != ""
				updater.RememberStoryForFollowUp(ttsCtx, session, plan.StoryID, spokenText, remember)
			} else if complete || strings.TrimSpace(spokenStory) != "" {
				c.RememberStoryForFollowUp(context.Background(), session, plan.StoryID, spokenText, complete)
			}
			if session.llmManager != nil {
				result := llmHandleResultFromArgs(args)
				result.deferTtsTurnEnd = false
				log.Infof("设备 %s 故事流 TTS 收尾 story_id=%s complete=%t playback_ok=%t spoken_runes=%d err=%v",
					c.DeviceID, plan.StoryID, complete, playbackOK, storySentChars, err)
				session.llmManager.finishTTSTurnWithReason(ttsCtx, err, result, "child_story_stream playback complete")
			}
			if err != nil {
				log.Errorf("设备 %s 故事流式播报结束异常: %v", c.DeviceID, err)
			}
		},
	})
}
