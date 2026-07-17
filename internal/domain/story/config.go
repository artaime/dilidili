package story

import (
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Enabled                  bool
	MinRetentionDays         int
	MaxRetentionDays         int
	LastNightStartHour       int
	LastNightEndHour         int
	ReplaySuggestThreshold   int
	ReplaySuggestInterval    int
	MemoryHintMinConfidence  float64
	StreamEnabled            bool
	FillerEnabled            bool
	FillerDefault            string
	FillerBedtime            string
	ProtectContinueThreshold int // 已播字数≥此值时停播不停写
	ShareExcludeDays         int // 本设备近期排斥天数
	SharePickTopK            int // 共享池 Top-K 随机
	FollowupEnabled          bool
	FollowupTTLMinutes       int
	FollowupMaxRunes         int
	FollowupClarifyMaxRounds int
}

func LoadConfig() Config {
	cfg := Config{
		Enabled:                 true,
		MinRetentionDays:        7,
		MaxRetentionDays:        90,
		LastNightStartHour:      18,
		LastNightEndHour:        7,
		ReplaySuggestThreshold:  5,
		ReplaySuggestInterval:   3,
		MemoryHintMinConfidence: 0.75,
		StreamEnabled:              true,
		FillerEnabled:              true,
		FillerDefault:              "好呀，我给你讲一个故事。",
		FillerBedtime:              "好呀，那我们讲个睡前故事，乖乖听哦。",
		ProtectContinueThreshold: DefaultProtectContinueThreshold,
		ShareExcludeDays:         DefaultShareExcludeDays,
		SharePickTopK:            DefaultSharePickTopK,
		FollowupEnabled:          true,
		FollowupTTLMinutes:       30,
		FollowupMaxRunes:         3000,
		FollowupClarifyMaxRounds: 2,
	}
	if viper.IsSet("story.enabled") {
		cfg.Enabled = viper.GetBool("story.enabled")
	}
	if viper.IsSet("story.min_retention_days") {
		cfg.MinRetentionDays = viper.GetInt("story.min_retention_days")
	}
	if viper.IsSet("story.max_retention_days") {
		cfg.MaxRetentionDays = viper.GetInt("story.max_retention_days")
	}
	if viper.IsSet("story.last_night_start_hour") {
		cfg.LastNightStartHour = viper.GetInt("story.last_night_start_hour")
	}
	if viper.IsSet("story.last_night_end_hour") {
		cfg.LastNightEndHour = viper.GetInt("story.last_night_end_hour")
	}
	if viper.IsSet("story.replay_suggest_threshold") {
		cfg.ReplaySuggestThreshold = viper.GetInt("story.replay_suggest_threshold")
	}
	if viper.IsSet("story.replay_suggest_interval") {
		cfg.ReplaySuggestInterval = viper.GetInt("story.replay_suggest_interval")
	}
	if viper.IsSet("story.memory_hint_min_confidence") {
		cfg.MemoryHintMinConfidence = viper.GetFloat64("story.memory_hint_min_confidence")
	}
	if viper.IsSet("story.stream_enabled") {
		cfg.StreamEnabled = viper.GetBool("story.stream_enabled")
	}
	if viper.IsSet("story.filler_enabled") {
		cfg.FillerEnabled = viper.GetBool("story.filler_enabled")
	}
	if v := strings.TrimSpace(viper.GetString("story.filler_default")); v != "" {
		cfg.FillerDefault = v
	}
	if v := strings.TrimSpace(viper.GetString("story.filler_bedtime")); v != "" {
		cfg.FillerBedtime = v
	}
	if viper.IsSet("story.protect_continue_threshold") {
		cfg.ProtectContinueThreshold = viper.GetInt("story.protect_continue_threshold")
	}
	if cfg.ProtectContinueThreshold <= 0 {
		cfg.ProtectContinueThreshold = DefaultProtectContinueThreshold
	}
	if viper.IsSet("story.share_exclude_days") {
		cfg.ShareExcludeDays = viper.GetInt("story.share_exclude_days")
	}
	if cfg.ShareExcludeDays <= 0 {
		cfg.ShareExcludeDays = DefaultShareExcludeDays
	}
	if viper.IsSet("story.share_pick_top_k") {
		cfg.SharePickTopK = viper.GetInt("story.share_pick_top_k")
	}
	if cfg.SharePickTopK <= 0 {
		cfg.SharePickTopK = DefaultSharePickTopK
	}
	if viper.IsSet("story.followup_enabled") {
		cfg.FollowupEnabled = viper.GetBool("story.followup_enabled")
	}
	if viper.IsSet("story.followup_ttl_minutes") {
		cfg.FollowupTTLMinutes = viper.GetInt("story.followup_ttl_minutes")
	}
	if cfg.FollowupTTLMinutes <= 0 {
		cfg.FollowupTTLMinutes = 30
	}
	if viper.IsSet("story.followup_max_runes") {
		cfg.FollowupMaxRunes = viper.GetInt("story.followup_max_runes")
	}
	if cfg.FollowupMaxRunes <= 0 {
		cfg.FollowupMaxRunes = 3000
	}
	if viper.IsSet("story.followup_clarify_max_rounds") {
		cfg.FollowupClarifyMaxRounds = viper.GetInt("story.followup_clarify_max_rounds")
	}
	if cfg.FollowupClarifyMaxRounds <= 0 {
		cfg.FollowupClarifyMaxRounds = 2
	}
	return cfg
}

func LastNightWindow(now time.Time, startHour, endHour int) (time.Time, time.Time) {
	loc := now.Location()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	yesterday := today.Add(-24 * time.Hour)

	windowStart := time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), startHour, 0, 0, 0, loc)
	windowEnd := time.Date(now.Year(), now.Month(), now.Day(), endHour, 0, 0, 0, loc)
	if now.Hour() < endHour {
		// 仍在「今日凌晨」段，窗口结束为今天 endHour
	} else if now.Before(windowStart) {
		// 白天：昨晚窗口为「再往前一天 18:00 ~ 今天 7:00」
		windowStart = windowStart.Add(-24 * time.Hour)
	}
	return windowStart, windowEnd
}

func InTimeWindow(t, start, end time.Time) bool {
	return !t.Before(start) && t.Before(end)
}
