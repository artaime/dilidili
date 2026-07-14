package configprovider

import "testing"

func TestNormalizeProviderInfersKnownProviderInsteadOfConfigID(t *testing.T) {
	tests := []struct {
		name       string
		configType string
		configID   string
		data       map[string]interface{}
		want       string
	}{
		{
			name:       "llm openai compatible proxy id",
			configType: "llm",
			configID:   "proxy_qwen",
			data: map[string]interface{}{
				"type":     "openai",
				"base_url": "https://api.onethingai.com/v1",
			},
			want: "openai",
		},
		{
			name:       "asr funasr custom id",
			configType: "asr",
			configID:   "local_asr",
			data: map[string]interface{}{
				"host": "127.0.0.1",
				"port": 10096,
			},
			want: "funasr",
		},
		{
			name:       "tts aliyun cosyvoice workspace without api_url",
			configType: "tts",
			configID:   "cosyvoice_bailian",
			data: map[string]interface{}{
				"api_key":      "sk-test",
				"workspace_id": "ws-123",
				"model":        "cosyvoice-v3-flash",
				"voice":        "longjielidou_v3",
			},
			want: "aliyun_cosyvoice",
		},
		{
			name:       "tts aliyun cosyvoice custom id",
			configType: "tts",
			configID:   "cosyvoice_bailian",
			data: map[string]interface{}{
				"api_url": "https://llm-test.cn-beijing.maas.aliyuncs.com/api/v1/services/audio/tts/SpeechSynthesizer",
				"model":   "cosyvoice-v3-flash",
				"voice":   "longjielidou_v3",
			},
			want: "aliyun_cosyvoice",
		},
		{
			name:       "tts aliyun qwen custom id",
			configType: "tts",
			configID:   "proxy_tts",
			data: map[string]interface{}{
				"api_url": "https://dashscope.aliyuncs.com/api/v1/services/aigc/multimodal-generation/generation",
				"model":   "qwen-tts",
			},
			want: "aliyun_qwen",
		},
		{
			name:       "asr tencent custom id",
			configType: "asr",
			configID:   "tencent_asr_default",
			data: map[string]interface{}{
				"provider":          "tencent_asr",
				"app_id":            1234567890,
				"secret_id":         "AKIDxxx",
				"engine_model_type": "16k_zh",
			},
			want: "tencent_asr",
		},
		{
			name:       "tts tencent custom id",
			configType: "tts",
			configID:   "tencent_tts_default",
			data: map[string]interface{}{
				"provider":   "tencent_tts",
				"secret_id":  "AKIDxxx",
				"voice_type": 101001,
			},
			want: "tencent_tts",
		},
		{
			configType: "vad",
			configID:   "custom_vad",
			data:       map[string]interface{}{},
			want:       "ten_vad",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NormalizeProvider(tt.configType, tt.configID, tt.data); got != tt.want {
				t.Fatalf("NormalizeProvider() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExportDataAddsNormalizedProviderAndLLMType(t *testing.T) {
	got := ExportData("llm", "proxy_qwen", "proxy_qwen", map[string]interface{}{
		"model_name": "qwen-plus",
	})

	if got["provider"] != "openai" {
		t.Fatalf("provider = %q, want openai", got["provider"])
	}
	if got["type"] != "openai" {
		t.Fatalf("type = %q, want openai", got["type"])
	}
}
