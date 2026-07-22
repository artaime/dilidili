package device_reset

import (
	"context"
	"testing"
	"time"

	"dili/manager/backend/models"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupDeviceResetTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:device_reset_test?mode=memory&cache=private"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&models.User{},
		&models.Device{},
		&models.DeviceMember{},
		&models.DeviceInvite{},
		&models.ChatMessage{},
		&models.ParentMessage{},
		&models.Config{},
		&models.Agent{},
	); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	return db
}

type noopNotifier struct{}

func (noopNotifier) NotifyDeviceReset(context.Context, string) {}

func TestResetToFactoryClearsRecordsAndResetsDevice(t *testing.T) {
	db := setupDeviceResetTestDB(t)
	user := models.User{Username: "parent-reset", Role: "user", Email: "parent-reset@test.com"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	agentID := uint(9)
	device := models.Device{
		UserID:     user.ID,
		AgentID:    agentID,
		DeviceName: "SN-RESET-001",
		DeviceCode: "444444",
		NickName:   "小明",
		Activated:  true,
	}
	if err := db.Create(&device).Error; err != nil {
		t.Fatalf("create device: %v", err)
	}
	roleID := uint(3)
	if err := db.Model(&device).Update("role_id", roleID).Error; err != nil {
		t.Fatalf("set role_id: %v", err)
	}
	now := time.Now()
	if err := db.Model(&device).Update("last_active_at", now).Error; err != nil {
		t.Fatalf("set last_active_at: %v", err)
	}

	chat := models.ChatMessage{
		MessageID: "m-reset-1",
		DeviceID:  device.DeviceName,
		AgentID:   "9",
		UserID:    user.ID,
		Role:      "user",
		Content:   "hello",
	}
	if err := db.Create(&chat).Error; err != nil {
		t.Fatalf("create chat: %v", err)
	}
	parent := models.ParentMessage{
		UserID:      user.ID,
		DeviceID:    device.ID,
		TextContent: "留言",
		SourceType:  "text",
		Status:      "pending",
	}
	if err := db.Create(&parent).Error; err != nil {
		t.Fatalf("create parent message: %v", err)
	}
	if err := db.Create(&models.DeviceMember{
		DeviceID: device.ID,
		UserID:   user.ID,
		Role:     "owner",
		Status:   "active",
		JoinedAt: now,
	}).Error; err != nil {
		t.Fatalf("create member: %v", err)
	}
	if err := db.Create(&models.DeviceInvite{
		DeviceID:  device.ID,
		Code:      "RESET1",
		CreatedBy: user.ID,
		ExpiresAt: now.Add(time.Hour),
		MaxUses:   5,
		Status:    "active",
	}).Error; err != nil {
		t.Fatalf("create invite: %v", err)
	}

	svc := NewService(db, nil, noopNotifier{})
	if err := svc.ResetToFactory(context.Background(), device.ID, user.ID); err != nil {
		t.Fatalf("ResetToFactory: %v", err)
	}

	var chatCount int64
	if err := db.Model(&models.ChatMessage{}).Where("device_id = ?", device.DeviceName).Count(&chatCount).Error; err != nil {
		t.Fatalf("count chat: %v", err)
	}
	if chatCount != 0 {
		t.Fatalf("chat messages = %d, want 0", chatCount)
	}

	var parentCount int64
	if err := db.Model(&models.ParentMessage{}).Where("device_id = ?", device.ID).Count(&parentCount).Error; err != nil {
		t.Fatalf("count parent: %v", err)
	}
	if parentCount != 0 {
		t.Fatalf("parent messages = %d, want 0", parentCount)
	}

	var reset models.Device
	if err := db.First(&reset, device.ID).Error; err != nil {
		t.Fatalf("load device: %v", err)
	}
	if reset.UserID != 0 || reset.Activated || reset.NickName != "" || reset.RoleID != nil {
		t.Fatalf("device not reset: user=%d activated=%v nick=%q role=%v", reset.UserID, reset.Activated, reset.NickName, reset.RoleID)
	}
	if reset.AgentID != agentID || reset.DeviceName != "SN-RESET-001" {
		t.Fatalf("factory fields changed: agent=%d sn=%s", reset.AgentID, reset.DeviceName)
	}

	var memberCount, inviteCount int64
	_ = db.Model(&models.DeviceMember{}).Where("device_id = ?", device.ID).Count(&memberCount)
	_ = db.Model(&models.DeviceInvite{}).Where("device_id = ?", device.ID).Count(&inviteCount)
	if memberCount != 0 || inviteCount != 0 {
		t.Fatalf("family not cleared: members=%d invites=%d", memberCount, inviteCount)
	}
}

func TestResetToFactoryForbiddenForOtherUser(t *testing.T) {
	db := setupDeviceResetTestDB(t)
	owner := models.User{Username: "owner", Role: "user", Email: "owner-reset@test.com"}
	other := models.User{Username: "other", Role: "user", Email: "other-reset@test.com"}
	if err := db.Create(&owner).Error; err != nil {
		t.Fatalf("create owner: %v", err)
	}
	if err := db.Create(&other).Error; err != nil {
		t.Fatalf("create other: %v", err)
	}
	device := models.Device{
		UserID:     owner.ID,
		AgentID:    1,
		DeviceName: "SN-RESET-002",
		DeviceCode: "555555",
		Activated:  true,
	}
	if err := db.Create(&device).Error; err != nil {
		t.Fatalf("create device: %v", err)
	}

	svc := NewService(db, nil, noopNotifier{})
	err := svc.ResetToFactory(context.Background(), device.ID, other.ID)
	if err == nil || err != ErrForbidden {
		t.Fatalf("ResetToFactory err = %v, want ErrForbidden", err)
	}
}

func TestResetToFactoryByAdminSkipsOwnership(t *testing.T) {
	db := setupDeviceResetTestDB(t)
	owner := models.User{Username: "owner-admin", Role: "user", Email: "owner-admin@test.com"}
	if err := db.Create(&owner).Error; err != nil {
		t.Fatalf("create owner: %v", err)
	}
	device := models.Device{
		UserID:     owner.ID,
		AgentID:    1,
		DeviceName: "SN-RESET-ADMIN",
		DeviceCode: "666666",
		NickName:   "测试",
		Activated:  true,
	}
	if err := db.Create(&device).Error; err != nil {
		t.Fatalf("create device: %v", err)
	}

	svc := NewService(db, nil, noopNotifier{})
	if err := svc.ResetToFactoryByAdmin(context.Background(), device.ID); err != nil {
		t.Fatalf("ResetToFactoryByAdmin: %v", err)
	}

	var reset models.Device
	if err := db.First(&reset, device.ID).Error; err != nil {
		t.Fatalf("load device: %v", err)
	}
	if reset.UserID != 0 || reset.Activated || reset.NickName != "" {
		t.Fatalf("device not reset by admin: user=%d activated=%v nick=%q", reset.UserID, reset.Activated, reset.NickName)
	}
}
