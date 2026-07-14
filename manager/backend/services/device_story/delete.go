package device_story

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"dili/manager/backend/services/story_persist"

	"github.com/redis/go-redis/v9"
)

// DeleteResult 设备故事删除结果（不含共享资产）。
type DeleteResult struct {
	DeviceSN      string `json:"device_sn"`
	StoryID       string `json:"story_id,omitempty"`
	PlaybackDeleted int64  `json:"playback_deleted"`
	RedisDeleted  int    `json:"redis_deleted"`
	RedisSkipped  bool   `json:"redis_skipped,omitempty"`
	RedisError    string `json:"redis_error,omitempty"`
}

// DeleteDeviceStory 删除单条：MySQL playback + Redis 设备键；不删 story_assets。
func (s *Service) DeleteDeviceStory(ctx context.Context, deviceID uint, storyID string) (*DeleteResult, error) {
	device, err := s.loadDevice(deviceID)
	if err != nil {
		return nil, err
	}
	storyID = strings.TrimSpace(storyID)
	if storyID == "" {
		return nil, ErrStoryNotFound
	}
	sn := device.DeviceName
	result := &DeleteResult{DeviceSN: sn, StoryID: storyID}

	persist := story_persist.NewService(s.DB)
	n, err := persist.DeletePlayback(ctx, sn, storyID)
	if err != nil {
		return nil, err
	}
	result.PlaybackDeleted = n

	redisN, redisErr := s.deleteRedisStory(ctx, sn, storyID)
	result.RedisDeleted = redisN
	if redisErr != nil {
		if errors.Is(redisErr, ErrRedisNotConfigured) {
			result.RedisSkipped = true
		} else {
			result.RedisError = redisErr.Error()
		}
	}

	if result.PlaybackDeleted == 0 && result.RedisDeleted == 0 && !result.RedisSkipped {
		// Redis 配置缺失时只要 MySQL 也无记录才算不存在
		return nil, ErrStoryNotFound
	}
	if result.PlaybackDeleted == 0 && result.RedisDeleted == 0 && result.RedisSkipped {
		return nil, ErrStoryNotFound
	}
	return result, nil
}

// ClearDeviceStories 清空设备全部故事记录（playback + Redis 索引/正文缓存）。
func (s *Service) ClearDeviceStories(ctx context.Context, deviceID uint) (*DeleteResult, error) {
	device, err := s.loadDevice(deviceID)
	if err != nil {
		return nil, err
	}
	sn := device.DeviceName
	result := &DeleteResult{DeviceSN: sn}

	persist := story_persist.NewService(s.DB)
	n, err := persist.DeletePlaybacksByDevice(ctx, sn)
	if err != nil {
		return nil, err
	}
	result.PlaybackDeleted = n

	redisN, redisErr := s.clearRedisStories(ctx, sn)
	result.RedisDeleted = redisN
	if redisErr != nil {
		if errors.Is(redisErr, ErrRedisNotConfigured) {
			result.RedisSkipped = true
		} else {
			result.RedisError = redisErr.Error()
		}
	}
	return result, nil
}

func (s *Service) deleteRedisStory(ctx context.Context, deviceSN, storyID string) (int, error) {
	reader, err := s.storyReader()
	if err != nil {
		return 0, err
	}
	return reader.deleteOne(ctx, deviceSN, storyID)
}

func (s *Service) clearRedisStories(ctx context.Context, deviceSN string) (int, error) {
	reader, err := s.storyReader()
	if err != nil {
		return 0, err
	}
	return reader.deleteAll(ctx, deviceSN)
}

func (r *redisStoryReader) byReplayKey(deviceID string) string {
	return fmt.Sprintf("%s:story:%s:by_replay", r.prefix, deviceID)
}

func (r *redisStoryReader) deleteOne(ctx context.Context, deviceID, storyID string) (int, error) {
	if r == nil || r.client == nil {
		return 0, ErrRedisNotConfigured
	}
	deleted := 0
	key := r.recordKey(deviceID, storyID)
	n, err := r.client.Del(ctx, key).Result()
	if err != nil {
		return 0, err
	}
	deleted += int(n)
	_ = r.client.ZRem(ctx, r.byTimeKey(deviceID), storyID).Err()
	_ = r.client.ZRem(ctx, r.byReplayKey(deviceID), storyID).Err()
	return deleted, nil
}

func (r *redisStoryReader) deleteAll(ctx context.Context, deviceID string) (int, error) {
	if r == nil || r.client == nil {
		return 0, ErrRedisNotConfigured
	}
	ids, err := r.client.ZRange(ctx, r.byTimeKey(deviceID), 0, -1).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		return 0, err
	}
	keys := []string{r.byTimeKey(deviceID), r.byReplayKey(deviceID)}
	for _, id := range ids {
		keys = append(keys, r.recordKey(deviceID, id))
	}
	if len(keys) == 0 {
		return 0, nil
	}
	n, err := r.client.Del(ctx, keys...).Result()
	if err != nil {
		return 0, err
	}
	return int(n), nil
}
