package story_persist

import (
	"context"
	"encoding/json"
	"errors"
	"math/rand"
	"strings"
	"time"
	"unicode/utf8"

	"dili/manager/backend/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrNotFound = errors.New("故事不存在")
)

const (
	PoolNamed   = "named"
	PoolOpen    = "open"
	PoolBedtime = "bedtime"
)

type UpsertAssetRequest struct {
	StoryID            string         `json:"story_id"`
	Title              string         `json:"title"`
	ThemeKey           string         `json:"theme_key"`
	PoolKind           string         `json:"pool_kind"`
	CanonicalKey       string         `json:"canonical_key"`
	Aliases            []string       `json:"aliases"`
	Mode               string         `json:"mode"`
	NarrationMode      string         `json:"narration_mode"`
	AgeBand            string         `json:"age_band"`
	FullText           string         `json:"full_text"`
	Segments           []string       `json:"segments"`
	GenerationComplete bool           `json:"generation_complete"`
	CreatorDeviceSN    string         `json:"creator_device_sn"`
	CreatorUserID      uint           `json:"creator_user_id"`
	ParamsSnapshot     map[string]any `json:"params_snapshot"`
}

type UpsertPlaybackRequest struct {
	DeviceSN          string    `json:"device_sn"`
	StoryID           string    `json:"story_id"`
	UserID            uint      `json:"user_id"`
	AgentID           string    `json:"agent_id"`
	LastPlayStatus    string    `json:"last_play_status"`
	CharOffset        int       `json:"char_offset"`
	SegmentIndex      int       `json:"segment_index"`
	LastSentenceIndex int       `json:"last_sentence_index"`
	LastSentence      string    `json:"last_sentence"`
	PlayCount         int       `json:"play_count"`
	CompleteCount     int       `json:"complete_count"`
	LastPlayedAt      time.Time `json:"last_played_at"`
}

type FindShareableQuery struct {
	PoolKind     string `json:"pool_kind"`
	Theme        string `json:"theme"`
	AgeBand      string `json:"age_band"`
	DeviceSN     string `json:"device_sn"`
	ExcludeDays  int    `json:"exclude_days"`
	TopK         int    `json:"top_k"`
	RandSeed     int64  `json:"-"` // 测试可注入；0 用时间种子
}

type AssetView struct {
	StoryID            string         `json:"story_id"`
	Title              string         `json:"title"`
	ThemeKey           string         `json:"theme_key"`
	PoolKind           string         `json:"pool_kind,omitempty"`
	CanonicalKey       string         `json:"canonical_key,omitempty"`
	Aliases            []string       `json:"aliases,omitempty"`
	Mode               string         `json:"mode"`
	NarrationMode      string         `json:"narration_mode"`
	AgeBand            string         `json:"age_band"`
	FullText           string         `json:"full_text"`
	Segments           []string       `json:"segments"`
	GenerationComplete bool           `json:"generation_complete"`
	Shareable          bool           `json:"shareable"`
	ParamsSnapshot     map[string]any `json:"params_snapshot,omitempty"`
	TextLength         int            `json:"text_length"`
	ReuseCount         int            `json:"reuse_count,omitempty"`
	CreatorDeviceSN    string         `json:"creator_device_sn,omitempty"`
	CreatedAt          time.Time      `json:"created_at,omitempty"`
	UpdatedAt          time.Time      `json:"updated_at,omitempty"`
}

// ListAssetsQuery 管理端列表筛选项。
type ListAssetsQuery struct {
	Q        string
	PoolKind string
	Shareable *bool
	Page     int
	PageSize int
}

type ListAssetsResult struct {
	Total int         `json:"total"`
	Items []AssetView `json:"items"`
}

type Service struct {
	DB *gorm.DB
}

func NewService(db *gorm.DB) *Service {
	return &Service{DB: db}
}

func (s *Service) UpsertAsset(ctx context.Context, req UpsertAssetRequest) error {
	if s == nil || s.DB == nil {
		return errors.New("db unavailable")
	}
	storyID := strings.TrimSpace(req.StoryID)
	if storyID == "" {
		return errors.New("story_id required")
	}
	pool := strings.TrimSpace(req.PoolKind)
	if pool == "" && req.ParamsSnapshot != nil {
		if v, ok := req.ParamsSnapshot["pool_kind"].(string); ok {
			pool = strings.TrimSpace(v)
		}
	}
	canonical := strings.TrimSpace(req.CanonicalKey)
	if canonical == "" && req.ParamsSnapshot != nil {
		if v, ok := req.ParamsSnapshot["canonical_key"].(string); ok {
			canonical = strings.TrimSpace(v)
		}
	}
	aliases := append([]string(nil), req.Aliases...)
	if len(aliases) == 0 && req.ParamsSnapshot != nil {
		aliases = aliasesFromSnapshot(req.ParamsSnapshot["aliases"])
	}
	segJSON, _ := json.Marshal(req.Segments)
	paramsJSON, _ := json.Marshal(req.ParamsSnapshot)
	aliasJSON, _ := json.Marshal(aliases)
		shareable := req.GenerationComplete && strings.TrimSpace(req.FullText) != "" &&
			(pool == PoolNamed || pool == PoolOpen || pool == PoolBedtime)
		if !shareable {
			// 未完整生成的故事不入池、不保留别名索引。
			pool = ""
			canonical = ""
			aliases = nil
		}

		row := models.StoryAsset{
		StoryID:            storyID,
		Title:              strings.TrimSpace(req.Title),
		ThemeKey:           strings.TrimSpace(req.ThemeKey),
		PoolKind:           pool,
		CanonicalKey:       canonical,
		AliasesJSON:        string(aliasJSON),
		Mode:               strings.TrimSpace(req.Mode),
		NarrationMode:      strings.TrimSpace(req.NarrationMode),
		AgeBand:            strings.TrimSpace(req.AgeBand),
		FullText:           req.FullText,
		SegmentsJSON:       string(segJSON),
		GenerationComplete: req.GenerationComplete,
		Shareable:          shareable,
		CreatorDeviceSN:    strings.TrimSpace(req.CreatorDeviceSN),
		CreatorUserID:      req.CreatorUserID,
		ParamsSnapshotJSON: string(paramsJSON),
	}
	err := s.DB.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "story_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"title", "theme_key", "pool_kind", "canonical_key", "aliases_json",
			"mode", "narration_mode", "age_band",
			"full_text", "segments_json", "generation_complete", "shareable",
			"params_snapshot_json", "updated_at",
		}),
	}).Create(&row).Error
		if err != nil {
			return err
		}
		if !shareable {
			_ = s.DB.WithContext(ctx).Where("story_id = ?", storyID).Delete(&models.StoryAssetAlias{})
			return nil
		}
		return s.syncAliases(ctx, storyID, canonical, aliases)
	}

func aliasesFromSnapshot(v any) []string {
	switch a := v.(type) {
	case []string:
		return a
	case []any:
		var out []string
		for _, x := range a {
			if s, ok := x.(string); ok && strings.TrimSpace(s) != "" {
				out = append(out, strings.TrimSpace(s))
			}
		}
		return out
	default:
		return nil
	}
}

func (s *Service) syncAliases(ctx context.Context, storyID, canonical string, aliases []string) error {
	_ = s.DB.WithContext(ctx).Where("story_id = ?", storyID).Delete(&models.StoryAssetAlias{})
	keys := map[string]struct{}{}
	add := func(k string) {
		k = strings.TrimSpace(k)
		if k == "" {
			return
		}
		keys[k] = struct{}{}
	}
	add(canonical)
	for _, a := range aliases {
		add(a)
	}
	if len(keys) == 0 {
		return nil
	}
	now := time.Now()
	for k := range keys {
		row := models.StoryAssetAlias{
			AliasKey:     k,
			StoryID:      storyID,
			CanonicalKey: canonical,
			CreatedAt:    now,
			UpdatedAt:    now,
		}
		if err := s.DB.WithContext(ctx).Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "alias_key"}},
			DoUpdates: clause.AssignmentColumns([]string{"story_id", "canonical_key", "updated_at"}),
		}).Create(&row).Error; err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) UpsertPlayback(ctx context.Context, req UpsertPlaybackRequest) error {
	if s == nil || s.DB == nil {
		return errors.New("db unavailable")
	}
	deviceSN := strings.TrimSpace(req.DeviceSN)
	storyID := strings.TrimSpace(req.StoryID)
	if deviceSN == "" || storyID == "" {
		return errors.New("device_sn and story_id required")
	}
	playedAt := req.LastPlayedAt
	if playedAt.IsZero() {
		playedAt = time.Now()
	}
	row := models.StoryPlayback{
		DeviceSN:          deviceSN,
		StoryID:           storyID,
		UserID:            req.UserID,
		AgentID:           strings.TrimSpace(req.AgentID),
		LastPlayStatus:    strings.TrimSpace(req.LastPlayStatus),
		CharOffset:        req.CharOffset,
		SegmentIndex:      req.SegmentIndex,
		LastSentenceIndex: req.LastSentenceIndex,
		LastSentence:      req.LastSentence,
		PlayCount:         req.PlayCount,
		CompleteCount:     req.CompleteCount,
		LastPlayedAt:      playedAt,
	}
	return s.DB.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "device_sn"}, {Name: "story_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"user_id", "agent_id", "last_play_status", "char_offset", "segment_index",
			"last_sentence_index", "last_sentence", "play_count", "complete_count",
			"last_played_at", "updated_at",
		}),
	}).Create(&row).Error
}

func (s *Service) GetAsset(ctx context.Context, storyID string) (*AssetView, error) {
	storyID = strings.TrimSpace(storyID)
	if storyID == "" {
		return nil, ErrNotFound
	}
	var row models.StoryAsset
	if err := s.DB.WithContext(ctx).Where("story_id = ?", storyID).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return assetToView(&row), nil
}

// ListAssets 管理端分页列表（不含特长全文时可截断 preview——此处仍返回全文，列表接口限制长度在 controller）。
func (s *Service) ListAssets(ctx context.Context, q ListAssetsQuery) (*ListAssetsResult, error) {
	if s == nil || s.DB == nil {
		return nil, errors.New("db unavailable")
	}
	page := q.Page
	if page <= 0 {
		page = 1
	}
	pageSize := q.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	dbq := s.DB.WithContext(ctx).Model(&models.StoryAsset{})
	if pool := strings.TrimSpace(q.PoolKind); pool != "" {
		dbq = dbq.Where("pool_kind = ?", pool)
	}
	if q.Shareable != nil {
		dbq = dbq.Where("shareable = ?", *q.Shareable)
	}
	if kw := strings.TrimSpace(q.Q); kw != "" {
		like := "%" + kw + "%"
		dbq = dbq.Where(
			"title LIKE ? OR theme_key LIKE ? OR canonical_key LIKE ? OR story_id LIKE ?",
			like, like, like, like,
		)
	}

	var total int64
	if err := dbq.Count(&total).Error; err != nil {
		return nil, err
	}
	var rows []models.StoryAsset
	if err := dbq.Order("updated_at DESC").
		Offset((page - 1) * pageSize).Limit(pageSize).
		Find(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]AssetView, 0, len(rows))
	for i := range rows {
		v := assetToView(&rows[i])
		if v != nil {
			items = append(items, *v)
		}
	}
	return &ListAssetsResult{Total: int(total), Items: items}, nil
}

// DeleteAsset 删除资产、别名，并清理该 story 的 playback（避免孤儿进度）。
func (s *Service) DeleteAsset(ctx context.Context, storyID string) error {
	if s == nil || s.DB == nil {
		return errors.New("db unavailable")
	}
	storyID = strings.TrimSpace(storyID)
	if storyID == "" {
		return errors.New("story_id required")
	}
	res := s.DB.WithContext(ctx).Where("story_id = ?", storyID).Delete(&models.StoryAsset{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	_ = s.DB.WithContext(ctx).Where("story_id = ?", storyID).Delete(&models.StoryAssetAlias{}).Error
	_ = s.DB.WithContext(ctx).Where("story_id = ?", storyID).Delete(&models.StoryPlayback{}).Error
	return nil
}

// FindShareable 双池取用：排斥本设备近期已播，Top-K 随机。
func (s *Service) FindShareable(ctx context.Context, q FindShareableQuery) (*AssetView, error) {
	pool := strings.TrimSpace(q.PoolKind)
	if pool == "" {
		pool = PoolNamed
	}
	topK := q.TopK
	if topK <= 0 {
		topK = 5
	}
	excludeDays := q.ExcludeDays
	if excludeDays <= 0 {
		excludeDays = 7
	}

	view, err := s.findShareableOnce(ctx, pool, q, topK, excludeDays)
	if err == nil {
		return view, nil
	}
	if pool == PoolBedtime && errors.Is(err, ErrNotFound) {
		return s.findShareableOnce(ctx, PoolOpen, q, topK, excludeDays)
	}
	return nil, err
}

func (s *Service) findShareableOnce(ctx context.Context, pool string, q FindShareableQuery, topK, excludeDays int) (*AssetView, error) {
	excluded := s.recentStoryIDs(ctx, q.DeviceSN, excludeDays)
	var candidates []models.StoryAsset

	switch pool {
	case PoolNamed:
		theme := strings.TrimSpace(q.Theme)
		if theme == "" {
			return nil, ErrNotFound
		}
		storyIDs := s.resolveNamedStoryIDs(ctx, theme)
		if len(storyIDs) == 0 {
			return nil, ErrNotFound
		}
		dbq := s.DB.WithContext(ctx).Where(
			"shareable = ? AND generation_complete = ? AND pool_kind = ? AND story_id IN ?",
			true, true, PoolNamed, storyIDs,
		)
		dbq = applyAgeFilter(dbq, q.AgeBand)
		if len(excluded) > 0 {
			dbq = dbq.Where("story_id NOT IN ?", excluded)
		}
		if err := dbq.Order("reuse_count ASC, updated_at DESC").Limit(topK).Find(&candidates).Error; err != nil {
			return nil, err
		}
	case PoolOpen, PoolBedtime:
		dbq := s.DB.WithContext(ctx).Where(
			"shareable = ? AND generation_complete = ? AND pool_kind = ?",
			true, true, pool,
		)
		dbq = applyAgeFilter(dbq, q.AgeBand)
		if len(excluded) > 0 {
			dbq = dbq.Where("story_id NOT IN ?", excluded)
		}
		if err := dbq.Order("reuse_count ASC, updated_at DESC").Limit(topK).Find(&candidates).Error; err != nil {
			return nil, err
		}
	default:
		return nil, ErrNotFound
	}

	if len(candidates) == 0 {
		return nil, ErrNotFound
	}
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	if q.RandSeed != 0 {
		rng = rand.New(rand.NewSource(q.RandSeed))
	}
	picked := candidates[rng.Intn(len(candidates))]
	_ = s.DB.WithContext(ctx).Model(&models.StoryAsset{}).Where("id = ?", picked.ID).
		UpdateColumn("reuse_count", gorm.Expr("reuse_count + 1")).Error
	return assetToView(&picked), nil
}

func applyAgeFilter(q *gorm.DB, ageBand string) *gorm.DB {
	if age := strings.TrimSpace(ageBand); age != "" {
		return q.Where("age_band = ? OR age_band = '' OR age_band IS NULL", age)
	}
	return q
}

func (s *Service) resolveNamedStoryIDs(ctx context.Context, theme string) []string {
	theme = strings.TrimSpace(theme)
	if theme == "" {
		return nil
	}
	// 客户端可传「口语|规范」多键，逗号分隔
	keys := strings.Split(theme, ",")
	var cleaned []string
	for _, k := range keys {
		k = strings.TrimSpace(k)
		if k != "" {
			cleaned = append(cleaned, k)
		}
	}
	if len(cleaned) == 0 {
		return nil
	}
	var aliases []models.StoryAssetAlias
	_ = s.DB.WithContext(ctx).Where("alias_key IN ?", cleaned).Find(&aliases).Error
	idSet := map[string]struct{}{}
	for _, a := range aliases {
		idSet[a.StoryID] = struct{}{}
	}
	var assets []models.StoryAsset
	_ = s.DB.WithContext(ctx).Where(
		"shareable = ? AND pool_kind = ? AND (theme_key IN ? OR canonical_key IN ?)",
		true, PoolNamed, cleaned, cleaned,
	).Limit(20).Find(&assets).Error
	for _, a := range assets {
		idSet[a.StoryID] = struct{}{}
	}
	out := make([]string, 0, len(idSet))
	for id := range idSet {
		out = append(out, id)
	}
	return out
}

func (s *Service) recentStoryIDs(ctx context.Context, deviceSN string, excludeDays int) []string {
	deviceSN = strings.TrimSpace(deviceSN)
	if deviceSN == "" || excludeDays <= 0 {
		return nil
	}
	since := time.Now().Add(-time.Duration(excludeDays) * 24 * time.Hour)
	var plays []models.StoryPlayback
	_ = s.DB.WithContext(ctx).Where("device_sn = ? AND last_played_at >= ?", deviceSN, since).
		Select("story_id").Find(&plays).Error
	out := make([]string, 0, len(plays))
	seen := map[string]struct{}{}
	for _, p := range plays {
		if p.StoryID == "" {
			continue
		}
		if _, ok := seen[p.StoryID]; ok {
			continue
		}
		seen[p.StoryID] = struct{}{}
		out = append(out, p.StoryID)
	}
	return out
}

func (s *Service) ListPlaybacksByDevice(ctx context.Context, deviceSN string, limit int) ([]AssetView, []models.StoryPlayback, error) {
	deviceSN = strings.TrimSpace(deviceSN)
	if deviceSN == "" {
		return nil, nil, errors.New("device_sn required")
	}
	if limit <= 0 {
		limit = 50
	}
	var plays []models.StoryPlayback
	if err := s.DB.WithContext(ctx).Where("device_sn = ?", deviceSN).
		Order("last_played_at DESC").Limit(limit).Find(&plays).Error; err != nil {
		return nil, nil, err
	}
	views := make([]AssetView, 0, len(plays))
	for _, p := range plays {
		v, err := s.GetAsset(ctx, p.StoryID)
		if err != nil {
			views = append(views, AssetView{StoryID: p.StoryID})
			continue
		}
		views = append(views, *v)
	}
	return views, plays, nil
}

// DeletePlayback 删除单设备单故事播放态（不删 story_assets）。
func (s *Service) DeletePlayback(ctx context.Context, deviceSN, storyID string) (int64, error) {
	if s == nil || s.DB == nil {
		return 0, errors.New("db unavailable")
	}
	deviceSN = strings.TrimSpace(deviceSN)
	storyID = strings.TrimSpace(storyID)
	if deviceSN == "" || storyID == "" {
		return 0, errors.New("device_sn and story_id required")
	}
	res := s.DB.WithContext(ctx).Where("device_sn = ? AND story_id = ?", deviceSN, storyID).
		Delete(&models.StoryPlayback{})
	return res.RowsAffected, res.Error
}

// DeletePlaybacksByDevice 清空设备全部播放态（不删 story_assets）。
func (s *Service) DeletePlaybacksByDevice(ctx context.Context, deviceSN string) (int64, error) {
	if s == nil || s.DB == nil {
		return 0, errors.New("db unavailable")
	}
	deviceSN = strings.TrimSpace(deviceSN)
	if deviceSN == "" {
		return 0, errors.New("device_sn required")
	}
	res := s.DB.WithContext(ctx).Where("device_sn = ?", deviceSN).Delete(&models.StoryPlayback{})
	return res.RowsAffected, res.Error
}

func assetToView(row *models.StoryAsset) *AssetView {
	if row == nil {
		return nil
	}
	var segs []string
	_ = json.Unmarshal([]byte(row.SegmentsJSON), &segs)
	var params map[string]any
	_ = json.Unmarshal([]byte(row.ParamsSnapshotJSON), &params)
	var aliases []string
	_ = json.Unmarshal([]byte(row.AliasesJSON), &aliases)
	return &AssetView{
		StoryID:            row.StoryID,
		Title:              row.Title,
		ThemeKey:           row.ThemeKey,
		PoolKind:           row.PoolKind,
		CanonicalKey:       row.CanonicalKey,
		Aliases:            aliases,
		Mode:               row.Mode,
		NarrationMode:      row.NarrationMode,
		AgeBand:            row.AgeBand,
		FullText:           row.FullText,
		Segments:           segs,
		GenerationComplete: row.GenerationComplete,
		Shareable:          row.Shareable,
		ParamsSnapshot:     params,
		TextLength:         utf8.RuneCountInString(row.FullText),
		ReuseCount:         row.ReuseCount,
		CreatorDeviceSN:    row.CreatorDeviceSN,
		CreatedAt:          row.CreatedAt,
		UpdatedAt:          row.UpdatedAt,
	}
}

// SegmentText 简易分段（管理端落库用）。
func SegmentText(fullText string) []string {
	fullText = strings.TrimSpace(fullText)
	if fullText == "" {
		return nil
	}
	seps := []rune{'。', '！', '？', '；', '\n', '.', '!', '?'}
	isSep := func(r rune) bool {
		for _, s := range seps {
			if r == s {
				return true
			}
		}
		return false
	}
	runes := []rune(fullText)
	var out []string
	var buf strings.Builder
	flush := func() {
		s := strings.TrimSpace(buf.String())
		if s != "" {
			out = append(out, s)
		}
		buf.Reset()
	}
	for i, r := range runes {
		buf.WriteRune(r)
		if isSep(r) {
			flush()
			continue
		}
		if i == len(runes)-1 {
			flush()
		}
	}
	if len(out) == 0 {
		return []string{fullText}
	}
	// 合并至每段最多 3 句
	const maxPer = 3
	var segs []string
	var group strings.Builder
	n := 0
	for _, sent := range out {
		group.WriteString(sent)
		n++
		if n >= maxPer {
			segs = append(segs, group.String())
			group.Reset()
			n = 0
		}
	}
	if group.Len() > 0 {
		segs = append(segs, group.String())
	}
	return segs
}
