package controllers

import "strings"

func normalizeCosyVoiceModel(model string) string {
	model = strings.ToLower(strings.TrimSpace(model))
	switch {
	case model == "":
		return "cosyvoice-v3-flash"
	case strings.HasPrefix(model, "cosyvoice-v3.5-flash"):
		return "cosyvoice-v3-flash"
	case strings.HasPrefix(model, "cosyvoice-v3.5-plus"):
		return "cosyvoice-v3-plus"
	case strings.HasPrefix(model, "cosyvoice-v3-flash"):
		return "cosyvoice-v3-flash"
	case strings.HasPrefix(model, "cosyvoice-v3-plus"):
		return "cosyvoice-v3-plus"
	case strings.HasPrefix(model, "cosyvoice-v2"):
		return "cosyvoice-v2"
	case strings.HasPrefix(model, "cosyvoice-v1"):
		return "cosyvoice-v1"
	default:
		return model
	}
}

// GetAliyunCosyVoiceVoicesByModel 根据 CosyVoice 模型名称返回可选系统音色。
func GetAliyunCosyVoiceVoicesByModel(model string) []VoiceOption {
	key := normalizeCosyVoiceModel(model)
	if voices, ok := cosyVoiceModelVoiceMap[key]; ok && len(voices) > 0 {
		out := make([]VoiceOption, len(voices))
		copy(out, voices)
		return out
	}
	if voices, ok := cosyVoiceModelVoiceMap["cosyvoice-v3-flash"]; ok {
		out := make([]VoiceOption, len(voices))
		copy(out, voices)
		return out
	}
	return []VoiceOption{}
}
