package story

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	log "dili-esp32-server-golang/logger"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
)

// StoreBackend 抽象存储，便于单测。
type StoreBackend interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value string) error
	Del(ctx context.Context, keys ...string) error
	ZAdd(ctx context.Context, key string, members ...redis.Z) error
	ZRevRange(ctx context.Context, key string, start, stop int64) ([]string, error)
	ZRem(ctx context.Context, key string, members ...string) error
	ZRevRangeByScore(ctx context.Context, key string, opt *redis.ZRangeBy) ([]string, error)
}

type redisBackend struct {
	client *redis.Client
}

func (r *redisBackend) Get(ctx context.Context, key string) (string, error) {
	return r.client.Get(ctx, key).Result()
}
func (r *redisBackend) Set(ctx context.Context, key string, value string) error {
	return r.client.Set(ctx, key, value, 0).Err()
}
func (r *redisBackend) Del(ctx context.Context, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}
	return r.client.Del(ctx, keys...).Err()
}
func (r *redisBackend) ZAdd(ctx context.Context, key string, members ...redis.Z) error {
	return r.client.ZAdd(ctx, key, members...).Err()
}
func (r *redisBackend) ZRevRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	return r.client.ZRevRange(ctx, key, start, stop).Result()
}
func (r *redisBackend) ZRem(ctx context.Context, key string, members ...string) error {
	any := make([]interface{}, len(members))
	for i, m := range members {
		any[i] = m
	}
	return r.client.ZRem(ctx, key, any...).Err()
}
func (r *redisBackend) ZRevRangeByScore(ctx context.Context, key string, opt *redis.ZRangeBy) ([]string, error) {
	return r.client.ZRevRangeByScore(ctx, key, opt).Result()
}

type memoryBackend struct {
	kv    map[string]string
	zsets map[string]map[string]float64
}

func newMemoryBackend() *memoryBackend {
	return &memoryBackend{
		kv:    make(map[string]string),
		zsets: make(map[string]map[string]float64),
	}
}

func (m *memoryBackend) Get(_ context.Context, key string) (string, error) {
	v, ok := m.kv[key]
	if !ok {
		return "", redis.Nil
	}
	return v, nil
}
func (m *memoryBackend) Set(_ context.Context, key, value string) error {
	m.kv[key] = value
	return nil
}
func (m *memoryBackend) Del(_ context.Context, keys ...string) error {
	for _, k := range keys {
		delete(m.kv, k)
		delete(m.zsets, k)
	}
	return nil
}
func (m *memoryBackend) ZAdd(_ context.Context, key string, members ...redis.Z) error {
	if m.zsets[key] == nil {
		m.zsets[key] = make(map[string]float64)
	}
	for _, z := range members {
		m.zsets[key][fmt.Sprint(z.Member)] = z.Score
	}
	return nil
}
func (m *memoryBackend) ZRevRange(_ context.Context, key string, start, stop int64) ([]string, error) {
	z := m.zsets[key]
	type pair struct {
		id    string
		score float64
	}
	var list []pair
	for id, score := range z {
		list = append(list, pair{id, score})
	}
	sort.Slice(list, func(i, j int) bool { return list[i].score > list[j].score })
	if start < 0 {
		start = 0
	}
	var out []string
	for i := start; i <= stop && int(i) < len(list); i++ {
		out = append(out, list[i].id)
	}
	return out, nil
}
func (m *memoryBackend) ZRem(_ context.Context, key string, members ...string) error {
	for _, mem := range members {
		delete(m.zsets[key], mem)
	}
	return nil
}
func (m *memoryBackend) ZRevRangeByScore(_ context.Context, key string, opt *redis.ZRangeBy) ([]string, error) {
	z := m.zsets[key]
	minS, maxS := parseScore(opt.Min), parseScore(opt.Max)
	type pair struct {
		id    string
		score float64
	}
	var list []pair
	for id, score := range z {
		if score >= minS && score <= maxS {
			list = append(list, pair{id, score})
		}
	}
	sort.Slice(list, func(i, j int) bool { return list[i].score > list[j].score })
	limit := int(opt.Count)
	if limit <= 0 {
		limit = len(list)
	}
	var out []string
	for i := 0; i < len(list) && i < limit; i++ {
		out = append(out, list[i].id)
	}
	return out, nil
}

func parseScore(s string) float64 {
	s = strings.TrimSpace(s)
	if s == "-inf" {
		return -1e18
	}
	if s == "+inf" {
		return 1e18
	}
	var f float64
	fmt.Sscanf(s, "%f", &f)
	return f
}

type Store struct {
	backend StoreBackend
	prefix  string
	cfg     Config
}

var defaultRedisClient *redis.Client

// SetDefaultRedisClient 主服务 Redis 初始化完成后注入，供 NewStore 使用。
func SetDefaultRedisClient(client *redis.Client) {
	defaultRedisClient = client
}

func NewStore(cfg Config) *Store {
	prefix := viper.GetString("redis.key_prefix")
	if prefix == "" {
		prefix = "dilidili"
	}
	var backend StoreBackend
	if defaultRedisClient != nil {
		backend = &redisBackend{client: defaultRedisClient}
	} else {
		log.Warn("Redis 不可用，Story Store 使用内存后端")
		backend = newMemoryBackend()
	}
	return &Store{backend: backend, prefix: prefix, cfg: cfg}
}

func NewStoreWithBackend(backend StoreBackend, prefix string, cfg Config) *Store {
	return &Store{backend: backend, prefix: prefix, cfg: cfg}
}

// NewRedisStore 使用外部 Redis 客户端创建 Store（供管理端等只读场景）。
func NewRedisStore(client *redis.Client, prefix string, cfg Config) *Store {
	if prefix == "" {
		prefix = "dilidili"
	}
	return NewStoreWithBackend(&redisBackend{client: client}, prefix, cfg)
}

func (s *Store) recordKey(deviceID, storyID string) string {
	return fmt.Sprintf("%s:story:%s:%s", s.prefix, deviceID, storyID)
}
func (s *Store) byTimeKey(deviceID string) string {
	return fmt.Sprintf("%s:story:%s:by_time", s.prefix, deviceID)
}
func (s *Store) byReplayKey(deviceID string) string {
	return fmt.Sprintf("%s:story:%s:by_replay", s.prefix, deviceID)
}

func (s *Store) Save(ctx context.Context, record *StoryRecord) error {
	if record == nil {
		return fmt.Errorf("record is nil")
	}
	if record.StoryID == "" {
		record.StoryID = uuid.NewString()
	}
	now := time.Now()
	if record.CreatedAt.IsZero() {
		record.CreatedAt = now
	}
	if record.LastPlayedAt.IsZero() {
		record.LastPlayedAt = now
	}
	data, err := json.Marshal(record)
	if err != nil {
		return err
	}
	key := s.recordKey(record.DeviceID, record.StoryID)
	if err := s.backend.Set(ctx, key, string(data)); err != nil {
		return err
	}
	ts := float64(record.LastPlayedAt.Unix())
	replayScore := float64(record.PlayCount)*1e9 + ts
	if err := s.backend.ZAdd(ctx, s.byTimeKey(record.DeviceID),
		redis.Z{Score: ts, Member: record.StoryID},
	); err != nil {
		return err
	}
	if err := s.backend.ZAdd(ctx, s.byReplayKey(record.DeviceID),
		redis.Z{Score: replayScore, Member: record.StoryID},
	); err != nil {
		return err
	}
	return s.evictExpired(ctx, record.DeviceID, now)
}

func (s *Store) Get(ctx context.Context, deviceID, storyID string) (*StoryRecord, error) {
	raw, err := s.backend.Get(ctx, s.recordKey(deviceID, storyID))
	if err != nil {
		return nil, err
	}
	var record StoryRecord
	if err := json.Unmarshal([]byte(raw), &record); err != nil {
		return nil, err
	}
	return &record, nil
}

func (s *Store) UpdateProgress(ctx context.Context, deviceID, storyID string, pos PlayPosition, status string, completed bool) error {
	record, err := s.Get(ctx, deviceID, storyID)
	if err != nil {
		return err
	}
	record.LastPosition = pos
	record.LastPlayStatus = status
	record.LastPlayedAt = time.Now()
	if completed {
		record.CompleteCount++
		record.LastPlayStatus = PlayStatusCompleted
	}
	return s.Save(ctx, record)
}

func (s *Store) RecordPlayStart(ctx context.Context, deviceID, storyID string) error {
	record, err := s.Get(ctx, deviceID, storyID)
	if err != nil {
		return err
	}
	record.PlayCount++
	record.LastPlayedAt = time.Now()
	record.LastPlayStatus = PlayStatusPlaying
	return s.Save(ctx, record)
}

// FindLatestByTheme 按主题查找最近一条故事；requireContent 为 true 时跳过无正文草稿。
func (s *Store) FindLatestByTheme(ctx context.Context, deviceID, theme string, requireContent bool) (*StoryRecord, error) {
	theme = NormalizeThemeKey(theme)
	if theme == "" {
		return nil, redis.Nil
	}
	ids, err := s.backend.ZRevRange(ctx, s.byTimeKey(deviceID), 0, 50)
	if err != nil {
		return nil, err
	}
	for _, id := range ids {
		rec, err := s.Get(ctx, deviceID, id)
		if err != nil {
			continue
		}
		if !ThemeMatchesRecord(theme, rec) {
			continue
		}
		if requireContent && !hasStoryContent(rec) {
			continue
		}
		return rec, nil
	}
	return nil, redis.Nil
}

func hasStoryContent(rec *StoryRecord) bool {
	if rec == nil {
		return false
	}
	if strings.TrimSpace(rec.FullText) != "" {
		return true
	}
	for _, seg := range rec.Segments {
		if strings.TrimSpace(seg) != "" {
			return true
		}
	}
	return false
}

func (s *Store) GetLast(ctx context.Context, deviceID string) (*StoryRecord, error) {
	ids, err := s.backend.ZRevRange(ctx, s.byTimeKey(deviceID), 0, 0)
	if err != nil || len(ids) == 0 {
		return nil, err
	}
	return s.Get(ctx, deviceID, ids[0])
}

func (s *Store) ListInWindow(ctx context.Context, deviceID string, start, end time.Time, limit int) ([]StoryRecord, error) {
	ids, err := s.backend.ZRevRangeByScore(ctx, s.byTimeKey(deviceID), &redis.ZRangeBy{
		Min:   fmt.Sprintf("%f", float64(start.Unix())),
		Max:   fmt.Sprintf("%f", float64(end.Unix())),
		Count: int64(limit),
	})
	if err != nil {
		return nil, err
	}
	return s.loadRecords(ctx, deviceID, ids)
}

func (s *Store) ListRecent(ctx context.Context, deviceID string, since time.Time, limit int) ([]StoryRecord, error) {
	ids, err := s.backend.ZRevRange(ctx, s.byTimeKey(deviceID), 0, int64(limit-1))
	if err != nil {
		return nil, err
	}
	var out []StoryRecord
	for _, id := range ids {
		rec, err := s.Get(ctx, deviceID, id)
		if err != nil {
			continue
		}
		if rec.CreatedAt.Before(since) && rec.LastPlayedAt.Before(since) {
			continue
		}
		out = append(out, *rec)
	}
	return out, nil
}

func (s *Store) GetLastInterrupted(ctx context.Context, deviceID string) (*StoryRecord, error) {
	ids, err := s.backend.ZRevRange(ctx, s.byTimeKey(deviceID), 0, 20)
	if err != nil {
		return nil, err
	}
	for _, id := range ids {
		rec, err := s.Get(ctx, deviceID, id)
		if err != nil {
			continue
		}
		if rec.LastPlayStatus == PlayStatusInterrupted {
			return rec, nil
		}
	}
	return nil, redis.Nil
}

func (s *Store) TopReplayThemes(ctx context.Context, deviceID string, topN int) []string {
	ids, err := s.backend.ZRevRange(ctx, s.byReplayKey(deviceID), 0, int64(topN-1))
	if err != nil {
		return nil
	}
	themeSet := map[string]struct{}{}
	var themes []string
	for _, id := range ids {
		rec, err := s.Get(ctx, deviceID, id)
		if err != nil {
			continue
		}
		if rec.ParamsSnapshot != nil {
			if t, ok := rec.ParamsSnapshot["theme"].(string); ok && t != "" {
				if _, exists := themeSet[t]; !exists {
					themeSet[t] = struct{}{}
					themes = append(themes, t)
				}
			}
		}
	}
	return themes
}

func (s *Store) loadRecords(ctx context.Context, deviceID string, ids []string) ([]StoryRecord, error) {
	var out []StoryRecord
	for _, id := range ids {
		rec, err := s.Get(ctx, deviceID, id)
		if err != nil {
			continue
		}
		out = append(out, *rec)
	}
	return out, nil
}

func (s *Store) evictExpired(ctx context.Context, deviceID string, now time.Time) error {
	ids, err := s.backend.ZRevRange(ctx, s.byTimeKey(deviceID), 0, 200)
	if err != nil {
		return err
	}
	for _, id := range ids {
		rec, err := s.Get(ctx, deviceID, id)
		if err != nil {
			continue
		}
		if ShouldEvict(*rec, now, s.cfg) {
			_ = s.backend.Del(ctx, s.recordKey(deviceID, id))
			_ = s.backend.ZRem(ctx, s.byTimeKey(deviceID), id)
			_ = s.backend.ZRem(ctx, s.byReplayKey(deviceID), id)
		}
	}
	return nil
}

func (s *Store) ShouldSuggestNewStory(record *StoryRecord) bool {
	if record == nil {
		return false
	}
	if record.PlayCount < s.cfg.ReplaySuggestThreshold {
		return false
	}
	interval := s.cfg.ReplaySuggestInterval
	if interval <= 0 {
		interval = 3
	}
	return record.PlayCount%interval == 0 && record.ReplaySuggestCount < record.PlayCount/interval
}

// DeleteAllForDevice 删除设备下全部故事记录及索引。
func (s *Store) DeleteAllForDevice(ctx context.Context, deviceID string) error {
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		return nil
	}

	ids, err := s.backend.ZRevRange(ctx, s.byTimeKey(deviceID), 0, 200)
	if err != nil && err != redis.Nil {
		return err
	}

	keys := []string{s.byTimeKey(deviceID), s.byReplayKey(deviceID)}
	for _, id := range ids {
		keys = append(keys, s.recordKey(deviceID, id))
	}
	if len(keys) == 0 {
		return nil
	}
	return s.backend.Del(ctx, keys...)
}

func (s *Store) MarkSuggestShown(ctx context.Context, deviceID, storyID string) error {
	rec, err := s.Get(ctx, deviceID, storyID)
	if err != nil {
		return err
	}
	rec.ReplaySuggestCount++
	return s.Save(ctx, rec)
}
