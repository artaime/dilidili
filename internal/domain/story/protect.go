package story

// DefaultProtectContinueThreshold 已播字数达到该阈值后，插话停播但不取消 LLM 续写。
const DefaultProtectContinueThreshold = 300

// ShouldCancelGenerationOnInterrupt 根据已听故事字数判断是否应取消 LLM 生成。
// heardRunes 为已成功送入 TTS 的故事正文 rune 数（不含 filler）；threshold≤0 时使用默认值。
func ShouldCancelGenerationOnInterrupt(heardRunes, threshold int) bool {
	if threshold <= 0 {
		threshold = DefaultProtectContinueThreshold
	}
	return heardRunes < threshold
}

// ProtectContinueThreshold 解析配置阈值。
func ProtectContinueThreshold(cfg Config) int {
	if cfg.ProtectContinueThreshold <= 0 {
		return DefaultProtectContinueThreshold
	}
	return cfg.ProtectContinueThreshold
}
