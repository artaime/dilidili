package device_reset

import (
	"os"
	"path/filepath"

	"dili/manager/backend/config"
	"dili/manager/backend/models"
	"dili/manager/backend/services/device_acl"

	"gorm.io/gorm"
)

func purgeDeviceDatabaseRecords(db *gorm.DB, cfg *config.Config, device *models.Device) error {
	if err := purgeParentMessages(db, cfg, device.ID); err != nil {
		return err
	}
	if err := purgeChatMessages(db, cfg, device.DeviceName); err != nil {
		return err
	}
	return device_acl.DeleteDeviceFamily(db, device.ID)
}

func purgeChatMessages(db *gorm.DB, cfg *config.Config, deviceSN string) error {
	audioBasePath := "./storage/chat_history/audio"
	if cfg != nil && cfg.History.AudioBasePath != "" {
		audioBasePath = cfg.History.AudioBasePath
	}

	var messages []models.ChatMessage
	if err := db.Where("device_id = ?", deviceSN).Find(&messages).Error; err != nil {
		return err
	}
	for _, msg := range messages {
		if msg.AudioPath != "" {
			_ = os.Remove(filepath.Join(audioBasePath, msg.AudioPath))
		}
	}
	if len(messages) == 0 {
		return nil
	}
	return db.Where("device_id = ?", deviceSN).Delete(&models.ChatMessage{}).Error
}

func purgeParentMessages(db *gorm.DB, cfg *config.Config, deviceDBID uint) error {
	audioBasePath := "./data/parent_messages/audio"
	if cfg != nil && cfg.ParentMessage.AudioBasePath != "" {
		audioBasePath = cfg.ParentMessage.AudioBasePath
	}

	var messages []models.ParentMessage
	if err := db.Where("device_id = ?", deviceDBID).Find(&messages).Error; err != nil {
		return err
	}
	for _, msg := range messages {
		if msg.AudioPath != "" {
			path := msg.AudioPath
			if _, err := os.Stat(path); os.IsNotExist(err) {
				path = filepath.Join(audioBasePath, msg.AudioPath)
			}
			_ = os.Remove(path)
		}
	}
	if len(messages) == 0 {
		return nil
	}
	return db.Where("device_id = ?", deviceDBID).Delete(&models.ParentMessage{}).Error
}
