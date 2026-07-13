package asr

import (
	"testing"

	"dili-esp32-server-golang/constants"
)

func TestNewAsrProviderTencent(t *testing.T) {
	cfg := map[string]interface{}{
		"provider":          constants.AsrTypeTencent,
		"app_id":            float64(1234567890),
		"secret_id":         "AKIDtest",
		"secret_key":        "testkey",
		"engine_model_type": "16k_zh",
	}
	provider, err := NewAsrProvider(constants.AsrTypeTencent, cfg)
	if err != nil {
		t.Fatalf("NewAsrProvider tencent_asr failed: %v", err)
	}
	if provider == nil {
		t.Fatal("expected non-nil provider")
	}
	if !provider.IsValid() {
		t.Fatal("expected valid provider before Close")
	}
	if err := provider.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}
}
