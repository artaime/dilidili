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

func (c *ChatManager) storyGenerationContext(parent context.Context) context.Context {
	if c == nil || c.clientState == nil {
		if parent != nil {
			return parent
		}
		return context.Background()
	}
	sessionCtx := c.clientState.SessionCtx.Get(c.clientState.Ctx)
	if sessionCtx != nil {
		return sessionCtx
	}
	if parent != nil {
		return parent
	}
	return c.clientState.Ctx
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
	return c.runStoryStream(ctx, req, plan, filler)
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
	return c.runStoryStream(ctx, req, plan, filler)
}

// runStoryStream 流式生成/续写：TTS 打断时同步停止 LLM；未完成生成不写播放进度。
func (c *ChatManager) runStoryStream(ctx context.Context, req story.ToolRequest, plan *story.GeneratePlan, filler string) error {
	c.cancelRetainedSessionCleanup("child_story_stream")
	session, err := c.ensureSession()
	if err != nil {
		return err
	}
	svc := c.getStoryService()

	session.ActivateStoryPlayback(&story.ToolResult{
		Status:       story.StatusReady,
		StoryID:      plan.StoryID,
		TextToSpeak:  filler,
		StartSegment: 0,
	})

	responseChan := make(chan llm_common.LLMResponseStruct, 16)
	sessionCtx := c.clientState.SessionCtx.Get(c.clientState.Ctx)
	ttsCtx := c.clientState.AfterAsrSessionCtx.Get(sessionCtx)
	genCtx, genCancel := context.WithTimeout(c.storyGenerationContext(ctx), 90*time.Second)

	var (
		fullBuilder        strings.Builder
		storySpokenBuilder strings.Builder
		sentenceBuf        story.SentenceBuffer
		metaFilter         story.MetaStreamFilter
		streamMu           sync.Mutex
		streamErr          error
		savedComplete      bool
	)

	// TTS 被打断时立即停止 LLM 生成。
	go func() {
		select {
		case <-ttsCtx.Done():
			genCancel()
		case <-genCtx.Done():
		}
	}()

	go func() {
		defer genCancel()
		defer close(responseChan)

		firstChunk := true
		send := func(resp llm_common.LLMResponseStruct, isStorySentence bool) bool {
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

		// B 方案：续写前先按整句补播未听草稿，避免从半句/半词机械切开。
		for _, sent := range plan.DraftPlaybackSentences {
			if ttsCtx.Err() != nil {
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
		if ttsCtx.Err() != nil {
			saveCheckpoint(true)
			finish()
			return
		}

		genErr := c.callLLMStreamForStory(genCtx, plan.SystemPrompt, plan.UserPrompt, func(chunk string) error {
			if chunk == "" {
				return nil
			}
			if ttsCtx.Err() != nil {
				return ttsCtx.Err()
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
				resp := llm_common.LLMResponseStruct{Text: sent}
				if firstChunk {
					resp.IsStart = true
					firstChunk = false
				}
				send(resp, true)
			}
			streamMu.Lock()
			if streamErr != nil {
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

		result, saveErr := svc.SaveGeneratedStory(genCtx, req, plan, fullBuilder.String())
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
		streamMu.Unlock()
		session.UpdateStoryPlaybackFromResult(result)

		if tail := sentenceBuf.Flush(); tail != "" {
			resp := llm_common.LLMResponseStruct{Text: tail, IsEnd: true}
			if firstChunk {
				resp.IsStart = true
			}
			send(resp, true)
			return
		}
		finish()
	}()

	log.Infof("设备 %s 开始流式故事 story_id=%s theme=%q continuation=%t filler=%t",
		c.DeviceID, plan.StoryID, plan.Params.Theme, plan.IsContinuation, filler != "")
	return session.llmManager.HandleLLMResponseChannelAsyncWithOptions(ttsCtx, nil, responseChan, llmResponseChannelOptions{
		ttsTurnEndPolicy:            ttsTurnEndPolicyNone,
		deferProtocolTtsStop:        true,
		waitTtsDrainWithoutCancel:   true,
		onEndFunc: func(err error, args ...any) {
			streamMu.Lock()
			genErr := streamErr
			spokenStory := storySpokenBuilder.String()
			complete := savedComplete
			streamMu.Unlock()
			if genErr != nil && err == nil {
				err = genErr
			}
			playbackOK := err == nil && ttsCtx.Err() == nil
			storySentChars, lastSent, lastSentIdx, segments, _, snapOK := session.StoryPlaybackSnapshot()
			if !snapOK && spokenStory != "" {
				storySentChars = len([]rune(spokenStory))
			}

			if complete && playbackOK {
				c.syncStoryStreamProgress(svc, c.DeviceID, plan.StoryID, spokenStory, storySentChars, lastSent, lastSentIdx, segments, true, true)
			}
			session.ClearStoryPlayback()

			spokenText := storySpokenText(filler, spokenStory)
			if updater := session.storyProgressUpdater; updater != nil {
				remember := playbackOK && (complete || strings.TrimSpace(spokenStory) != "")
				updater.RememberStoryForFollowUp(ttsCtx, session, plan.StoryID, spokenText, remember)
			} else if playbackOK && (complete || strings.TrimSpace(spokenStory) != "") {
				c.rememberRecentStoryByID(context.Background(), plan.StoryID)
				c.ensureStoryAssistantMessage(ttsCtx, session, spokenText)
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
