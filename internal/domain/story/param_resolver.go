package story

import (
	"regexp"
	"strconv"
	"strings"
	"time"
)

var agePattern = regexp.MustCompile(`(\d{1,2})\s*岁`)

func ResolveParams(params StoryParams, memoryContext string, cfg Config, now time.Time) ResolvedParams {
	out := params
	assumed := map[string]string{}
	minConf := cfg.MemoryHintMinConfidence

	// 显式 age_years → age_band
	if out.AgeYears != nil && out.AgeBand == "" {
		out.AgeBand = AgeBandFromYears(*out.AgeYears)
	}

	// memory hints 高置信
	for _, hint := range out.MemoryHints {
		if hint.Confidence < minConf {
			continue
		}
		switch hint.Field {
		case "age", "age_years":
			if out.AgeYears == nil {
				if y, err := strconv.Atoi(strings.TrimSpace(hint.Value)); err == nil && y > 0 && y < 20 {
					out.AgeYears = &y
					out.AgeBand = AgeBandFromYears(y)
				}
			}
		case "age_band":
			if out.AgeBand == "" {
				out.AgeBand = strings.TrimSpace(hint.Value)
			}
		case "interest", "interests":
			if len(out.Interests) == 0 && strings.TrimSpace(hint.Value) != "" {
				out.Interests = []string{strings.TrimSpace(hint.Value)}
			}
		}
	}

	// 从 memoryContext 文本解析年龄（仅明确「N岁」模式）
	if out.AgeBand == "" && memoryContext != "" {
		if m := agePattern.FindStringSubmatch(memoryContext); len(m) >= 2 {
			if y, err := strconv.Atoi(m[1]); err == nil && y >= 2 && y <= 18 {
				out.AgeYears = &y
				out.AgeBand = AgeBandFromYears(y)
				assumed["age_band"] = "memory_context"
			}
		}
	}

	// 时间弱推断 bedtime（不当作年龄）
	if out.IsBedtime == nil {
		h := now.Hour()
		if h >= 20 || h < 7 {
			bedtime := true
			out.IsBedtime = &bedtime
			assumed["is_bedtime"] = "inferred_time"
		}
	}

	var missing []string
	if out.AgeBand == "" {
		// 语音场景优先开讲：无明确年龄时使用适龄默认档，不阻断生成
		out.AgeBand = "primary_low"
		assumed["age_band"] = "default_voice"
		if out.UserSaidCasual {
			assumed["age_band"] = "default_casual"
		}
	}

	return ResolvedParams{
		Params:        out,
		Missing:       missing,
		AssumedFields: assumed,
	}
}
