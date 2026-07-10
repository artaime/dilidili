package controllers

import (
	"context"
	"testing"

	"dili/manager/backend/models"
	"dili/manager/backend/services/device_reset"

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
	ctrl := &MpDeviceController{DB: db}

	_, found, err := ctrl.findDeviceBySN("SN-TEST-UNREGISTERED")
	if err != nil {
		t.Fatalf("find device: %v", err)
	}
	if found {
		t.Fatal("unregistered device should not be found")
	}
}

func TestMpDeviceBindRequiresFactoryAgent(t *testing.T) {
	db := setupMpDeviceTestDB(t)
	user := createServiceTestUser(t, db, "mp-parent", "user")
	admin := createServiceTestUser(t, db, "mp-admin", "admin")
	agent := models.Agent{UserID: admin.ID, Name: "factory-agent", Nickname: "狄哩"}
	if err := db.Create(&agent).Error; err != nil {
		t.Fatalf("create agent: %v", err)
	}

	device := models.Device{DeviceName: "SN-TEST-001", DeviceCode: "111111", NickName: "test-device", AgentID: agent.ID}
	if err := db.Create(&device).Error; err != nil {
		t.Fatalf("create device: %v", err)
	}

	updates := map[string]interface{}{
		"user_id":   user.ID,
		"activated": true,
		"nick_name": "小明",
	}
	if err := updateDeviceColumns(db, device.ID, updates); err != nil {
		t.Fatalf("bind device: %v", err)
	}

	var bound models.Device
	if err := db.First(&bound, device.ID).Error; err != nil {
		t.Fatalf("load bound device: %v", err)
	}
	if bound.UserID != user.ID || !bound.Activated || bound.AgentID != agent.ID {
		t.Fatalf("bound device = user:%d agent:%d activated:%v", bound.UserID, bound.AgentID, bound.Activated)
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

func TestMpDeviceUnbindPreservesFactoryAgent(t *testing.T) {
	db := setupMpDeviceTestDB(t)
	if err := db.AutoMigrate(&models.ChatMessage{}, &models.ParentMessage{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	user := createServiceTestUser(t, db, "mp-unbind", "user")
	admin := createServiceTestUser(t, db, "mp-admin-unbind", "admin")
	agent := models.Agent{UserID: admin.ID, Name: "factory-agent", Nickname: "狄哩"}
	if err := db.Create(&agent).Error; err != nil {
		t.Fatalf("create agent: %v", err)
	}

	roleID := uint(7)
	device := models.Device{
		UserID:     user.ID,
		AgentID:    agent.ID,
		RoleID:     &roleID,
		NickName:   "小明",
		DeviceName: "SN-TEST-UNBIND",
		DeviceCode: "333333",
		Activated:  true,
	}
	if err := db.Create(&device).Error; err != nil {
		t.Fatalf("create device: %v", err)
	}

	svc := device_reset.NewService(db, nil, device_resetNoopNotifier{})
	if err := svc.ResetToFactory(context.Background(), device.ID, user.ID); err != nil {
		t.Fatalf("ResetToFactory: %v", err)
	}

	var unbound models.Device
	if err := db.First(&unbound, device.ID).Error; err != nil {
		t.Fatalf("load unbound device: %v", err)
	}
	if unbound.UserID != 0 || unbound.Activated || unbound.AgentID != agent.ID {
		t.Fatalf("unbound device = user:%d agent:%d activated:%v, want user:0 agent:%d activated:false", unbound.UserID, unbound.AgentID, unbound.Activated, agent.ID)
	}
	if unbound.NickName != "" || unbound.RoleID != nil {
		t.Fatalf("nick_name/role_id not cleared: nick=%q role=%v", unbound.NickName, unbound.RoleID)
	}
}

type device_resetNoopNotifier struct{}

func (device_resetNoopNotifier) NotifyDeviceReset(context.Context, string) {}
