package device_reset

import (
	"context"
	"errors"
	"fmt"
	"log"

	"dili/manager/backend/config"
	"dili/manager/backend/models"
	"dili/manager/backend/services/device_memory"

	"gorm.io/gorm"
)

var (
	ErrDeviceNotFound = errors.New("设备不存在")
	ErrForbidden      = errors.New("无权操作该设备")
	ErrMissingSN      = errors.New("设备缺少 SN")
)

// SessionNotifier 通知主服务重置设备会话（best-effort）。
type SessionNotifier interface {
	NotifyDeviceReset(ctx context.Context, deviceSN string)
}

type Service struct {
	DB       *gorm.DB
	Cfg      *config.Config
	Notifier SessionNotifier
	Memory   *device_memory.Service
}

func NewService(db *gorm.DB, cfg *config.Config, notifier SessionNotifier) *Service {
	return &Service{
		DB:       db,
		Cfg:      cfg,
		Notifier: notifier,
		Memory:   device_memory.NewService(db),
	}
}

// ResetToFactory 解绑设备：清理全部数据并恢复出厂登记字段（小程序，校验设备归属）。
func (s *Service) ResetToFactory(ctx context.Context, deviceDBID uint, actorUserID uint) error {
	device, err := s.loadOwnedDevice(deviceDBID, actorUserID)
	if err != nil {
		return err
	}
	return s.executeReset(ctx, device)
}

// ResetToFactoryByAdmin 管理员出厂重置：不校验 user_id 归属。
func (s *Service) ResetToFactoryByAdmin(ctx context.Context, deviceDBID uint) error {
	device, err := s.loadDevice(deviceDBID)
	if err != nil {
		return err
	}
	return s.executeReset(ctx, device)
}

func (s *Service) executeReset(ctx context.Context, device *models.Device) error {
	sn := device.DeviceName

	if s.Notifier != nil {
		s.Notifier.NotifyDeviceReset(ctx, sn)
	}

	if err := s.Memory.PurgeByDeviceSN(ctx, sn); err != nil {
		return fmt.Errorf("清理 Memobase 记忆失败: %w", err)
	}

	if err := purgeRedisDeviceKeys(ctx, s.Cfg, sn); err != nil {
		return fmt.Errorf("清理 Redis 设备数据失败: %w", err)
	}

	if err := purgeDeviceDatabaseRecords(s.DB, s.Cfg, device); err != nil {
		return fmt.Errorf("清理数据库记录失败: %w", err)
	}

	if err := resetDeviceRow(s.DB, device.ID); err != nil {
		return fmt.Errorf("重置设备出厂状态失败: %w", err)
	}

	log.Printf("设备解绑出厂重置完成: id=%d sn=%s", device.ID, sn)
	return nil
}

func (s *Service) loadDevice(deviceDBID uint) (*models.Device, error) {
	var device models.Device
	if err := s.DB.First(&device, deviceDBID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrDeviceNotFound
		}
		return nil, err
	}
	if device.DeviceName == "" {
		return nil, ErrMissingSN
	}
	return &device, nil
}

func (s *Service) loadOwnedDevice(deviceDBID uint, actorUserID uint) (*models.Device, error) {
	device, err := s.loadDevice(deviceDBID)
	if err != nil {
		return nil, err
	}
	if device.UserID != actorUserID {
		return nil, ErrForbidden
	}
	return device, nil
}

func resetDeviceRow(db *gorm.DB, deviceID uint) error {
	return db.Model(&models.Device{}).Where("id = ?", deviceID).Updates(map[string]interface{}{
		"user_id":        0,
		"activated":      false,
		"nick_name":      "",
		"role_id":        nil,
		"last_active_at": nil,
	}).Error
}
