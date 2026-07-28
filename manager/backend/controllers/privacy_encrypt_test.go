package controllers

import (
	"testing"

	"dili/manager/backend/models"
	"dili/manager/backend/privacy"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestConversationRecordsDecryptEncryptedContent(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:privacy_conv_test?mode=memory&cache=private"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(
		&models.User{}, &models.Device{}, &models.DeviceMember{},
		&models.ChatMessage{}, &models.ParentMessage{}, &models.DeviceEncryptionKey{},
	); err != nil {
		t.Fatal(err)
	}
	user := models.User{Username: "priv-user", Password: "x", Role: "user"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	device := models.Device{UserID: user.ID, DeviceName: "SN-PRIV-001", Activated: true, AgentID: 1}
	if err := db.Create(&device).Error; err != nil {
		t.Fatal(err)
	}

	kek := make([]byte, 32)
	for i := range kek {
		kek[i] = byte(i + 3)
	}
	svc, err := privacy.NewService(db, privacy.Config{Enabled: true, KeyID: "k1", KEK: kek})
	if err != nil {
		t.Fatal(err)
	}
	ct, err := svc.EncryptText(device.ID, "孩子你好")
	if err != nil {
		t.Fatal(err)
	}
	if !privacy.IsCiphertext(ct) {
		t.Fatal("expected ciphertext")
	}
	msg := models.ChatMessage{
		MessageID: "m1", DeviceID: device.DeviceName, AgentID: "1",
		UserID: user.ID, Role: "user", Content: ct,
	}
	if err := db.Create(&msg).Error; err != nil {
		t.Fatal(err)
	}

	items, _, _, err := listConversationRecords(db, svc, device.DeviceName, device.ID, user.ID, conversationRecordQuery{Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Content != "孩子你好" {
		t.Fatalf("got %+v", items)
	}
}
