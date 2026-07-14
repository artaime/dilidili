package storypersist

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	httpc "dili-esp32-server-golang/internal/components/http"
	"dili-esp32-server-golang/internal/domain/story"
	"dili-esp32-server-golang/internal/util"

	"github.com/spf13/viper"
)

// Client 主服务 → Manager 故事持久化客户端。
type Client struct {
	client  *httpc.ManagerClient
	enabled bool
}

func NewFromViper() *Client {
	base := util.GetBackendURL()
	token := util.GetManagerAuthToken()
	enabled := base != "" && viper.GetString("config_provider.type") == "manager"
	timeout := viper.GetDuration("manager.history_timeout")
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	return &Client{
		client: httpc.NewManagerClient(httpc.ManagerClientConfig{
			BaseURL:    base,
			AuthToken:  token,
			Timeout:    timeout,
			MaxRetries: 2,
		}),
		enabled: enabled,
	}
}

func (c *Client) Enabled() bool {
	return c != nil && c.enabled && c.client != nil
}

type assetReq struct {
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
	ParamsSnapshot     map[string]any `json:"params_snapshot"`
}

type playbackReq struct {
	DeviceSN          string    `json:"device_sn"`
	StoryID           string    `json:"story_id"`
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

type assetView struct {
	StoryID            string         `json:"story_id"`
	Title              string         `json:"title"`
	ThemeKey           string         `json:"theme_key"`
	PoolKind           string         `json:"pool_kind"`
	CanonicalKey       string         `json:"canonical_key"`
	Mode               string         `json:"mode"`
	NarrationMode      string         `json:"narration_mode"`
	AgeBand            string         `json:"age_band"`
	FullText           string         `json:"full_text"`
	Segments           []string       `json:"segments"`
	GenerationComplete bool           `json:"generation_complete"`
	Shareable          bool           `json:"shareable"`
	ParamsSnapshot     map[string]any `json:"params_snapshot"`
}

// FindShareableParams 共享池查询参数。
type FindShareableParams struct {
	PoolKind    string
	Theme       string
	ThemeRaw    string
	AgeBand     string
	DeviceSN    string
	ExcludeDays int
	TopK        int
}

// UpsertFromRecord 将 StoryRecord dual-write 到 MySQL asset + playback。
func (c *Client) UpsertFromRecord(ctx context.Context, rec *story.StoryRecord) error {
	if !c.Enabled() || rec == nil || rec.StoryID == "" {
		return nil
	}
	theme := ""
	narration := ""
	if rec.ParamsSnapshot != nil {
		if v, ok := rec.ParamsSnapshot["theme"].(string); ok {
			theme = story.NormalizeThemeKey(v)
		}
		if v, ok := rec.ParamsSnapshot["narration_mode"].(string); ok {
			narration = v
		}
	}
	if narration == "" && story.ShouldTellCanonical(story.StoryParams{
		RequestType:   rec.Mode,
		NarrationMode: "",
		Theme:         theme,
	}) {
		narration = story.NarrationCanonical
	}
	pool, canonical, aliases := story.ShareEnrollmentFromRecord(rec)
	if err := c.client.DoRequest(ctx, httpc.RequestOptions{
		Method: "POST",
		Path:   "/api/internal/stories/assets",
		Body: assetReq{
			StoryID:            rec.StoryID,
			Title:              rec.Title,
			ThemeKey:           theme,
			PoolKind:           pool,
			CanonicalKey:       canonical,
			Aliases:            aliases,
			Mode:               rec.Mode,
			NarrationMode:      narration,
			AgeBand:            rec.AgeBand,
			FullText:           rec.FullText,
			Segments:           rec.Segments,
			GenerationComplete: story.IsGenerationComplete(rec),
			CreatorDeviceSN:    rec.DeviceID,
			ParamsSnapshot:     rec.ParamsSnapshot,
		},
	}); err != nil {
		return fmt.Errorf("upsert asset: %w", err)
	}
	return c.client.DoRequest(ctx, httpc.RequestOptions{
		Method: "POST",
		Path:   "/api/internal/stories/playbacks",
		Body: playbackReq{
			DeviceSN:          rec.DeviceID,
			StoryID:           rec.StoryID,
			AgentID:           rec.AgentID,
			LastPlayStatus:    rec.LastPlayStatus,
			CharOffset:        rec.LastPosition.CharOffset,
			SegmentIndex:      rec.LastPosition.SegmentIndex,
			LastSentenceIndex: rec.LastPosition.LastSentenceIndex,
			LastSentence:      rec.LastPosition.LastSentence,
			PlayCount:         rec.PlayCount,
			CompleteCount:     rec.CompleteCount,
			LastPlayedAt:      rec.LastPlayedAt,
		},
	})
}

// FindShareable 按池查询可复用正文。
func (c *Client) FindShareable(ctx context.Context, p FindShareableParams) (*story.StoryRecord, error) {
	if !c.Enabled() {
		return nil, fmt.Errorf("persist client disabled")
	}
	pool := strings.TrimSpace(p.PoolKind)
	if pool == "" {
		return nil, fmt.Errorf("empty pool")
	}
	themeParam := story.NormalizeThemeKey(p.Theme)
	if pool == story.SharePoolNamed {
		keys := story.BuildShareLookupKeys(p.Theme, p.ThemeRaw)
		themeParam = strings.Join(keys, ",")
		if themeParam == "" {
			return nil, fmt.Errorf("empty theme")
		}
	}
	q := map[string]string{
		"pool_kind": pool,
		"theme":     themeParam,
		"age_band":  strings.TrimSpace(p.AgeBand),
		"device_sn": strings.TrimSpace(p.DeviceSN),
	}
	if p.ExcludeDays > 0 {
		q["exclude_days"] = strconv.Itoa(p.ExcludeDays)
	}
	if p.TopK > 0 {
		q["top_k"] = strconv.Itoa(p.TopK)
	}
	raw, err := c.client.DoRequestRaw(ctx, httpc.RequestOptions{
		Method:      "GET",
		Path:        "/api/internal/stories/shareable",
		QueryParams: q,
	})
	if err != nil {
		return nil, err
	}
	var resp struct {
		Data assetView `json:"data"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, err
	}
	if resp.Data.StoryID == "" || resp.Data.FullText == "" {
		return nil, fmt.Errorf("empty shareable")
	}
	params := resp.Data.ParamsSnapshot
	if params == nil {
		params = map[string]any{}
	}
	if resp.Data.ThemeKey != "" {
		params["theme"] = resp.Data.ThemeKey
	}
	if resp.Data.NarrationMode != "" {
		params["narration_mode"] = resp.Data.NarrationMode
	}
	params["generation_complete"] = true
	if resp.Data.PoolKind != "" {
		params[story.SnapshotKeyPoolKind] = resp.Data.PoolKind
	}
	if resp.Data.CanonicalKey != "" {
		params[story.SnapshotKeyCanonicalKey] = resp.Data.CanonicalKey
	}
	return &story.StoryRecord{
		StoryID:            resp.Data.StoryID,
		Title:              resp.Data.Title,
		FullText:           resp.Data.FullText,
		Segments:           resp.Data.Segments,
		Mode:               resp.Data.Mode,
		AgeBand:            resp.Data.AgeBand,
		GenerationComplete: true,
		ParamsSnapshot:     params,
		LastPlayStatus:     story.PlayStatusPlaying,
	}, nil
}
