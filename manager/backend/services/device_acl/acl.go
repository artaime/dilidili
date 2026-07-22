package device_acl

import (
	"errors"
	"fmt"
	"time"

	"dili/manager/backend/models"

	"gorm.io/gorm"
)

const (
	RoleOwner  = "owner"
	RoleMember = "member"

	StatusActive  = "active"
	StatusRevoked = "revoked"

	InviteStatusActive    = "active"
	InviteStatusExhausted = "exhausted"
	InviteStatusRevoked   = "revoked"
	InviteStatusExpired   = "expired"

	MaxMembersPerDevice = 6
	InviteCodeLength    = 6
	InviteTTL           = 24 * time.Hour
	InviteMaxUses       = 5
)

var (
	ErrForbidden       = errors.New("无权操作该设备")
	ErrNotMember       = errors.New("不是该设备的家庭成员")
	ErrAlreadyMember   = errors.New("已是该设备的家庭成员")
	ErrMemberFull      = errors.New("家庭成员已满")
	ErrInviteInvalid   = errors.New("邀请码无效或已失效")
	ErrCannotKickOwner = errors.New("不能移除属主")
	ErrCannotLeaveOwner = errors.New("属主不能退出，请解绑设备或转让后再试")
	ErrOwnerOnly       = errors.New("仅属主可执行此操作")
)

// CanAccess 属主或 active member 可访问设备。
func CanAccess(db *gorm.DB, deviceID, userID uint) bool {
	if deviceID == 0 || userID == 0 {
		return false
	}
	var device models.Device
	if err := db.Select("id", "user_id").First(&device, deviceID).Error; err != nil {
		return false
	}
	if device.UserID == userID {
		return true
	}
	var count int64
	if err := db.Model(&models.DeviceMember{}).
		Where("device_id = ? AND user_id = ? AND status = ?", deviceID, userID, StatusActive).
		Count(&count).Error; err != nil {
		return false
	}
	return count > 0
}

// CanManage 仅属主（devices.user_id）可管理。
func CanManage(db *gorm.DB, deviceID, userID uint) bool {
	if deviceID == 0 || userID == 0 {
		return false
	}
	var device models.Device
	if err := db.Select("id", "user_id").First(&device, deviceID).Error; err != nil {
		return false
	}
	return device.UserID == userID
}

// MemberRole 返回用户在设备上的角色；无权限时返回空串。
func MemberRole(db *gorm.DB, deviceID, userID uint) string {
	if deviceID == 0 || userID == 0 {
		return ""
	}
	var device models.Device
	if err := db.Select("id", "user_id").First(&device, deviceID).Error; err != nil {
		return ""
	}
	if device.UserID == userID {
		return RoleOwner
	}
	var member models.DeviceMember
	err := db.Where("device_id = ? AND user_id = ? AND status = ?", deviceID, userID, StatusActive).
		First(&member).Error
	if err != nil {
		return ""
	}
	if member.Role == RoleOwner {
		return RoleOwner
	}
	return RoleMember
}

// ListAccessibleDeviceIDs 返回用户可访问的设备 ID（属主 ∪ active member）。
func ListAccessibleDeviceIDs(db *gorm.DB, userID uint) ([]uint, error) {
	if userID == 0 {
		return nil, nil
	}
	idSet := map[uint]struct{}{}

	var owned []uint
	if err := db.Model(&models.Device{}).Where("user_id = ?", userID).Pluck("id", &owned).Error; err != nil {
		return nil, err
	}
	for _, id := range owned {
		idSet[id] = struct{}{}
	}

	var memberOf []uint
	if err := db.Model(&models.DeviceMember{}).
		Where("user_id = ? AND status = ?", userID, StatusActive).
		Pluck("device_id", &memberOf).Error; err != nil {
		return nil, err
	}
	for _, id := range memberOf {
		idSet[id] = struct{}{}
	}

	out := make([]uint, 0, len(idSet))
	for id := range idSet {
		out = append(out, id)
	}
	return out, nil
}

// EnsureOwnerMember 保证属主在 members 表有 active owner 行（幂等）。
func EnsureOwnerMember(db *gorm.DB, deviceID, ownerUserID uint) error {
	if deviceID == 0 || ownerUserID == 0 {
		return fmt.Errorf("invalid device or owner")
	}
	now := time.Now()
	var existing models.DeviceMember
	err := db.Where("device_id = ? AND user_id = ?", deviceID, ownerUserID).First(&existing).Error
	if err == gorm.ErrRecordNotFound {
		member := models.DeviceMember{
			DeviceID:  deviceID,
			UserID:    ownerUserID,
			Role:      RoleOwner,
			Status:    StatusActive,
			JoinedAt:  now,
			CreatedAt: now,
			UpdatedAt: now,
		}
		return db.Create(&member).Error
	}
	if err != nil {
		return err
	}
	return db.Model(&existing).Updates(map[string]interface{}{
		"role":       RoleOwner,
		"status":     StatusActive,
		"revoked_at": nil,
		"updated_at": now,
	}).Error
}

// CountActiveMembers 统计 active 成员数（含 owner）。
func CountActiveMembers(db *gorm.DB, deviceID uint) (int64, error) {
	var count int64
	err := db.Model(&models.DeviceMember{}).
		Where("device_id = ? AND status = ?", deviceID, StatusActive).
		Count(&count).Error
	return count, err
}

// BackfillOwnerMembers 为已绑定但缺 owner 行的设备补插属主成员。
func BackfillOwnerMembers(db *gorm.DB) (int, error) {
	var devices []models.Device
	if err := db.Where("user_id > 0").Find(&devices).Error; err != nil {
		return 0, err
	}
	n := 0
	for _, d := range devices {
		var count int64
		if err := db.Model(&models.DeviceMember{}).
			Where("device_id = ? AND user_id = ? AND role = ? AND status = ?", d.ID, d.UserID, RoleOwner, StatusActive).
			Count(&count).Error; err != nil {
			return n, err
		}
		if count > 0 {
			continue
		}
		if err := EnsureOwnerMember(db, d.ID, d.UserID); err != nil {
			return n, err
		}
		n++
	}
	return n, nil
}

// DeleteDeviceFamily 出厂解绑时清理成员与邀请。
func DeleteDeviceFamily(db *gorm.DB, deviceID uint) error {
	if err := db.Where("device_id = ?", deviceID).Delete(&models.DeviceMember{}).Error; err != nil {
		return err
	}
	return db.Where("device_id = ?", deviceID).Delete(&models.DeviceInvite{}).Error
}

// AssertAccess 无权限时返回 ErrForbidden。
func AssertAccess(db *gorm.DB, deviceID, userID uint) error {
	if !CanAccess(db, deviceID, userID) {
		return ErrForbidden
	}
	return nil
}

// AssertManage 非属主返回 ErrOwnerOnly。
func AssertManage(db *gorm.DB, deviceID, userID uint) error {
	if !CanManage(db, deviceID, userID) {
		return ErrOwnerOnly
	}
	return nil
}
