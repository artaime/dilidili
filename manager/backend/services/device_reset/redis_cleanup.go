package device_reset

import (
	"context"
	"fmt"
	"strings"
	"time"

	"dili/manager/backend/config"

	"github.com/redis/go-redis/v9"
)

func purgeRedisDeviceKeys(ctx context.Context, cfg *config.Config, deviceSN string) error {
	deviceSN = strings.TrimSpace(deviceSN)
	if deviceSN == "" {
		return nil
	}
	client, prefix, err := getRedisClient(cfg)
	if err != nil {
		return err
	}
	if client == nil {
		return nil
	}

	keys := []string{
		fmt.Sprintf("%s:llm:%s", prefix, deviceSN),
		fmt.Sprintf("%s:llm:system:%s", prefix, deviceSN),
		fmt.Sprintf("%s:userconfig:%s", prefix, deviceSN),
		fmt.Sprintf("%s:story:%s:by_time", prefix, deviceSN),
		fmt.Sprintf("%s:story:%s:by_replay", prefix, deviceSN),
	}

	pattern := fmt.Sprintf("%s:story:%s:*", prefix, deviceSN)
	var cursor uint64
	for {
		batch, next, err := client.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return fmt.Errorf("scan story keys: %w", err)
		}
		keys = append(keys, batch...)
		cursor = next
		if cursor == 0 {
			break
		}
	}

	if len(keys) == 0 {
		return nil
	}
	if err := client.Del(ctx, keys...).Err(); err != nil {
		return fmt.Errorf("del redis keys: %w", err)
	}
	return nil
}

func getRedisClient(cfg *config.Config) (*redis.Client, string, error) {
	if cfg == nil || cfg.Redis == nil || strings.TrimSpace(cfg.Redis.Host) == "" {
		return nil, "", nil
	}

	rc := cfg.Redis
	port := rc.Port
	if port == 0 {
		port = 6379
	}
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", rc.Host, port),
		Password: rc.Password,
		DB:       rc.DB,
	})
	pingCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Ping(pingCtx).Err(); err != nil {
		return nil, "", fmt.Errorf("连接 Redis 失败: %w", err)
	}

	prefix := strings.TrimSpace(cfg.Redis.KeyPrefix)
	if prefix == "" {
		prefix = "dilidili"
	}
	return client, prefix, nil
}
