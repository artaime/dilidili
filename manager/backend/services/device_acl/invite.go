package device_acl

import (
	"crypto/rand"
	"fmt"
	"strings"
	"time"

	"dili/manager/backend/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const inviteCodeAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

// CreateInvite 属主创建邀请码。
func CreateInvite(db *gorm.DB, deviceID, ownerUserID uint) (*models.DeviceInvite, error) {
	if err := AssertManage(db, deviceID, ownerUserID); err != nil {
		return nil, err
	}
	code, err := generateInviteCode(db)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	invite := models.DeviceInvite{
		DeviceID:  deviceID,
		Code:      code,
		CreatedBy: ownerUserID,
		ExpiresAt: now.Add(InviteTTL),
		MaxUses:   InviteMaxUses,
		UsedCount: 0,
		Status:    InviteStatusActive,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := db.Create(&invite).Error; err != nil {
		return nil, err
	}
	return &invite, nil
}

func generateInviteCode(db *gorm.DB) (string, error) {
	for attempt := 0; attempt < 16; attempt++ {
		buf := make([]byte, InviteCodeLength)
		if _, err := rand.Read(buf); err != nil {
			return "", err
		}
		b := make([]byte, InviteCodeLength)
		for i := range b {
			b[i] = inviteCodeAlphabet[int(buf[i])%len(inviteCodeAlphabet)]
		}
		code := string(b)
		var count int64
		if err := db.Model(&models.DeviceInvite{}).Where("code = ?", code).Count(&count).Error; err != nil {
			return "", err
		}
		if count == 0 {
			return code, nil
		}
	}
	return "", fmt.Errorf("生成邀请码失败，请重试")
}

// JoinByCode 用邀请码加入设备家庭。
func JoinByCode(db *gorm.DB, userID uint, rawCode string) (*models.Device, error) {
	code := strings.ToUpper(strings.TrimSpace(rawCode))
	if code == "" {
		return nil, ErrInviteInvalid
	}

	var joinedDevice *models.Device
	err := db.Transaction(func(tx *gorm.DB) error {
		var invite models.DeviceInvite
		q := tx.Where("code = ?", code)
		if tx.Dialector.Name() != "sqlite" {
			q = q.Clauses(clause.Locking{Strength: "UPDATE"})
		}
		if err := q.First(&invite).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return ErrInviteInvalid
			}
			return err
		}
		now := time.Now()
		if invite.Status != InviteStatusActive || invite.ExpiresAt.Before(now) || invite.UsedCount >= invite.MaxUses {
			_ = markInviteExpiredOrExhausted(tx, &invite, now)
			return ErrInviteInvalid
		}

		var device models.Device
		if err := tx.First(&device, invite.DeviceID).Error; err != nil {
			return err
		}
		if device.UserID == 0 {
			return ErrInviteInvalid
		}
		if device.UserID == userID {
			return ErrAlreadyMember
		}

		var existing models.DeviceMember
		err := tx.Where("device_id = ? AND user_id = ?", device.ID, userID).First(&existing).Error
		if err == nil && existing.Status == StatusActive {
			return ErrAlreadyMember
		}
		if err != nil && err != gorm.ErrRecordNotFound {
			return err
		}

		count, err := CountActiveMembers(tx, device.ID)
		if err != nil {
			return err
		}
		if count >= MaxMembersPerDevice {
			return ErrMemberFull
		}

		invitedBy := invite.CreatedBy
		if existing.ID > 0 {
			existing.Role = RoleMember
			existing.Status = StatusActive
			existing.InvitedBy = &invitedBy
			existing.JoinedAt = now
			existing.RevokedAt = nil
			existing.UpdatedAt = now
			if err := tx.Save(&existing).Error; err != nil {
				return err
			}
		} else {
			member := models.DeviceMember{
				DeviceID:  device.ID,
				UserID:    userID,
				Role:      RoleMember,
				Status:    StatusActive,
				InvitedBy: &invitedBy,
				JoinedAt:  now,
				CreatedAt: now,
				UpdatedAt: now,
			}
			if err := tx.Create(&member).Error; err != nil {
				return err
			}
		}

		invite.UsedCount++
		invite.UpdatedAt = now
		if invite.UsedCount >= invite.MaxUses {
			invite.Status = InviteStatusExhausted
		}
		if err := tx.Save(&invite).Error; err != nil {
			return err
		}
		joinedDevice = &device
		return nil
	})
	if err != nil {
		return nil, err
	}
	return joinedDevice, nil
}

func markInviteExpiredOrExhausted(tx *gorm.DB, invite *models.DeviceInvite, now time.Time) error {
	if invite.ExpiresAt.Before(now) {
		invite.Status = InviteStatusExpired
	} else if invite.UsedCount >= invite.MaxUses {
		invite.Status = InviteStatusExhausted
	}
	invite.UpdatedAt = now
	return tx.Save(invite).Error
}

// RevokeMember 属主踢出成员。
func RevokeMember(db *gorm.DB, deviceID, actorUserID, targetUserID uint) error {
	if err := AssertManage(db, deviceID, actorUserID); err != nil {
		return err
	}
	if targetUserID == actorUserID {
		return ErrCannotKickOwner
	}
	var device models.Device
	if err := db.Select("id", "user_id").First(&device, deviceID).Error; err != nil {
		return err
	}
	if targetUserID == device.UserID {
		return ErrCannotKickOwner
	}

	var member models.DeviceMember
	if err := db.Where("device_id = ? AND user_id = ? AND status = ?", deviceID, targetUserID, StatusActive).
		First(&member).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return ErrNotMember
		}
		return err
	}
	if member.Role == RoleOwner {
		return ErrCannotKickOwner
	}
	now := time.Now()
	return db.Model(&member).Updates(map[string]interface{}{
		"status":     StatusRevoked,
		"revoked_at": now,
		"updated_at": now,
	}).Error
}

// LeaveDevice 成员主动退出。
func LeaveDevice(db *gorm.DB, deviceID, userID uint) error {
	var device models.Device
	if err := db.Select("id", "user_id").First(&device, deviceID).Error; err != nil {
		return err
	}
	if device.UserID == userID {
		return ErrCannotLeaveOwner
	}
	var member models.DeviceMember
	if err := db.Where("device_id = ? AND user_id = ? AND status = ?", deviceID, userID, StatusActive).
		First(&member).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return ErrNotMember
		}
		return err
	}
	if member.Role == RoleOwner {
		return ErrCannotLeaveOwner
	}
	now := time.Now()
	return db.Model(&member).Updates(map[string]interface{}{
		"status":     StatusRevoked,
		"revoked_at": now,
		"updated_at": now,
	}).Error
}

// ListMembers 列出 active 成员。
func ListMembers(db *gorm.DB, deviceID uint) ([]models.DeviceMember, error) {
	var members []models.DeviceMember
	err := db.Where("device_id = ? AND status = ?", deviceID, StatusActive).
		Order("CASE WHEN role = 'owner' THEN 0 ELSE 1 END, id ASC").Find(&members).Error
	return members, err
}
