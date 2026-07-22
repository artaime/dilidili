package device_acl

import (
	"testing"
	"time"

	"dili/manager/backend/models"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupACLTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:device_acl_test?mode=memory&cache=private"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.Device{}, &models.DeviceMember{}, &models.DeviceInvite{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	return db
}

func createUser(t *testing.T, db *gorm.DB, name string) models.User {
	t.Helper()
	u := models.User{Username: name, Password: "x", Role: "user", Email: name + "@test.com", Nickname: name, FamilyRole: "妈妈"}
	if err := db.Create(&u).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	return u
}

func TestEnsureOwnerAndInviteJoin(t *testing.T) {
	db := setupACLTestDB(t)
	owner := createUser(t, db, "owner1")
	member := createUser(t, db, "member1")

	device := models.Device{UserID: owner.ID, DeviceName: "SN-ACL-1", Activated: true, NickName: "小明"}
	if err := db.Create(&device).Error; err != nil {
		t.Fatalf("create device: %v", err)
	}
	if err := EnsureOwnerMember(db, device.ID, owner.ID); err != nil {
		t.Fatalf("EnsureOwnerMember: %v", err)
	}
	if !CanManage(db, device.ID, owner.ID) {
		t.Fatal("owner should manage")
	}
	if CanManage(db, device.ID, member.ID) {
		t.Fatal("member should not manage before join")
	}

	invite, err := CreateInvite(db, device.ID, owner.ID)
	if err != nil {
		t.Fatalf("CreateInvite: %v", err)
	}
	if len(invite.Code) != InviteCodeLength {
		t.Fatalf("code length = %d", len(invite.Code))
	}

	joined, err := JoinByCode(db, member.ID, invite.Code)
	if err != nil {
		t.Fatalf("JoinByCode: %v", err)
	}
	if joined.ID != device.ID {
		t.Fatalf("joined device = %d", joined.ID)
	}
	if !CanAccess(db, device.ID, member.ID) {
		t.Fatal("member should access after join")
	}
	if MemberRole(db, device.ID, member.ID) != RoleMember {
		t.Fatal("expected member role")
	}
	if _, err := JoinByCode(db, member.ID, invite.Code); err != ErrAlreadyMember {
		t.Fatalf("second join err = %v, want ErrAlreadyMember", err)
	}

	ids, err := ListAccessibleDeviceIDs(db, member.ID)
	if err != nil || len(ids) != 1 || ids[0] != device.ID {
		t.Fatalf("accessible ids = %v err=%v", ids, err)
	}
}

func TestMemberCannotInviteOrRenameBoundary(t *testing.T) {
	db := setupACLTestDB(t)
	owner := createUser(t, db, "owner2")
	member := createUser(t, db, "member2")
	device := models.Device{UserID: owner.ID, DeviceName: "SN-ACL-2", Activated: true}
	if err := db.Create(&device).Error; err != nil {
		t.Fatalf("create device: %v", err)
	}
	_ = EnsureOwnerMember(db, device.ID, owner.ID)
	invite, _ := CreateInvite(db, device.ID, owner.ID)
	_, _ = JoinByCode(db, member.ID, invite.Code)

	if _, err := CreateInvite(db, device.ID, member.ID); err != ErrOwnerOnly {
		t.Fatalf("member invite err = %v", err)
	}
	if err := LeaveDevice(db, device.ID, owner.ID); err != ErrCannotLeaveOwner {
		t.Fatalf("owner leave err = %v", err)
	}
	if err := LeaveDevice(db, device.ID, member.ID); err != nil {
		t.Fatalf("member leave: %v", err)
	}
	if CanAccess(db, device.ID, member.ID) {
		t.Fatal("member should lose access after leave")
	}
}

func TestRevokeMemberAndBackfill(t *testing.T) {
	db := setupACLTestDB(t)
	owner := createUser(t, db, "owner3")
	member := createUser(t, db, "member3")
	device := models.Device{UserID: owner.ID, DeviceName: "SN-ACL-3", Activated: true}
	if err := db.Create(&device).Error; err != nil {
		t.Fatalf("create device: %v", err)
	}
	_ = EnsureOwnerMember(db, device.ID, owner.ID)
	invite, _ := CreateInvite(db, device.ID, owner.ID)
	_, _ = JoinByCode(db, member.ID, invite.Code)

	if err := RevokeMember(db, device.ID, owner.ID, member.ID); err != nil {
		t.Fatalf("RevokeMember: %v", err)
	}
	if CanAccess(db, device.ID, member.ID) {
		t.Fatal("revoked member should not access")
	}

	device2 := models.Device{UserID: owner.ID, DeviceName: "SN-ACL-4", Activated: true}
	if err := db.Create(&device2).Error; err != nil {
		t.Fatalf("create device2: %v", err)
	}
	n, err := BackfillOwnerMembers(db)
	if err != nil {
		t.Fatalf("BackfillOwnerMembers: %v", err)
	}
	if n < 1 {
		t.Fatalf("expected backfill >= 1, got %d", n)
	}
	if MemberRole(db, device2.ID, owner.ID) != RoleOwner {
		t.Fatal("backfill should create owner member")
	}
}

func TestInviteExpired(t *testing.T) {
	db := setupACLTestDB(t)
	owner := createUser(t, db, "owner4")
	member := createUser(t, db, "member4")
	device := models.Device{UserID: owner.ID, DeviceName: "SN-ACL-5", Activated: true}
	if err := db.Create(&device).Error; err != nil {
		t.Fatalf("create device: %v", err)
	}
	_ = EnsureOwnerMember(db, device.ID, owner.ID)
	invite := models.DeviceInvite{
		DeviceID:  device.ID,
		Code:      "ZZZZ99",
		CreatedBy: owner.ID,
		ExpiresAt: time.Now().Add(-time.Hour),
		MaxUses:   5,
		Status:    InviteStatusActive,
	}
	if err := db.Create(&invite).Error; err != nil {
		t.Fatalf("create invite: %v", err)
	}
	if _, err := JoinByCode(db, member.ID, "ZZZZ99"); err != ErrInviteInvalid {
		t.Fatalf("expired invite err = %v", err)
	}
}

func TestDeleteDeviceFamily(t *testing.T) {
	db := setupACLTestDB(t)
	owner := createUser(t, db, "owner5")
	device := models.Device{UserID: owner.ID, DeviceName: "SN-ACL-6", Activated: true}
	if err := db.Create(&device).Error; err != nil {
		t.Fatalf("create device: %v", err)
	}
	_ = EnsureOwnerMember(db, device.ID, owner.ID)
	_, _ = CreateInvite(db, device.ID, owner.ID)
	if err := DeleteDeviceFamily(db, device.ID); err != nil {
		t.Fatalf("DeleteDeviceFamily: %v", err)
	}
	var mc, ic int64
	_ = db.Model(&models.DeviceMember{}).Where("device_id = ?", device.ID).Count(&mc)
	_ = db.Model(&models.DeviceInvite{}).Where("device_id = ?", device.ID).Count(&ic)
	if mc != 0 || ic != 0 {
		t.Fatalf("expected empty family, members=%d invites=%d", mc, ic)
	}
}
