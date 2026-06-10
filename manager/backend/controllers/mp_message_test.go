package controllers

import (
	"testing"
	"time"

	"xiaozhi/manager/backend/models"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupMpMessageTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:mp_message_test?mode=memory&cache=private"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.Device{}, &models.ParentMessage{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	return db
}

func TestParentMessageStatusUpdate(t *testing.T) {
	db := setupMpMessageTestDB(t)
	user := createServiceTestUser(t, db, "msg-parent", "user")
	device := models.Device{UserID: user.ID, DeviceName: "11:22:33:44:55:66", DeviceCode: "333333", Activated: true}
	if err := db.Create(&device).Error; err != nil {
		t.Fatalf("create device: %v", err)
	}

	msg := models.ParentMessage{
		UserID:      user.ID,
		DeviceID:    device.ID,
		TextContent: "宝贝今天要开心哦",
		SourceType:  "text",
		Status:      "pending",
	}
	if err := db.Create(&msg).Error; err != nil {
		t.Fatalf("create message: %v", err)
	}

	if err := updateParentMessageStatus(db, msg.ID, "played"); err != nil {
		t.Fatalf("update status: %v", err)
	}

	var updated models.ParentMessage
	if err := db.First(&updated, msg.ID).Error; err != nil {
		t.Fatalf("load message: %v", err)
	}
	if updated.Status != "played" || updated.PlayedAt == nil {
		t.Fatalf("status = %s played_at = %v", updated.Status, updated.PlayedAt)
	}
	if updated.PlayedAt.Before(time.Now().Add(-time.Minute)) {
		t.Fatalf("played_at not recent: %v", updated.PlayedAt)
	}
}

func TestFindPendingParentMessage(t *testing.T) {
	db := setupMpMessageTestDB(t)
	user := createServiceTestUser(t, db, "pending-parent", "user")
	device := models.Device{UserID: user.ID, DeviceName: "AA-BB-CC-DD-EE-FF", DeviceCode: "444444", Activated: true}
	if err := db.Create(&device).Error; err != nil {
		t.Fatalf("create device: %v", err)
	}
	older := models.ParentMessage{UserID: user.ID, DeviceID: device.ID, TextContent: "第一条", Status: "pending"}
	newer := models.ParentMessage{UserID: user.ID, DeviceID: device.ID, TextContent: "第二条", Status: "pending"}
	if err := db.Create(&older).Error; err != nil || db.Create(&newer).Error != nil {
		t.Fatalf("create messages")
	}

	pending, err := findPendingParentMessage(db, "aa:bb:cc:dd:ee:ff")
	if err != nil {
		t.Fatalf("find pending: %v", err)
	}
	if pending.ID != older.ID || pending.TextContent != "第一条" {
		t.Fatalf("pending = id:%d text:%q, want older message", pending.ID, pending.TextContent)
	}
}
