package privacy

import (
	"encoding/base64"
	"fmt"
	"testing"

	"dili/manager/backend/models"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:privacy_test_%s?mode=memory&cache=shared", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.Device{}, &models.DeviceEncryptionKey{}); err != nil {
		t.Fatal(err)
	}
	dev := models.Device{UserID: 1, DeviceName: "SN-" + t.Name()}
	if err := db.Create(&dev).Error; err != nil {
		t.Fatal(err)
	}
	return db
}

func TestServiceEncryptDecryptWithDEK(t *testing.T) {
	db := setupTestDB(t)
	var dev models.Device
	_ = db.First(&dev).Error
	kek := make([]byte, 32)
	for i := range kek {
		kek[i] = byte(i + 1)
	}
	svc, err := NewService(db, Config{Enabled: true, KeyID: "k1", KEK: kek})
	if err != nil {
		t.Fatal(err)
	}
	ct, err := svc.EncryptText(dev.ID, "家长留言内容")
	if err != nil {
		t.Fatal(err)
	}
	if !IsCiphertext(ct) {
		t.Fatal("expected ciphertext")
	}
	pt, err := svc.DecryptText(dev.ID, ct)
	if err != nil {
		t.Fatal(err)
	}
	if pt != "家长留言内容" {
		t.Fatalf("got %q", pt)
	}
	// same device reuses same key row
	var count int64
	db.Model(&models.DeviceEncryptionKey{}).Where("device_id = ?", dev.ID).Count(&count)
	if count != 1 {
		t.Fatalf("expected 1 key row, got %d", count)
	}
}

func TestNewServiceEnabledWithoutKEKFails(t *testing.T) {
	db := setupTestDB(t)
	t.Setenv(KEKEnv, "")
	_, err := NewService(db, Config{Enabled: true, KeyID: "k1"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestNewServiceDisabledOK(t *testing.T) {
	db := setupTestDB(t)
	svc, err := NewService(db, Config{Enabled: false})
	if err != nil {
		t.Fatal(err)
	}
	out, err := svc.EncryptText(1, "plain")
	if err != nil || out != "plain" {
		t.Fatalf("got %q %v", out, err)
	}
}

func TestLoadKEKFromEnv(t *testing.T) {
	raw := make([]byte, 32)
	for i := range raw {
		raw[i] = byte(i)
	}
	t.Setenv(KEKEnv, base64.StdEncoding.EncodeToString(raw))
	kek, err := LoadKEKFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	if len(kek) != 32 {
		t.Fatal(len(kek))
	}
}
