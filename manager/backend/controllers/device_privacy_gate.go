package controllers

import (
	"errors"
	"net/http"

	"dili/manager/backend/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ErrAdminPrivacyForbidden 管理端仅允许查看「本人绑定」设备的隐私数据。
var ErrAdminPrivacyForbidden = errors.New("仅可查看本人绑定设备的隐私数据")

func actorUserIDFromContext(c *gin.Context) uint {
	if c == nil {
		return 0
	}
	uid, _ := c.Get("user_id")
	id, _ := uid.(uint)
	return id
}

// assertDeviceBoundToActor 管理端隐私读路径门禁：设备须绑定到当前登录管理员。
func assertDeviceBoundToActor(db *gorm.DB, deviceID, actorUserID uint) error {
	if db == nil || deviceID == 0 {
		return ErrAdminPrivacyForbidden
	}
	if actorUserID == 0 {
		return ErrAdminPrivacyForbidden
	}
	var device models.Device
	if err := db.Select("id", "user_id").Where("id = ?", deviceID).First(&device).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return gorm.ErrRecordNotFound
		}
		return err
	}
	if device.UserID == 0 || device.UserID != actorUserID {
		return ErrAdminPrivacyForbidden
	}
	return nil
}

func writeAdminPrivacyGateError(c *gin.Context, err error) bool {
	if err == nil {
		return false
	}
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "设备不存在"})
	case errors.Is(err, ErrAdminPrivacyForbidden):
		c.JSON(http.StatusForbidden, gin.H{"error": ErrAdminPrivacyForbidden.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "校验设备隐私权限失败"})
	}
	return true
}
