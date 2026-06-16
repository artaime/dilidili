package controllers

import (
	"testing"
	"time"

	"dili/manager/backend/models"

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
	device := models.Device{UserID: user.ID, DeviceName: "SN-TEST-001", DeviceCode: "333333", Activated: true}
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

func TestFindSelectableParentMessagesIncludesPlayedByCreatedAt(t *testing.T) {
	db := setupMpMessageTestDB(t)
	user := createServiceTestUser(t, db, "search-parent", "user")
	device := models.Device{UserID: user.ID, DeviceName: "SN-TEST-SEARCH", DeviceCode: "444444", Activated: true}
	if err := db.Create(&device).Error; err != nil {
		t.Fatalf("create device: %v", err)
	}

	loc := time.Local
	afternoon := time.Date(2026, 6, 15, 15, 0, 0, 0, loc)
	morning := time.Date(2026, 6, 15, 9, 0, 0, 0, loc)
	playedAt := afternoon.Add(2 * time.Hour)

	playedMsg := models.ParentMessage{
		UserID: user.ID, DeviceID: device.ID, TextContent: "下午留言",
		SourceType: "text", Status: "played", CreatedAt: afternoon, PlayedAt: &playedAt,
	}
	pendingMsg := models.ParentMessage{
		UserID: user.ID, DeviceID: device.ID, TextContent: "早上留言",
		SourceType: "text", Status: "pending", CreatedAt: morning,
	}
	if err := db.Create(&playedMsg).Error; err != nil {
		t.Fatalf("create played: %v", err)
	}
	if err := db.Create(&pendingMsg).Error; err != nil {
		t.Fatalf("create pending: %v", err)
	}

	start := time.Date(2026, 6, 15, 12, 0, 0, 0, loc)
	end := time.Date(2026, 6, 15, 19, 0, 0, 0, loc)
	messages, err := findSelectableParentMessages(db, "SN-TEST-SEARCH", &start, &end, 100)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(messages) != 1 || messages[0].ID != playedMsg.ID {
		t.Fatalf("expected played afternoon message, got %+v", messages)
	}
}

func TestDeletePlayedParentMessage(t *testing.T) {
	db := setupMpMessageTestDB(t)
	user := createServiceTestUser(t, db, "delete-parent", "user")
	device := models.Device{UserID: user.ID, DeviceName: "SN-TEST-DELETE", DeviceCode: "555555", Activated: true}
	if err := db.Create(&device).Error; err != nil {
		t.Fatalf("create device: %v", err)
	}
	now := time.Now()
	msg := models.ParentMessage{
		UserID: user.ID, DeviceID: device.ID, TextContent: "已播留言",
		SourceType: "text", Status: "played", PlayedAt: &now,
	}
	if err := db.Create(&msg).Error; err != nil {
		t.Fatalf("create message: %v", err)
	}
	if err := db.Delete(&msg).Error; err != nil {
		t.Fatalf("delete played message: %v", err)
	}
	var count int64
	if err := db.Model(&models.ParentMessage{}).Where("id = ?", msg.ID).Count(&count).Error; err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected message deleted, count=%d", count)
	}
}

func TestFindPendingParentMessage(t *testing.T) {
	db := setupMpMessageTestDB(t)
	user := createServiceTestUser(t, db, "pending-parent", "user")
	device := models.Device{UserID: user.ID, DeviceName: "SN-TEST-PENDING", DeviceCode: "444444", Activated: true}
	if err := db.Create(&device).Error; err != nil {
		t.Fatalf("create device: %v", err)
	}
	older := models.ParentMessage{UserID: user.ID, DeviceID: device.ID, TextContent: "第一条", Status: "pending"}
	newer := models.ParentMessage{UserID: user.ID, DeviceID: device.ID, TextContent: "第二条", Status: "pending"}
	if err := db.Create(&older).Error; err != nil || db.Create(&newer).Error != nil {
		t.Fatalf("create messages")
	}

	pending, err := findPendingParentMessage(db, "SN-TEST-PENDING")
	if err != nil {
		t.Fatalf("find pending: %v", err)
	}
	if pending.ID != older.ID || pending.TextContent != "第一条" {
		t.Fatalf("pending = id:%d text:%q, want older message", pending.ID, pending.TextContent)
	}
}
