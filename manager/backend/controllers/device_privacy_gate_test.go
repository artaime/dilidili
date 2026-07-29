package controllers

import (
	"testing"

	"dili/manager/backend/models"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupPrivacyGateTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:privacy_gate_" + t.Name() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.Device{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func TestAssertDeviceBoundToActor(t *testing.T) {
	db := setupPrivacyGateTestDB(t)

	adminA := models.User{Username: "admin-a", Password: "x", Email: "a@t.test", Role: "admin"}
	adminB := models.User{Username: "admin-b", Password: "x", Email: "b@t.test", Role: "admin"}
	if err := db.Create(&adminA).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&adminB).Error; err != nil {
		t.Fatal(err)
	}

	ownDev := models.Device{UserID: adminA.ID, DeviceName: "SN-A", AgentID: 1}
	otherDev := models.Device{UserID: adminB.ID, DeviceName: "SN-B", AgentID: 1}
	unbound := models.Device{UserID: 0, DeviceName: "SN-FREE", AgentID: 1}
	if err := db.Create(&ownDev).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&otherDev).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&unbound).Error; err != nil {
		t.Fatal(err)
	}

	if err := assertDeviceBoundToActor(db, ownDev.ID, adminA.ID); err != nil {
		t.Fatalf("own device: %v", err)
	}
	if err := assertDeviceBoundToActor(db, otherDev.ID, adminA.ID); err != ErrAdminPrivacyForbidden {
		t.Fatalf("other admin device: want forbidden, got %v", err)
	}
	if err := assertDeviceBoundToActor(db, unbound.ID, adminA.ID); err != ErrAdminPrivacyForbidden {
		t.Fatalf("unbound: want forbidden, got %v", err)
	}
	if err := assertDeviceBoundToActor(db, ownDev.ID, 0); err != ErrAdminPrivacyForbidden {
		t.Fatalf("zero actor: want forbidden, got %v", err)
	}
}
