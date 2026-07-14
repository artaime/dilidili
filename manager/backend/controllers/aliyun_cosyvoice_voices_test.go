package controllers

import "testing"

func TestGetAliyunCosyVoiceVoicesByModel(t *testing.T) {
	tests := []struct {
		model string
		want  string
	}{
		{model: "", want: "longanyang"},
		{model: "cosyvoice-v3-flash", want: "longanyang"},
		{model: "cosyvoice-v3-flash-2025-01-01", want: "longanyang"},
		{model: "cosyvoice-v3-plus", want: "longanyang"},
		{model: "cosyvoice-v3.5-flash", want: "longanyang"},
		{model: "cosyvoice-v2", want: "longyingxiao"},
	}

	for _, tt := range tests {
		voices := GetAliyunCosyVoiceVoicesByModel(tt.model)
		if len(voices) == 0 {
			t.Fatalf("model=%q returned empty voices", tt.model)
		}
		if voices[0].Value != tt.want {
			t.Fatalf("model=%q first voice=%q, want %q", tt.model, voices[0].Value, tt.want)
		}
	}
}

func TestNormalizeCosyVoiceModel(t *testing.T) {
	if got := normalizeCosyVoiceModel("cosyvoice-v3.5-plus"); got != "cosyvoice-v3-plus" {
		t.Fatalf("normalizeCosyVoiceModel() = %q, want cosyvoice-v3-plus", got)
	}
}
