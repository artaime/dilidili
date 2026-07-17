package chat

import (
	"strings"
	"time"

	"github.com/spf13/viper"
)

const (
	defaultShortContextRecentLimit = 10
	defaultShortContextLoadLimit   = 20
	defaultShortContextRedisTTL    = 24 * time.Hour
)

func shortContextEnabled() bool {
	if !viper.IsSet("chat.short_context.enabled") {
		return true
	}
	return viper.GetBool("chat.short_context.enabled")
}

func shortContextRecentMessageLimit() int {
	if !viper.IsSet("chat.short_context.recent_message_limit") {
		return defaultShortContextRecentLimit
	}
	n := viper.GetInt("chat.short_context.recent_message_limit")
	if n <= 0 {
		return defaultShortContextRecentLimit
	}
	return n
}

func shortContextLoadLimit() int {
	if !viper.IsSet("chat.short_context.load_limit") {
		return defaultShortContextLoadLimit
	}
	n := viper.GetInt("chat.short_context.load_limit")
	if n <= 0 {
		return defaultShortContextLoadLimit
	}
	return n
}

func shortContextRedisEnabled() bool {
	if !viper.IsSet("chat.short_context.redis_enabled") {
		return true
	}
	return viper.GetBool("chat.short_context.redis_enabled")
}

func shortContextRedisTTL() time.Duration {
	if !viper.IsSet("chat.short_context.redis_ttl") {
		return defaultShortContextRedisTTL
	}
	d := viper.GetDuration("chat.short_context.redis_ttl")
	if d <= 0 {
		return defaultShortContextRedisTTL
	}
	return d
}

func shortContextReuseSessionOnFreshHello() bool {
	if !viper.IsSet("chat.short_context.reuse_session_on_fresh_hello") {
		return true
	}
	return viper.GetBool("chat.short_context.reuse_session_on_fresh_hello")
}

func hasValidShortContextIdentity(userID uint, deviceID, agentID string) bool {
	return userID > 0 && strings.TrimSpace(deviceID) != "" && strings.TrimSpace(agentID) != "" && strings.TrimSpace(agentID) != "0"
}
