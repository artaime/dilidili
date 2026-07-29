package controllers

import (
	"errors"
	"net/http"
	"strconv"

	"dili/manager/backend/services/device_memory"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type DeviceMemoryController struct {
	DB *gorm.DB
}

func (ctrl *DeviceMemoryController) GetDeviceMemory(c *gin.Context) {
	deviceID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || deviceID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "设备 ID 无效"})
		return
	}
	if writeAdminPrivacyGateError(c, assertDeviceBoundToActor(ctrl.DB, uint(deviceID), actorUserIDFromContext(c))) {
		return
	}

	svc := device_memory.NewService(ctrl.DB)
	view, err := svc.GetDeviceMemory(c.Request.Context(), uint(deviceID))
	if err != nil {
		writeDeviceMemoryError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": view})
}

func (ctrl *DeviceMemoryController) DeleteDeviceMemory(c *gin.Context) {
	deviceID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || deviceID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "设备 ID 无效"})
		return
	}
	if writeAdminPrivacyGateError(c, assertDeviceBoundToActor(ctrl.DB, uint(deviceID), actorUserIDFromContext(c))) {
		return
	}

	svc := device_memory.NewService(ctrl.DB)
	if err := svc.DeleteDeviceMemory(c.Request.Context(), uint(deviceID)); err != nil {
		writeDeviceMemoryError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "设备长期记忆已清空"})
}

func writeDeviceMemoryError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, device_memory.ErrDeviceNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	case errors.Is(err, device_memory.ErrDeviceMissingSN),
		errors.Is(err, device_memory.ErrLongMemoryDisabled),
		errors.Is(err, device_memory.ErrMemobaseNotConfigured):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
}
