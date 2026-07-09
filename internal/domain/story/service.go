package story

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

func persistContext(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return context.WithoutCancel(ctx)
}

// GenerateFunc 故事专用 LLM 同步生成。
type GenerateFunc func(ctx context.Context, systemPrompt, userPrompt string) (string, error)

var ErrGenerateUnavailable = errors.New("story generate func unavailable")

type Service struct {
	store    *Store
	cfg      Config
	generate GenerateFunc
	prefSync PreferenceSync
}

func NewService(generate GenerateFunc) *Service {
	cfg := LoadConfig()
	return &Service{
		store:    NewStore(cfg),
		cfg:      cfg,
		generate: generate,
		prefSync: noopPreferenceSync{},
	}
}

func NewServiceWithStore(store *Store, cfg Config, generate GenerateFunc) *Service {
	return &Service{store: store, cfg: cfg, generate: generate, prefSync: noopPreferenceSync{}}
}

func (s *Service) Store() *Store { return s.store }

func (s *Service) Config() Config { return s.cfg }

func (s *Service) FillerText(params StoryParams) string {
	return BuildFillerText(params, s.cfg)
}

// GeneratePlan 流式生成前的参数与 prompt 计划。
type GeneratePlan struct {
	Params                 StoryParams
	AssumedFields          map[string]string
	SystemPrompt           string
	UserPrompt             string
	StoryID                string
	ContinueFrom           string
	IsContinuation         bool
	SpokenBaseline         string   // 续讲前听众已听前缀（整句边界）
	DraftPlaybackSentences []string // 续讲前先 TTS 补播的草稿句
	StoryMeta              StoryMeta
}

// PlanGenerate 解析参数并构建故事生成 prompt；缺参时返回 need_params。
func (s *Service) PlanGenerate(ctx context.Context, req ToolRequest) (*ToolResult, *GeneratePlan, error) {
	resolved := ResolveParams(req.StoryParams, req.MemoryContext, s.cfg, req.Now)
	if len(resolved.Missing) > 0 {
		return &ToolResult{
			Status:  StatusNeedParams,
			Missing: resolved.Missing,
			Message: BuildQuestionForMissing(resolved.Missing),
		}, nil, nil
	}
	if s.generate == nil {
		return nil, nil, ErrGenerateUnavailable
	}
	params := resolved.Params
	NormalizeStoryParams(&params)
	weakThemes := s.store.TopReplayThemes(ctx, req.DeviceID, 3)
	return nil, &GeneratePlan{
		Params:        params,
		AssumedFields: resolved.AssumedFields,
		SystemPrompt:  BuildSystemPrompt(params),
		UserPrompt:    BuildUserPrompt(params, weakThemes),
		StoryID:       uuid.NewString(),
	}, nil
}

// SaveDraftStory 流式生成开始前落库占位，便于管理端可见与打断后续写。
func (s *Service) SaveDraftStory(ctx context.Context, req ToolRequest, plan *GeneratePlan) error {
	if plan == nil {
		return errors.New("generate plan is nil")
	}
	params := plan.Params
	record := &StoryRecord{
		StoryID:        plan.StoryID,
		DeviceID:       req.DeviceID,
		AgentID:        req.AgentID,
		Title:          TitleFromTheme(params.Theme),
		FullText:       "",
		Segments:       nil,
		Mode:           params.RequestType,
		AgeBand:        params.AgeBand,
		LastPlayStatus: PlayStatusPlaying,
		ParamsSnapshot: map[string]any{
			"theme":          params.Theme,
			"style":          params.Style,
			"age_band":       params.AgeBand,
			"assumed_fields": plan.AssumedFields,
			"draft":          true,
		},
	}
	setGenerationComplete(record, false)
	if params.IsBedtime != nil && *params.IsBedtime {
		record.Tags = append(record.Tags, "bedtime")
		record.Mode = StoryModeBedtime
	}
	return s.store.Save(persistContext(ctx), record)
}

// SavePartialStory 打断或失败时保存已生成片段（GenerationComplete=false）。
func (s *Service) SavePartialStory(ctx context.Context, req ToolRequest, plan *GeneratePlan, partial string, interrupted bool) error {
	if plan == nil {
		return errors.New("generate plan is nil")
	}
	partial = TrimStoryOutput(partial)
	meta, body := StripLeadingMeta(partial)
	if body != "" {
		partial = body
	}
	if plan.IsContinuation && plan.ContinueFrom != "" {
		partial = mergeStoryText(plan.ContinueFrom, partial)
	}
	status := PlayStatusAbandoned
	if interrupted {
		status = PlayStatusInterrupted
	}
	saveCtx := persistContext(ctx)
	rec, err := s.store.Get(saveCtx, req.DeviceID, plan.StoryID)
	if err != nil {
		if partial == "" {
			return nil
		}
		return s.SaveDraftStory(ctx, req, plan)
	}
	if partial != "" {
		rec.FullText = partial
		rec.Segments = SegmentText(partial)
	}
	applyStoryMeta(rec, mergeStoryMeta(plan.StoryMeta, meta), plan.Params)
	if rec.Title == "" || looksLikeStoryOpening(rec.Title) {
		rec.Title = ResolveStoryTitle(themeFromRecord(rec, plan), rec.FullText, rec.Title)
	}
	setGenerationComplete(rec, false)
	rec.LastPlayStatus = status
	rec.LastPlayedAt = time.Now()
	return s.store.Save(saveCtx, rec)
}

func mergeStoryText(base, addition string) string {
	base = strings.TrimSpace(base)
	addition = TrimStoryOutput(addition)
	if base == "" {
		return addition
	}
	if addition == "" {
		return base
	}
	if strings.HasPrefix(addition, base) {
		return addition
	}
	return base + addition
}

// SaveGeneratedStory 将流式/同步生成的全文落库并返回朗读结果。
func (s *Service) SaveGeneratedStory(ctx context.Context, req ToolRequest, plan *GeneratePlan, fullText string) (*ToolResult, error) {
	if plan == nil {
		return nil, errors.New("generate plan is nil")
	}
	fullText = TrimStoryOutput(fullText)
	meta, body := StripLeadingMeta(fullText)
	if body != "" {
		fullText = body
	}
	if plan.IsContinuation && plan.ContinueFrom != "" {
		fullText = mergeStoryText(plan.ContinueFrom, fullText)
	}
	if fullText == "" {
		return &ToolResult{Status: StatusNotFound, Message: "故事生成失败，请稍后再试"}, nil
	}
	params := plan.Params
	segments := SegmentText(fullText)
	saveCtx := persistContext(ctx)
	existing, _ := s.store.Get(saveCtx, req.DeviceID, plan.StoryID)

	record := &StoryRecord{
		StoryID:        plan.StoryID,
		DeviceID:       req.DeviceID,
		AgentID:        req.AgentID,
		Title:          ResolveStoryTitle(params.Theme, fullText, TitleFromTheme(params.Theme)),
		FullText:       fullText,
		Segments:       segments,
		Mode:           params.RequestType,
		AgeBand:        params.AgeBand,
		LastPlayStatus: PlayStatusPlaying,
		ParamsSnapshot: map[string]any{
			"theme":          params.Theme,
			"style":          params.Style,
			"age_band":       params.AgeBand,
			"assumed_fields": plan.AssumedFields,
		},
	}
	applyStoryMeta(record, mergeStoryMeta(plan.StoryMeta, meta), params)
	if params.IsBedtime != nil && *params.IsBedtime {
		record.Tags = append(record.Tags, "bedtime")
	}
	if existing != nil {
		record.CreatedAt = existing.CreatedAt
		record.PlayCount = existing.PlayCount
		record.CompleteCount = existing.CompleteCount
		record.LastPlayedAt = existing.LastPlayedAt
		if existing.LastPlayStatus == PlayStatusInterrupted && IsGenerationComplete(existing) {
			record.LastPosition = existing.LastPosition
			record.LastPlayStatus = PlayStatusInterrupted
		}
	}
	setGenerationComplete(record, true)
	if err := s.store.Save(saveCtx, record); err != nil {
		return nil, err
	}
	_ = s.store.RecordPlayStart(saveCtx, req.DeviceID, record.StoryID)
	return s.buildSpeakResult(record, 0, StatusReady, ""), nil
}

// ResolveResumeRecord 查找续讲目标故事。
func (s *Service) ResolveResumeRecord(ctx context.Context, req ToolRequest) (*StoryRecord, error) {
	var rec *StoryRecord
	var err error
	if req.StoryRef != "" && req.StoryRef != StoryRefLast {
		rec, err = s.store.Get(ctx, req.DeviceID, req.StoryRef)
	} else {
		rec, err = s.store.GetLastInterrupted(ctx, req.DeviceID)
		if err != nil {
			rec, err = s.store.GetLast(ctx, req.DeviceID)
		}
	}
	if err != nil || rec == nil {
		return nil, err
	}
	if theme := NormalizeThemeKey(req.Theme); theme != "" && !ThemeMatchesRecord(theme, rec) {
		if themed, findErr := s.store.FindLatestByTheme(ctx, req.DeviceID, theme, true); findErr == nil && themed != nil {
			rec = themed
		}
	}
	return rec, nil
}

// PlanContinueGenerate 为未完成生成的故事构建续写计划（B 方案：先补播草稿句，再从全文末 LLM 续写）。
func (s *Service) PlanContinueGenerate(_ context.Context, rec *StoryRecord) (*GeneratePlan, error) {
	if rec == nil || !hasStoryContent(rec) {
		return nil, errors.New("no story to continue")
	}
	params := paramsFromRecord(rec)
	NormalizeStoryParams(&params)
	written := strings.TrimSpace(rec.FullText)
	return &GeneratePlan{
		Params:                 params,
		SystemPrompt:           BuildContinueSystemPrompt(params),
		UserPrompt:             BuildContinueUserPrompt(params, written),
		StoryID:                rec.StoryID,
		ContinueFrom:           written,
		IsContinuation:         true,
		SpokenBaseline:         SpokenTextPrefix(rec),
		DraftPlaybackSentences: DraftPlaybackSentences(rec),
	}, nil
}

// SaveGenerationCheckpoint 保存生成中断时的片段与断点（不标记为完整生成）。
func (s *Service) SaveGenerationCheckpoint(ctx context.Context, req ToolRequest, plan *GeneratePlan, partial, heardStoryText string, interrupted bool) error {
	if err := s.SavePartialStory(ctx, req, plan, partial, interrupted); err != nil {
		return err
	}
	if plan == nil || plan.StoryID == "" {
		return nil
	}
	heardStoryText = strings.TrimSpace(heardStoryText)
	if heardStoryText == "" {
		return nil
	}
	saveCtx := persistContext(ctx)
	rec, err := s.store.Get(saveCtx, req.DeviceID, plan.StoryID)
	if err != nil {
		return err
	}
	if rec.FullText != "" {
		rec.LastPosition = ComputePlayPosition(rec.FullText, heardStoryText)
	}
	return s.store.Save(saveCtx, rec)
}

func (s *Service) Handle(ctx context.Context, req ToolRequest) (*ToolResult, error) {
	if !s.cfg.Enabled {
		return &ToolResult{Status: StatusNotFound, Message: "故事功能未启用"}, nil
	}

	switch req.Action {
	case ActionGenerate, "":
		return s.handleGenerate(ctx, req)
	case ActionReplay:
		return s.handleReplay(ctx, req)
	case ActionResume:
		return s.handleResume(ctx, req)
	case ActionListRecent:
		return s.handleListRecent(ctx, req)
	default:
		return &ToolResult{Status: StatusNotFound, Message: "不支持的操作: " + req.Action}, nil
	}
}

func (s *Service) handleGenerate(ctx context.Context, req ToolRequest) (*ToolResult, error) {
	resolved := ResolveParams(req.StoryParams, req.MemoryContext, s.cfg, req.Now)
	if len(resolved.Missing) > 0 {
		return &ToolResult{
			Status:  StatusNeedParams,
			Missing: resolved.Missing,
			Message: BuildQuestionForMissing(resolved.Missing),
		}, nil
	}

	params := resolved.Params
	NormalizeStoryParams(&params)
	weakThemes := s.store.TopReplayThemes(ctx, req.DeviceID, 3)
	if s.generate == nil {
		return nil, ErrGenerateUnavailable
	}

	systemPrompt := BuildSystemPrompt(params)
	userPrompt := BuildUserPrompt(params, weakThemes)
	fullText, err := s.generate(ctx, systemPrompt, userPrompt)
	if err != nil {
		return nil, err
	}
	fullText = trimStoryOutput(fullText)
	meta, body := StripLeadingMeta(fullText)
	if body != "" {
		fullText = body
	}
	if fullText == "" {
		return &ToolResult{Status: StatusNotFound, Message: "故事生成失败，请稍后再试"}, nil
	}

	segments := SegmentText(fullText)
	record := &StoryRecord{
		DeviceID:       req.DeviceID,
		AgentID:        req.AgentID,
		Title:          ResolveStoryTitle(params.Theme, fullText, TitleFromTheme(params.Theme)),
		FullText:       fullText,
		Segments:       segments,
		Mode:           params.RequestType,
		AgeBand:        params.AgeBand,
		LastPlayStatus: PlayStatusPlaying,
		ParamsSnapshot: map[string]any{
			"theme":          params.Theme,
			"style":          params.Style,
			"age_band":       params.AgeBand,
			"assumed_fields": resolved.AssumedFields,
		},
	}
	applyStoryMeta(record, meta, params)
	if params.IsBedtime != nil && *params.IsBedtime {
		record.Tags = append(record.Tags, "bedtime")
	}
	setGenerationComplete(record, true)
	if err := s.store.Save(ctx, record); err != nil {
		return nil, err
	}
	_ = s.store.RecordPlayStart(ctx, req.DeviceID, record.StoryID)

	return s.buildSpeakResult(record, 0, StatusReady, ""), nil
}

func (s *Service) handleReplay(ctx context.Context, req ToolRequest) (*ToolResult, error) {
	ref := req.StoryRef
	if ref == "" {
		ref = StoryRefLast
	}
	theme := NormalizeThemeKey(req.Theme)

	var records []StoryRecord

	// 用户点名主题复播（如「再讲一遍女娲补天」）时优先按主题找有正文的记录。
	if theme != "" && (ref == StoryRefLast || req.StoryRef == "") {
		if rec, err := s.store.FindLatestByTheme(ctx, req.DeviceID, theme, true); err == nil && rec != nil {
			records = []StoryRecord{*rec}
		} else {
			// 无已保存正文时重新生成同主题故事。
			genReq := req
			genReq.Action = ActionGenerate
			return s.handleGenerate(ctx, genReq)
		}
	}

	if len(records) == 0 {
		switch ref {
		case StoryRefLast:
			rec, e := s.store.GetLast(ctx, req.DeviceID)
			if e != nil {
				return &ToolResult{Status: StatusNotFound, Message: "还没有讲过的故事记录，要不要听一个新的？"}, nil
			}
			records = []StoryRecord{*rec}
		case StoryRefLastNight:
			start, end := LastNightWindow(req.Now, s.cfg.LastNightStartHour, s.cfg.LastNightEndHour)
			list, e := s.store.ListInWindow(ctx, req.DeviceID, start, end, 10)
			if e != nil || len(list) == 0 {
				return &ToolResult{Status: StatusNotFound, Message: "没有找到昨晚讲过的故事，要不要听一个新的？"}, nil
			}
			records = list
		case StoryRefFavorite:
			ids, e := s.store.backend.ZRevRange(ctx, s.store.byReplayKey(req.DeviceID), 0, 0)
			if e != nil || len(ids) == 0 {
				return &ToolResult{Status: StatusNotFound, Message: "还没有常听的故事记录"}, nil
			}
			rec, e := s.store.Get(ctx, req.DeviceID, ids[0])
			if e != nil {
				return &ToolResult{Status: StatusNotFound, Message: "没有找到故事记录"}, nil
			}
			records = []StoryRecord{*rec}
		default:
			rec, e := s.store.Get(ctx, req.DeviceID, ref)
			if e != nil {
				return &ToolResult{Status: StatusNotFound, Message: "没有找到指定的故事"}, nil
			}
			records = []StoryRecord{*rec}
		}
	}

	if len(records) > 1 {
		cands := make([]StoryCandidate, 0, len(records))
		for _, r := range records {
			cands = append(cands, StoryCandidate{
				StoryID: r.StoryID, Title: r.Title,
				LastPlayedAt: r.LastPlayedAt, PlayCount: r.PlayCount,
			})
		}
		return &ToolResult{
			Status:     StatusCandidates,
			Candidates: cands,
			Message:    "找到多个故事，想问用户想听哪一个",
		}, nil
	}

	rec := records[0]
	if !hasStoryContent(&rec) {
		if theme != "" {
			genReq := req
			genReq.Action = ActionGenerate
			return s.handleGenerate(ctx, genReq)
		}
		return &ToolResult{Status: StatusNotFound, Message: "这个故事还没有保存全文，我重新给你讲一个吧。"}, nil
	}

	startSeg := 0
	if req.FromBeginning != nil && !*req.FromBeginning && rec.LastPosition.SegmentIndex > 0 {
		startSeg = rec.LastPosition.SegmentIndex
	}
	if len(rec.Segments) > 0 && startSeg >= len(rec.Segments) {
		startSeg = 0
	}
	_ = s.store.RecordPlayStart(persistContext(ctx), req.DeviceID, rec.StoryID)
	result := s.buildSpeakResult(&rec, startSeg, StatusReplay, "")
	if strings.TrimSpace(result.TextToSpeak) == "" {
		if theme != "" {
			genReq := req
			genReq.Action = ActionGenerate
			return s.handleGenerate(ctx, genReq)
		}
		return &ToolResult{Status: StatusNotFound, Message: "这个故事还没有保存全文，我重新给你讲一个吧。"}, nil
	}
	result.SuggestNewStory = s.store.ShouldSuggestNewStory(&rec)
	return result, nil
}

func (s *Service) handleResume(ctx context.Context, req ToolRequest) (*ToolResult, error) {
	rec, err := s.ResolveResumeRecord(ctx, req)
	if err != nil || rec == nil {
		return &ToolResult{Status: StatusNotFound, Message: "没有找到可以续讲的故事，要不要从头讲一个新的？"}, nil
	}

	if !hasStoryContent(rec) {
		if theme := NormalizeThemeKey(req.Theme); theme != "" {
			genReq := req
			genReq.Action = ActionGenerate
			return s.handleGenerate(ctx, genReq)
		}
		return &ToolResult{Status: StatusNotFound, Message: "上次的故事还没保存下来，要不要从头讲一个新的？"}, nil
	}

	if !IsGenerationComplete(rec) {
		if !s.cfg.StreamEnabled {
			return &ToolResult{Status: StatusNotFound, Message: "上次的故事还没生成完，要不再讲一个新的？"}, nil
		}
		return &ToolResult{
			Status:  StatusStreaming,
			StoryID: rec.StoryID,
			Title:   rec.Title,
			Message: "故事续写中",
			Meta:    map[string]any{"continue_generate": true},
		}, nil
	}

	startSeg, prefix, body := ResumeSpeakPlan(rec)
	_ = s.store.RecordPlayStart(persistContext(ctx), req.DeviceID, rec.StoryID)
	text := body
	if prefix != "" && text != "" {
		text = prefix + text
	}
	result := &ToolResult{
		Status:       StatusResume,
		StoryID:      rec.StoryID,
		Title:        rec.Title,
		TextToSpeak:  text,
		Segments:     rec.Segments,
		StartSegment: startSeg,
		Message:      rec.Title,
		Meta: map[string]any{
			"age_band": rec.AgeBand,
			"mode":     rec.Mode,
		},
	}
	if strings.TrimSpace(result.TextToSpeak) == "" {
		if theme := NormalizeThemeKey(req.Theme); theme != "" {
			genReq := req
			genReq.Action = ActionGenerate
			return s.handleGenerate(ctx, genReq)
		}
		return &ToolResult{Status: StatusNotFound, Message: "上次的故事还没保存下来，要不要从头讲一个新的？"}, nil
	}
	return result, nil
}

func (s *Service) handleListRecent(ctx context.Context, req ToolRequest) (*ToolResult, error) {
	since := req.Now.Add(-7 * 24 * time.Hour)
	list, err := s.store.ListRecent(ctx, req.DeviceID, since, 10)
	if err != nil {
		return nil, err
	}
	cands := make([]StoryCandidate, 0, len(list))
	for _, r := range list {
		cands = append(cands, StoryCandidate{
			StoryID: r.StoryID, Title: r.Title,
			LastPlayedAt: r.LastPlayedAt, PlayCount: r.PlayCount,
		})
	}
	return &ToolResult{
		Status:     StatusCandidates,
		Candidates: cands,
		Message:    "近期故事列表",
		Meta:       map[string]any{"count": len(cands)},
	}, nil
}

func (s *Service) buildSpeakResult(rec *StoryRecord, startSeg int, status string, prefix string) *ToolResult {
	text := TextFromSegmentIndex(rec.Segments, startSeg)
	if prefix != "" && text != "" {
		text = prefix + text
	}
	if text == "" {
		text = rec.FullText
		startSeg = 0
	}
	return &ToolResult{
		Status:       status,
		StoryID:      rec.StoryID,
		Title:        rec.Title,
		TextToSpeak:  text,
		Segments:     rec.Segments,
		StartSegment: startSeg,
		Message:      rec.Title,
		Meta: map[string]any{
			"age_band": rec.AgeBand,
			"mode":     rec.Mode,
		},
	}
}

func (s *Service) UpdatePlaybackProgress(ctx context.Context, deviceID, storyID string, pos PlayPosition, interrupted, completed bool) error {
	status := PlayStatusPlaying
	if interrupted {
		status = PlayStatusInterrupted
	}
	if completed {
		status = PlayStatusCompleted
	}
	return s.store.UpdateProgress(ctx, deviceID, storyID, pos, status, completed)
}

func TrimStoryOutput(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")
	return strings.TrimSpace(s)
}

func trimStoryOutput(s string) string { return TrimStoryOutput(s) }
