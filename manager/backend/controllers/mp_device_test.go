package controllers

import (
	"testing"

	"dili/manager/backend/models"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupMpDeviceTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:mp_device_test?mode=memory&cache=private"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&models.User{},
		&models.Config{},
		&models.Agent{},
		&models.Device{},
		&models.MCPMarketService{},
		&models.AgentKnowledgeBase{},
	); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	return db
}

func TestMpDeviceBindRejectsUnregisteredDevice(t *testing.T) {
	db := setupMpDeviceTestDB(t)
	user := createServiceTestUser(t, db, "mp-parent", "user")
	ctrl := &MpDeviceController{DB: db}

	_, found, err := ctrl.findDeviceBySN("SN-TEST-UNREGISTERED")
	if err != nil {
		t.Fatalf("find device: %v", err)
	}
	if found {
		t.Fatal("unregistered device should not be found")
	}

	var llmCount int64
	if db.Model(&models.Config{}).Where("type = ? AND is_default = ? AND enabled = ?", "llm", true, true).Count(&llmCount).Error != nil || llmCount == 0 {
		createServiceTestConfig(t, db, "llm", "llm-mp-default", "openai")
	}
	var ttsCount int64
	if db.Model(&models.Config{}).Where("type = ? AND is_default = ? AND enabled = ?", "tts", true, true).Count(&ttsCount).Error != nil || ttsCount == 0 {
		createServiceTestConfig(t, db, "tts", "tts-mp-default", "doubao")
	}

	device := models.Device{DeviceName: "SN-TEST-001", DeviceCode: "111111", NickName: "test-device"}
	if err := db.Create(&device).Error; err != nil {
		t.Fatalf("create device: %v", err)
	}

	agent, err := ctrl.ensureDefaultAgent(user.ID, "小明")
	if err != nil {
		t.Fatalf("ensure default agent: %v", err)
	}
	if agent.Name != "小明的小伙伴" {
		t.Fatalf("agent name = %q", agent.Name)
	}

	updates := map[string]interface{}{
		"user_id":   user.ID,
		"agent_id":  agent.ID,
		"activated": true,
	}
	if err := updateDeviceColumns(db, device.ID, updates); err != nil {
		t.Fatalf("bind device: %v", err)
	}

	var bound models.Device
	if err := db.First(&bound, device.ID).Error; err != nil {
		t.Fatalf("load bound device: %v", err)
	}
	if bound.UserID != user.ID || !bound.Activated {
		t.Fatalf("bound device = user:%d activated:%v", bound.UserID, bound.Activated)
	}
}

func TestMpDeviceBindRejectsOccupiedDevice(t *testing.T) {
	db := setupMpDeviceTestDB(t)
	userA := createServiceTestUser(t, db, "parent-a", "user")
	userB := createServiceTestUser(t, db, "parent-b", "user")

	device := models.Device{
		UserID:     userA.ID,
		DeviceName: "SN-TEST-OCCUPIED",
		DeviceCode: "222222",
		Activated:  true,
	}
	if err := db.Create(&device).Error; err != nil {
		t.Fatalf("create device: %v", err)
	}

	ctrl := &MpDeviceController{DB: db}
	foundDevice, found, err := ctrl.findDeviceBySN("SN-TEST-OCCUPIED")
	if err != nil || !found {
		t.Fatalf("find device: found=%v err=%v", found, err)
	}
	if foundDevice.UserID != userA.ID {
		t.Fatalf("device owner = %d", foundDevice.UserID)
	}
	if foundDevice.UserID == userB.ID {
		t.Fatal("device should not belong to userB")
	}
}
