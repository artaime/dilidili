package device_story

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"dili/manager/backend/config"
	"dili/manager/backend/models"
	"dili/manager/backend/services/story_persist"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

var (
	ErrDeviceNotFound     = errors.New("设备不存在")
	ErrDeviceMissingSN    = errors.New("设备缺少 SN")
	ErrStoryNotFound      = errors.New("故事不存在")
	ErrRedisNotConfigured = errors.New("未配置 Redis，无法读取故事数据")

	playStatusCompleted   = "completed"
	playStatusInterrupted = "interrupted"
)

type storyRecord struct {
	StoryID            string         `json:"story_id"`
	Title              string         `json:"title"`
	FullText           string         `json:"full_text"`
	Segments           []string       `json:"segments"`
	Mode               string         `json:"mode"`
	AgeBand            string         `json:"age_band"`
	CreatedAt          time.Time      `json:"created_at"`
	LastPlayedAt       time.Time      `json:"last_played_at"`
	PlayCount          int            `json:"play_count"`
	CompleteCount      int            `json:"complete_count"`
	LastPlayStatus     string         `json:"last_play_status"`
	LastPosition       playPosition   `json:"last_position"`
	GenerationComplete bool           `json:"generation_complete,omitempty"`
	Tags               []string       `json:"tags"`
	ParamsSnapshot     map[string]any `json:"params_snapshot"`
}

type playPosition struct {
	SegmentIndex      int    `json:"segment_index"`
	CharOffset        int    `json:"char_offset"`
	LastSentenceIndex int    `json:"last_sentence_index,omitempty"`
	LastSentence      string `json:"last_sentence,omitempty"`
}

type StoryPositionView struct {
	SegmentIndex       int    `json:"segment_index"`
	SegmentTotal       int    `json:"segment_total"`
	ProgressPercent    int    `json:"progress_percent"`
	ProgressAvailable  bool   `json:"progress_available"`
	CharOffset         int    `json:"char_offset,omitempty"`
	LastSentenceIndex  int    `json:"last_sentence_index,omitempty"`
	LastSentence       string `json:"last_sentence,omitempty"`
}

type StoryListItem struct {
	StoryID            string            `json:"story_id"`
	Title              string            `json:"title"`
	Theme              string            `json:"theme,omitempty"`
	Genre              string            `json:"genre,omitempty"`
	Style              string            `json:"style,omitempty"`
	AgeBand            string            `json:"age_band,omitempty"`
	Mode               string            `json:"mode,omitempty"`
	Tags               []string          `json:"tags,omitempty"`
	CreatedAt          time.Time         `json:"created_at"`
	LastPlayedAt       time.Time         `json:"last_played_at"`
	PlayCount          int               `json:"play_count"`
	CompleteCount      int               `json:"complete_count"`
	LastPlayStatus     string            `json:"last_play_status"`
	LastPosition       StoryPositionView `json:"last_position"`
	GenerationComplete bool              `json:"generation_complete"`
	TextLength         int               `json:"text_length"`
	SegmentCount       int               `json:"segment_count"`
	TextPreview        string            `json:"text_preview"`
	AssumedFields      map[string]string `json:"assumed_fields,omitempty"`
}

type DeviceStoryListView struct {
	DeviceID uint            `json:"device_id"`
	DeviceSN string          `json:"device_sn"`
	AgentID  uint            `json:"agent_id"`
	Total    int             `json:"total"`
	Items    []StoryListItem `json:"items"`
}

type StoryDetailView struct {
	StoryListItem
	FullText string   `json:"full_text"`
	Segments []string `json:"segments"`
}

type redisStoryReader struct {
	client *redis.Client
	prefix string
}

type Service struct {
	DB        *gorm.DB
	Cfg       *config.Config
	reader    *redisStoryReader
	readerOnce sync.Once
	readerErr  error
}

func NewService(db *gorm.DB, cfg *config.Config) *Service {
	return &Service{DB: db, Cfg: cfg}
}

func (s *Service) ListDeviceStories(ctx context.Context, deviceID uint, limit int) (*DeviceStoryListView, error) {
	device, err := s.loadDevice(deviceID)
	if err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}

	if items, ok := s.listFromMySQL(ctx, device.DeviceName, limit); ok && len(items) > 0 {
		return &DeviceStoryListView{
			DeviceID: device.ID,
			DeviceSN: device.DeviceName,
			AgentID:  device.AgentID,
			Total:    len(items),
			Items:    items,
		}, nil
	}

	reader, err := s.storyReader()
	if err != nil {
		// MySQL 空且 Redis 不可用时返回空列表而非错误（利于迁移期）。
		if errors.Is(err, ErrRedisNotConfigured) {
			return &DeviceStoryListView{
				DeviceID: device.ID,
				DeviceSN: device.DeviceName,
				AgentID:  device.AgentID,
				Items:    []StoryListItem{},
			}, nil
		}
		return nil, err
	}

	records, err := reader.listRecent(ctx, device.DeviceName, limit)
	if err != nil {
		return nil, fmt.Errorf("读取故事列表失败: %w", err)
	}

	items := make([]StoryListItem, 0, len(records))
	for i := range records {
		items = append(items, mapStoryListItem(&records[i]))
	}

	return &DeviceStoryListView{
		DeviceID: device.ID,
		DeviceSN: device.DeviceName,
		AgentID:  device.AgentID,
		Total:    len(items),
		Items:    items,
	}, nil
}

func (s *Service) GetDeviceStory(ctx context.Context, deviceID uint, storyID string) (*StoryDetailView, error) {
	device, err := s.loadDevice(deviceID)
	if err != nil {
		return nil, err
	}
	storyID = strings.TrimSpace(storyID)
	if storyID == "" {
		return nil, ErrStoryNotFound
	}

	if view, ok := s.getFromMySQL(ctx, device.DeviceName, storyID); ok {
		return view, nil
	}

	reader, err := s.storyReader()
	if err != nil {
		return nil, ErrStoryNotFound
	}
	rec, err := reader.get(ctx, device.DeviceName, storyID)
	if err != nil {
		if errors.Is(err, redis.Nil) {
			// 回落：仅 asset 存在（聊天卡片跨设备资产）
			if view, ok := s.getAssetOnlyFromMySQL(ctx, storyID); ok {
				return view, nil
			}
			return nil, ErrStoryNotFound
		}
		return nil, fmt.Errorf("读取故事详情失败: %w", err)
	}
	item := mapStoryListItem(rec)
	return &StoryDetailView{
		StoryListItem: item,
		FullText:      rec.FullText,
		Segments:      rec.Segments,
	}, nil
}

func (s *Service) listFromMySQL(ctx context.Context, deviceSN string, limit int) ([]StoryListItem, bool) {
	if s == nil || s.DB == nil {
		return nil, false
	}
	persist := story_persist.NewService(s.DB)
	assets, plays, err := persist.ListPlaybacksByDevice(ctx, deviceSN, limit)
	if err != nil || len(plays) == 0 {
		return nil, false
	}
	items := make([]StoryListItem, 0, len(plays))
	for i, p := range plays {
		rec := mergePlaybackAsset(p, nil)
		if i < len(assets) {
			rec = mergePlaybackAsset(p, &assets[i])
		}
		items = append(items, mapStoryListItem(rec))
	}
	return items, true
}

func (s *Service) getFromMySQL(ctx context.Context, deviceSN, storyID string) (*StoryDetailView, bool) {
	if s == nil || s.DB == nil {
		return nil, false
	}
	persist := story_persist.NewService(s.DB)
	asset, err := persist.GetAsset(ctx, storyID)
	if err != nil {
		return nil, false
	}
	var play models.StoryPlayback
	_ = s.DB.WithContext(ctx).Where("device_sn = ? AND story_id = ?", deviceSN, storyID).First(&play).Error
	rec := mergePlaybackAsset(play, asset)
	item := mapStoryListItem(rec)
	return &StoryDetailView{StoryListItem: item, FullText: asset.FullText, Segments: asset.Segments}, true
}

func (s *Service) getAssetOnlyFromMySQL(ctx context.Context, storyID string) (*StoryDetailView, bool) {
	if s == nil || s.DB == nil {
		return nil, false
	}
	asset, err := story_persist.NewService(s.DB).GetAsset(ctx, storyID)
	if err != nil {
		return nil, false
	}
	rec := mergePlaybackAsset(models.StoryPlayback{StoryID: storyID}, asset)
	item := mapStoryListItem(rec)
	return &StoryDetailView{StoryListItem: item, FullText: asset.FullText, Segments: asset.Segments}, true
}

func mergePlaybackAsset(play models.StoryPlayback, asset *story_persist.AssetView) *storyRecord {
	rec := &storyRecord{
		StoryID:        play.StoryID,
		LastPlayStatus: play.LastPlayStatus,
		PlayCount:      play.PlayCount,
		CompleteCount:  play.CompleteCount,
		LastPlayedAt:   play.LastPlayedAt,
		LastPosition: playPosition{
			SegmentIndex:      play.SegmentIndex,
			CharOffset:        play.CharOffset,
			LastSentenceIndex: play.LastSentenceIndex,
			LastSentence:      play.LastSentence,
		},
	}
	if asset != nil {
		rec.StoryID = asset.StoryID
		rec.Title = asset.Title
		rec.FullText = asset.FullText
		rec.Segments = asset.Segments
		rec.Mode = asset.Mode
		rec.AgeBand = asset.AgeBand
		rec.GenerationComplete = asset.GenerationComplete
		rec.ParamsSnapshot = asset.ParamsSnapshot
		if rec.ParamsSnapshot == nil {
			rec.ParamsSnapshot = map[string]any{}
		}
		if asset.ThemeKey != "" {
			rec.ParamsSnapshot["theme"] = asset.ThemeKey
		}
		if asset.NarrationMode != "" {
			rec.ParamsSnapshot["narration_mode"] = asset.NarrationMode
		}
	}
	return rec
}

func (s *Service) loadDevice(deviceID uint) (*models.Device, error) {
	var device models.Device
	if err := s.DB.First(&device, deviceID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrDeviceNotFound
		}
		return nil, err
	}
	if strings.TrimSpace(device.DeviceName) == "" {
		return nil, ErrDeviceMissingSN
	}
	return &device, nil
}

func (s *Service) storyReader() (*redisStoryReader, error) {
	s.readerOnce.Do(func() {
		if s.Cfg == nil || s.Cfg.Redis == nil {
			s.readerErr = ErrRedisNotConfigured
			return
		}
		rc := s.Cfg.Redis
		if strings.TrimSpace(rc.Host) == "" {
			s.readerErr = ErrRedisNotConfigured
			return
		}
		port := rc.Port
		if port == 0 {
			port = 6379
		}
		client := redis.NewClient(&redis.Options{
			Addr:     fmt.Sprintf("%s:%d", rc.Host, port),
			Password: rc.Password,
			DB:       rc.DB,
		})
		pingCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := client.Ping(pingCtx).Err(); err != nil {
			s.readerErr = fmt.Errorf("连接 Redis 失败: %w", err)
			return
		}
		prefix := strings.TrimSpace(rc.KeyPrefix)
		if prefix == "" {
			prefix = "dilidili"
		}
		s.reader = &redisStoryReader{client: client, prefix: prefix}
	})
	return s.reader, s.readerErr
}

func (r *redisStoryReader) recordKey(deviceID, storyID string) string {
	return fmt.Sprintf("%s:story:%s:%s", r.prefix, deviceID, storyID)
}

func (r *redisStoryReader) byTimeKey(deviceID string) string {
	return fmt.Sprintf("%s:story:%s:by_time", r.prefix, deviceID)
}

func (r *redisStoryReader) get(ctx context.Context, deviceID, storyID string) (*storyRecord, error) {
	raw, err := r.client.Get(ctx, r.recordKey(deviceID, storyID)).Result()
	if err != nil {
		return nil, err
	}
	var rec storyRecord
	if err := json.Unmarshal([]byte(raw), &rec); err != nil {
		return nil, err
	}
	return &rec, nil
}

func (r *redisStoryReader) listRecent(ctx context.Context, deviceID string, limit int) ([]storyRecord, error) {
	ids, err := r.client.ZRevRange(ctx, r.byTimeKey(deviceID), 0, int64(limit-1)).Result()
	if err != nil {
		return nil, err
	}
	var out []storyRecord
	for _, id := range ids {
		rec, err := r.get(ctx, deviceID, id)
		if err != nil {
			continue
		}
		out = append(out, *rec)
	}
	return out, nil
}

func playbackProgress(rec *storyRecord) (segmentIndex, segmentTotal, percent int, showProgress bool) {
	if rec == nil {
		return 0, 0, 0, false
	}
	// 有正文或断点即可展示字数进度；未完整生成时分母为当前已生成长度。
	if strings.TrimSpace(rec.FullText) == "" && rec.LastPosition.CharOffset <= 0 && len(rec.Segments) == 0 {
		return 0, 0, 0, false
	}
	showProgress = true
	segmentTotal = len(rec.Segments)
	segmentIndex = rec.LastPosition.SegmentIndex
	if rec.LastPlayStatus == playStatusCompleted && isGenerationComplete(rec) {
		if segmentTotal > 0 {
			return segmentTotal - 1, segmentTotal, 100, showProgress
		}
		return 0, 0, 100, showProgress
	}
	if rec.FullText != "" && rec.LastPosition.CharOffset > 0 {
		totalRunes := utf8.RuneCountInString(rec.FullText)
		if totalRunes > 0 {
			percent = rec.LastPosition.CharOffset * 100 / totalRunes
			if percent > 100 {
				percent = 100
			}
		}
		if segmentTotal > 0 {
			acc := 0
			for i, seg := range rec.Segments {
				acc += utf8.RuneCountInString(seg)
				if acc >= rec.LastPosition.CharOffset {
					segmentIndex = i
					break
				}
			}
			if segmentIndex >= segmentTotal {
				segmentIndex = segmentTotal - 1
			}
		}
		return segmentIndex, segmentTotal, percent, showProgress
	}
	if segmentTotal == 0 {
		return segmentIndex, 0, 0, showProgress
	}
	if segmentIndex < 0 {
		segmentIndex = 0
	}
	percent = (segmentIndex + 1) * 100 / segmentTotal
	if percent > 100 {
		percent = 100
	}
	return segmentIndex, segmentTotal, percent, showProgress
}

func isGenerationComplete(rec *storyRecord) bool {
	if rec == nil {
		return false
	}
	if rec.GenerationComplete {
		return true
	}
	if rec.ParamsSnapshot != nil {
		if v, ok := rec.ParamsSnapshot["generation_complete"].(bool); ok {
			return v
		}
		if draft, ok := rec.ParamsSnapshot["draft"].(bool); ok && draft {
			return false
		}
	}
	return strings.TrimSpace(rec.FullText) != ""
}

func mapStoryListItem(rec *storyRecord) StoryListItem {
	theme, style, ageBand, genre, assumed := snapshotFields(rec)
	if ageBand == "" {
		ageBand = rec.AgeBand
	}
	segIdx, segTotal, pct, progressOK := playbackProgress(rec)
	preview := rec.FullText
	if utf8.RuneCountInString(preview) > 80 {
		preview = string([]rune(preview)[:80]) + "…"
	}
	genComplete := isGenerationComplete(rec)
	displayTitle := resolveDisplayTitle(theme, genre, rec)
	return StoryListItem{
		StoryID:            rec.StoryID,
		Title:              displayTitle,
		Theme:              theme,
		Genre:              genre,
		Style:              style,
		AgeBand:            ageBand,
		Mode:               rec.Mode,
		Tags:               rec.Tags,
		CreatedAt:          rec.CreatedAt,
		LastPlayedAt:       rec.LastPlayedAt,
		PlayCount:          rec.PlayCount,
		CompleteCount:      rec.CompleteCount,
		LastPlayStatus:     rec.LastPlayStatus,
		GenerationComplete: genComplete,
		LastPosition: StoryPositionView{
			SegmentIndex:      segIdx,
			SegmentTotal:      segTotal,
			ProgressPercent:   pct,
			ProgressAvailable: progressOK,
			CharOffset:        rec.LastPosition.CharOffset,
			LastSentenceIndex: rec.LastPosition.LastSentenceIndex,
			LastSentence:      rec.LastPosition.LastSentence,
		},
		TextLength:    utf8.RuneCountInString(rec.FullText),
		SegmentCount:  len(rec.Segments),
		TextPreview:   preview,
		AssumedFields: assumed,
	}
}

func snapshotFields(rec *storyRecord) (theme, style, ageBand, genre string, assumed map[string]string) {
	if rec == nil || rec.ParamsSnapshot == nil {
		return "", "", "", "", nil
	}
	if v, ok := rec.ParamsSnapshot["theme"].(string); ok {
		theme = v
	}
	if v, ok := rec.ParamsSnapshot["genre"].(string); ok {
		genre = v
	}
	if v, ok := rec.ParamsSnapshot["style"].(string); ok {
		style = v
	}
	if v, ok := rec.ParamsSnapshot["age_band"].(string); ok {
		ageBand = v
	}
	if raw, ok := rec.ParamsSnapshot["assumed_fields"].(map[string]any); ok {
		assumed = make(map[string]string, len(raw))
		for k, val := range raw {
			if s, ok := val.(string); ok {
				assumed[k] = s
			}
		}
	}
	return theme, style, ageBand, genre, assumed
}

func resolveDisplayTitle(theme, genre string, rec *storyRecord) string {
	if rec != nil && rec.ParamsSnapshot != nil {
		if v, ok := rec.ParamsSnapshot["story_title"].(string); ok {
			if t := strings.TrimSpace(v); t != "" && !looksLikeStoryOpening(t) {
				return t
			}
		}
	}
	theme = strings.TrimSpace(theme)
	if theme != "" {
		if strings.Contains(theme, "故事") {
			return theme
		}
		return theme + "的故事"
	}
	if rec != nil {
		stored := strings.TrimSpace(rec.Title)
		if stored != "" && !looksLikeStoryOpening(stored) && stored != "儿童故事" {
			return stored
		}
	}
	if genre != "" {
		return genre + "故事"
	}
	return "儿童故事"
}

func looksLikeStoryOpening(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}
	for _, prefix := range []string{
		"很久很久以前", "在很久很久以前", "从前", "有一天",
		"好的，我来", "让我来", "我来给你讲", "接下来",
	} {
		if strings.HasPrefix(s, prefix) {
			return true
		}
	}
	return utf8.RuneCountInString(s) > 20
}
