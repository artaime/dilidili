package controllers

import (
	"errors"
	"strings"

	"dili/manager/backend/models"

	"gorm.io/gorm"
)

// 设备更新只写入明确声明的列，避免将历史零时间 created_at 等字段整行回写。
func updateDeviceColumns(db *gorm.DB, deviceID uint, updates map[string]interface{}) error {
	if deviceID == 0 {
		return errors.New("device id is required")
	}
	if len(updates) == 0 {
		return nil
	}

	return wrapDevicePersistenceError(db.Model(&models.Device{}).Where("id = ?", deviceID).Updates(updates).Error)
}

func isDuplicateDeviceNameError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "duplicate") ||
		strings.Contains(msg, "unique constraint") ||
		strings.Contains(msg, "idx_devices_device_name")
}

func wrapDevicePersistenceError(err error) error {
	if err == nil {
		return nil
	}
	if isDuplicateDeviceNameError(err) {
		return errors.New("设备已添加")
	}
	return err
}

func countDevicesByAgentID(db *gorm.DB, agentID uint) (int64, error) {
	if agentID == 0 {
		return 0, nil
	}

	var count int64
	err := db.Model(&models.Device{}).Where("agent_id = ?", agentID).Count(&count).Error
	return count, err
}
