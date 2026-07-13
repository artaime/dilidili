package tencent

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

const (
	defaultEngineModelType = "16k_zh"
	defaultVoiceFormat     = 1
	defaultSampleRate      = 16000
	defaultConnectTimeout  = 10 * time.Second
)

// Config 腾讯云实时语音识别 WebSocket v2 配置。
type Config struct {
	AppID            int64
	SecretID         string
	SecretKey        string
	EngineModelType  string
	VoiceFormat      int
	SampleRate       int
	InputSampleRate  int
	NeedVAD          int
	FilterDirty      int
	FilterModal      int
	FilterPunc       int
	ConvertNumMode   int
	VadSilenceTime   int
	HotwordID        string
	CustomizationID  string
	ConnectTimeout   time.Duration
}

func defaultConfig() Config {
	return Config{
		EngineModelType: defaultEngineModelType,
		VoiceFormat:     defaultVoiceFormat,
		SampleRate:      defaultSampleRate,
		NeedVAD:         0,
		FilterDirty:     0,
		FilterModal:     0,
		FilterPunc:      0,
		ConvertNumMode:  1,
		ConnectTimeout:  defaultConnectTimeout,
	}
}

// ConfigFromMap 从配置 map 解析腾讯 ASR 配置。
func ConfigFromMap(config map[string]interface{}) (Config, error) {
	cfg := defaultConfig()
	if config == nil {
		return cfg, fmt.Errorf("tencent_asr config is nil")
	}

	cfg.AppID = parseInt64(config, "app_id")
	cfg.SecretID = strings.TrimSpace(stringValue(config, "secret_id"))
	cfg.SecretKey = strings.TrimSpace(stringValue(config, "secret_key"))
	if engine := strings.TrimSpace(stringValue(config, "engine_model_type")); engine != "" {
		cfg.EngineModelType = engine
	}
	if voiceFormat := parseInt(config, "voice_format", defaultVoiceFormat); voiceFormat > 0 {
		cfg.VoiceFormat = voiceFormat
	}
	if sampleRate := parseInt(config, "sample_rate", defaultSampleRate); sampleRate > 0 {
		cfg.SampleRate = sampleRate
	}
	cfg.InputSampleRate = parseInt(config, "input_sample_rate", 0)
	cfg.NeedVAD = parseInt(config, "needvad", cfg.NeedVAD)
	cfg.FilterDirty = parseInt(config, "filter_dirty", cfg.FilterDirty)
	cfg.FilterModal = parseInt(config, "filter_modal", cfg.FilterModal)
	cfg.FilterPunc = parseInt(config, "filter_punc", cfg.FilterPunc)
	cfg.ConvertNumMode = parseInt(config, "convert_num_mode", cfg.ConvertNumMode)
	cfg.VadSilenceTime = parseInt(config, "vad_silence_time", 0)
	cfg.HotwordID = strings.TrimSpace(stringValue(config, "hotword_id"))
	cfg.CustomizationID = strings.TrimSpace(stringValue(config, "customization_id"))
	if timeout := parseInt(config, "timeout", 30); timeout > 0 {
		cfg.ConnectTimeout = time.Duration(timeout) * time.Second
	}

	if cfg.AppID <= 0 {
		return cfg, fmt.Errorf("tencent_asr missing app_id")
	}
	if cfg.SecretID == "" {
		return cfg, fmt.Errorf("tencent_asr missing secret_id")
	}
	if cfg.SecretKey == "" {
		return cfg, fmt.Errorf("tencent_asr missing secret_key")
	}
	if cfg.EngineModelType == "" {
		return cfg, fmt.Errorf("tencent_asr missing engine_model_type")
	}
	switch cfg.SampleRate {
	case 8000, 16000:
	default:
		return cfg, fmt.Errorf("tencent_asr unsupported sample_rate: %d", cfg.SampleRate)
	}
	return cfg, nil
}

func stringValue(config map[string]interface{}, key string) string {
	value, ok := config[key]
	if !ok || value == nil {
		return ""
	}
	switch v := value.(type) {
	case string:
		return v
	default:
		return fmt.Sprint(v)
	}
}

func parseInt(config map[string]interface{}, key string, defaultValue int) int {
	value, ok := config[key]
	if !ok || value == nil {
		return defaultValue
	}
	switch v := value.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case string:
		if parsed, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
			return parsed
		}
	}
	return defaultValue
}

func parseInt64(config map[string]interface{}, key string) int64 {
	value, ok := config[key]
	if !ok || value == nil {
		return 0
	}
	switch v := value.(type) {
	case int:
		return int64(v)
	case int64:
		return v
	case float64:
		return int64(v)
	case string:
		if parsed, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64); err == nil {
			return parsed
		}
	}
	return 0
}
