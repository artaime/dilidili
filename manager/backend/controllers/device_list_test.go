package controllers

import (
	"testing"
	"time"

	"dili/manager/backend/models"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupDeviceListTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:device_list_" + t.Name() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.Agent{}, &models.Device{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func TestDeviceServiceListPaged(t *testing.T) {
	db := setupDeviceListTestDB(t)
	admin := models.User{Username: "admin1", Password: "x", Email: "a@t.test", Role: "admin"}
	user := models.User{Username: "parent1", Password: "x", Email: "p@t.test", Role: "user"}
	if err := db.Create(&admin).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	agentA := models.Agent{UserID: admin.ID, Name: "助手A"}
	agentB := models.Agent{UserID: user.ID, Name: "助手B"}
	if err := db.Create(&agentA).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&agentB).Error; err != nil {
		t.Fatal(err)
	}

	now := time.Now()
	mine := models.Device{
		UserID: admin.ID, AgentID: agentA.ID, DeviceName: "SN-MINE",
		NickName: "我的设备", Activated: true, LastActiveAt: &now,
	}
	other := models.Device{
		UserID: user.ID, AgentID: agentB.ID, DeviceName: "SN-OTHER",
		NickName: "别人的", Activated: false,
	}
	unassigned := models.Device{
		UserID: 0, AgentID: 0, DeviceName: "SN-FREE", NickName: "未分配机", Activated: false,
	}
	for _, d := range []*models.Device{&other, &unassigned, &mine} {
		if err := db.Create(d).Error; err != nil {
			t.Fatal(err)
		}
	}

	svc := NewDeviceService(db)
	scope := accessScope{ActorUserID: admin.ID, IsAdmin: true}

	result, err := svc.ListPaged(scope, DeviceListQuery{Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("ListPaged: %v", err)
	}
	if result.Total != 3 {
		t.Fatalf("total=%d want 3", result.Total)
	}
	if len(result.Items) == 0 || result.Items[0].DeviceName != "SN-MINE" {
		t.Fatalf("own device should be pinned first, got %+v", result.Items)
	}
	if len(result.AgentStats) != 3 {
		t.Fatalf("agent_stats len=%d want 3", len(result.AgentStats))
	}

	filtered, err := svc.ListPaged(scope, DeviceListQuery{
		DeviceID: "SN-OTHER", Page: 1, PageSize: 10,
	})
	if err != nil {
		t.Fatal(err)
	}
	if filtered.Total != 1 || filtered.Items[0].DeviceName != "SN-OTHER" {
		t.Fatalf("device_id filter failed: %+v", filtered)
	}

	byUser, err := svc.ListPaged(scope, DeviceListQuery{
		BindUser: "parent1", Page: 1, PageSize: 10,
	})
	if err != nil {
		t.Fatal(err)
	}
	if byUser.Total != 1 || byUser.Items[0].UserID != user.ID {
		t.Fatalf("bind_user filter failed: %+v", byUser)
	}

	agentID := agentA.ID
	byAgent, err := svc.ListPaged(scope, DeviceListQuery{
		AgentID: &agentID, Activated: "activated", Page: 1, PageSize: 10,
	})
	if err != nil {
		t.Fatal(err)
	}
	if byAgent.Total != 1 || byAgent.Items[0].ID != mine.ID {
		t.Fatalf("agent+activated filter failed: %+v", byAgent)
	}

	zero := uint(0)
	unassignedList, err := svc.ListPaged(scope, DeviceListQuery{
		AgentID: &zero, Page: 1, PageSize: 10,
	})
	if err != nil {
		t.Fatal(err)
	}
	if unassignedList.Total != 1 || unassignedList.Items[0].AgentID != 0 {
		t.Fatalf("unassigned filter failed: %+v", unassignedList)
	}
}
