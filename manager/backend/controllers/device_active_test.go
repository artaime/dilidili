package controllers

import (
	"testing"
	"time"

	"dili/manager/backend/models"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupDeviceActiveTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:device_active_" + t.Name() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&models.Device{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func TestMarkDeviceActiveAndInactivePreservesLastActiveAt(t *testing.T) {
	db := setupDeviceActiveTestDB(t)
	device := models.Device{DeviceName: "SN-ACTIVE-1", NickName: "测试设备", Activated: true}
	if err := db.Create(&device).Error; err != nil {
		t.Fatal(err)
	}

	activeAt := time.Date(2026, 7, 29, 10, 0, 0, 0, time.UTC)
	got, err := markDeviceActive(db, "SN-ACTIVE-1", activeAt)
	if err != nil {
		t.Fatalf("markDeviceActive: %v", err)
	}
	if !got.Equal(activeAt) {
		t.Fatalf("activeAt=%v want %v", got, activeAt)
	}

	var afterActive models.Device
	if err := db.Where("device_name = ?", "SN-ACTIVE-1").First(&afterActive).Error; err != nil {
		t.Fatal(err)
	}
	if afterActive.LastActiveAt == nil || !afterActive.LastActiveAt.Equal(activeAt) {
		t.Fatalf("last_active_at after active=%v want %v", afterActive.LastActiveAt, activeAt)
	}

	preserved, err := markDeviceInactive(db, "SN-ACTIVE-1")
	if err != nil {
		t.Fatalf("markDeviceInactive: %v", err)
	}
	if preserved == nil || !preserved.Equal(activeAt) {
		t.Fatalf("inactive returned last_active_at=%v want %v", preserved, activeAt)
	}

	var afterInactive models.Device
	if err := db.Where("device_name = ?", "SN-ACTIVE-1").First(&afterInactive).Error; err != nil {
		t.Fatal(err)
	}
	if afterInactive.LastActiveAt == nil || !afterInactive.LastActiveAt.Equal(activeAt) {
		t.Fatalf("last_active_at cleared on inactive: got %v", afterInactive.LastActiveAt)
	}
}

func TestMarkDeviceInactiveMissingDevice(t *testing.T) {
	db := setupDeviceActiveTestDB(t)
	_, err := markDeviceInactive(db, "SN-MISSING")
	if err != gorm.ErrRecordNotFound {
		t.Fatalf("err=%v want ErrRecordNotFound", err)
	}
}
