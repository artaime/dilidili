package controllers

import (
	"errors"
	"net/http"
	"strconv"

	"dili/manager/backend/config"
	"dili/manager/backend/models"
	"dili/manager/backend/services/device_acl"
	"dili/manager/backend/services/device_story"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type DeviceStoryController struct {
	DB  *gorm.DB
	Cfg *config.Config
}

func (ctrl *DeviceStoryController) ListDeviceStories(c *gin.Context) {
	deviceID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || deviceID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "设备 ID 无效"})
		return
	}

	limit := 50
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}

	svc := device_story.NewService(ctrl.DB, ctrl.Cfg)
	view, err := svc.ListDeviceStories(c.Request.Context(), uint(deviceID), limit)
	if err != nil {
		writeDeviceStoryError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": view})
}

func (ctrl *DeviceStoryController) GetDeviceStory(c *gin.Context) {
	deviceID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || deviceID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "设备 ID 无效"})
		return
	}
	storyID := c.Param("storyId")
	if storyID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "故事 ID 无效"})
		return
	}

	svc := device_story.NewService(ctrl.DB, ctrl.Cfg)
	view, err := svc.GetDeviceStory(c.Request.Context(), uint(deviceID), storyID)
	if err != nil {
		writeDeviceStoryError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": view})
}

func (ctrl *DeviceStoryController) DeleteDeviceStory(c *gin.Context) {
	deviceID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || deviceID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "设备 ID 无效"})
		return
	}
	storyID := c.Param("storyId")
	if storyID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "故事 ID 无效"})
		return
	}
	svc := device_story.NewService(ctrl.DB, ctrl.Cfg)
	result, err := svc.DeleteDeviceStory(c.Request.Context(), uint(deviceID), storyID)
	if err != nil {
		writeDeviceStoryError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": result, "ok": true})
}

func (ctrl *DeviceStoryController) ClearDeviceStories(c *gin.Context) {
	deviceID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || deviceID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "设备 ID 无效"})
		return
	}
	svc := device_story.NewService(ctrl.DB, ctrl.Cfg)
	result, err := svc.ClearDeviceStories(c.Request.Context(), uint(deviceID))
	if err != nil {
		writeDeviceStoryError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": result, "ok": true})
}

// MpGetDeviceStory 小程序按 story_id 拉取全文（需设备访问权限）。
func (ctrl *DeviceStoryController) MpGetDeviceStory(c *gin.Context) {
	userID, _ := c.Get("user_id")
	uid, _ := userID.(uint)
	deviceID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || deviceID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "设备 ID 无效"})
		return
	}
	var device models.Device
	if err := ctrl.DB.First(&device, uint(deviceID)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "设备不存在"})
		return
	}
	if !device_acl.CanAccess(ctrl.DB, device.ID, uid) {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权访问该设备"})
		return
	}
	ctrl.GetDeviceStory(c)
}

func writeDeviceStoryError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, device_story.ErrDeviceNotFound),
		errors.Is(err, device_story.ErrStoryNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	case errors.Is(err, device_story.ErrDeviceMissingSN),
		errors.Is(err, device_story.ErrRedisNotConfigured):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
}
