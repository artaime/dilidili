package shortctx

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	i_redis "dili-esp32-server-golang/internal/db/redis"
	log "dili-esp32-server-golang/logger"

	"github.com/cloudwego/eino/schema"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
)

var (
	once     sync.Once
	instance *Store
)

// Store 设备短时对话滚动窗口（与 config_provider 解耦）。
type Store struct {
	redis     *redis.Client
	keyPrefix string
}

func Get() *Store {
	once.Do(func() {
		instance = &Store{
			redis:     i_redis.GetClient(),
			keyPrefix: strings.TrimSpace(viper.GetString("redis.key_prefix")),
		}
		if instance.keyPrefix == "" {
			instance.keyPrefix = "dili"
		}
	})
	return instance
}

func (s *Store) key(userID uint, deviceID, agentID string) string {
	return fmt.Sprintf("%s:shortctx:%d:%s:%s", s.keyPrefix, userID, deviceID, agentID)
}

// DeviceKeyPattern 出厂重置时按设备 SN 扫描 shortctx key。
func DeviceKeyPattern(prefix, deviceSN string) string {
	if strings.TrimSpace(prefix) == "" {
		prefix = "dili"
	}
	return fmt.Sprintf("%s:shortctx:*:%s:*", prefix, strings.TrimSpace(deviceSN))
}

// AddMessage 追加消息并截断到 limit，刷新 TTL。
func (s *Store) AddMessage(ctx context.Context, userID uint, deviceID, agentID string, msg schema.Message, limit int, ttl time.Duration) error {
	if s == nil || s.redis == nil {
		return nil
	}
	if userID == 0 || strings.TrimSpace(deviceID) == "" || strings.TrimSpace(agentID) == "" || agentID == "0" {
		return nil
	}
	if limit <= 0 {
		limit = 20
	}
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}

	payload, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal shortctx message: %w", err)
	}

	key := s.key(userID, deviceID, agentID)
	pipe := s.redis.Pipeline()
	pipe.RPush(ctx, key, string(payload))
	pipe.LTrim(ctx, key, int64(-limit), -1)
	pipe.Expire(ctx, key, ttl)
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("shortctx push: %w", err)
	}
	return nil
}

// GetMessages 返回旧→新顺序的最近消息（至多 limit 条）。
func (s *Store) GetMessages(ctx context.Context, userID uint, deviceID, agentID string, limit int) ([]*schema.Message, error) {
	if s == nil || s.redis == nil {
		return nil, nil
	}
	if userID == 0 || strings.TrimSpace(deviceID) == "" || strings.TrimSpace(agentID) == "" || agentID == "0" {
		return nil, nil
	}
	if limit <= 0 {
		limit = 20
	}

	key := s.key(userID, deviceID, agentID)
	raw, err := s.redis.LRange(ctx, key, int64(-limit), -1).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("shortctx lrange: %w", err)
	}

	out := make([]*schema.Message, 0, len(raw))
	for _, item := range raw {
		var msg schema.Message
		if err := json.Unmarshal([]byte(item), &msg); err != nil {
			log.Warnf("shortctx unmarshal skip: %v", err)
			continue
		}
		copied := msg
		out = append(out, &copied)
	}
	return out, nil
}

// DeleteForIdentity 删除指定三维键。
func (s *Store) DeleteForIdentity(ctx context.Context, userID uint, deviceID, agentID string) error {
	if s == nil || s.redis == nil {
		return nil
	}
	return s.redis.Del(ctx, s.key(userID, deviceID, agentID)).Err()
}
