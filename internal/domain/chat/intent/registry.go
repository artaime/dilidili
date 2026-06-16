package intent

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/viper"
)

func MinConfidence() float64 {
	if viper.IsSet("chat.intent_router.min_confidence") {
		return viper.GetFloat64("chat.intent_router.min_confidence")
	}
	return 0.70
}

func IntentRouterEnabled() bool {
	if !viper.IsSet("chat.intent_router.enabled") {
		return true
	}
	return viper.GetBool("chat.intent_router.enabled")
}

func ParseRouterResponse(raw string) (RouterResponse, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return RouterResponse{}, fmt.Errorf("empty router response")
	}
	candidate := extractJSONObject(raw)
	var resp RouterResponse
	if err := json.Unmarshal([]byte(candidate), &resp); err != nil {
		return RouterResponse{}, err
	}
	resp.Intent = strings.TrimSpace(strings.ToLower(resp.Intent))
	if resp.Intent == "" {
		return RouterResponse{}, fmt.Errorf("missing intent")
	}
	return resp, nil
}

func extractJSONObject(raw string) string {
	start := strings.Index(raw, "{")
	end := strings.LastIndex(raw, "}")
	if start >= 0 && end > start {
		return raw[start : end+1]
	}
	return raw
}

func ParseConfidence(raw string) (float64, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, fmt.Errorf("empty confidence")
	}
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0, err
	}
	if value < 0 {
		value = 0
	}
	if value > 1 {
		value = 1
	}
	return value, nil
}

func ParseData[T any](data json.RawMessage) (T, error) {
	var out T
	if len(data) == 0 || string(data) == "null" {
		return out, nil
	}
	err := json.Unmarshal(data, &out)
	return out, err
}
