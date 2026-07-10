package controllers

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"dili/manager/backend/models"
	"dili/manager/backend/services/device_reset"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type MpDeviceController struct {
	DB           *gorm.DB
	ResetService *device_reset.Service
}

func (ctrl *MpDeviceController) CheckDevice(c *gin.Context) {
	sn := normalizeDeviceSN(c.Query("sn"))
	if sn == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "sn 参数必填"})
		return
	}

	device, found, err := ctrl.findDeviceBySN(sn)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询设备失败"})
		return
	}
	if !found {
		c.JSON(http.StatusOK, gin.H{
			"registered": false,
			"bound":      false,
			"bindable":   false,
			"message":    "设备未登记，请联系厂商",
		})
		return
	}

	userID, _ := c.Get("user_id")
	currentUID, _ := userID.(uint)
	bound := device.UserID != 0
	hasAgent := device.AgentID > 0
	bindable := hasAgent && (!bound || device.UserID == currentUID)

	resp := gin.H{
		"registered":  true,
		"bound":       bound,
		"bindable":    bindable,
		"has_agent":   hasAgent,
		"device_id":   device.ID,
		"device_name": device.DeviceName,
		"nick_name":   device.NickName,
		"activated":   device.Activated,
	}
	if !hasAgent {
		resp["message"] = "设备未绑定智能体，请联系厂商"
	} else if bound && device.UserID != currentUID {
		resp["message"] = "设备已被其他家长绑定"
	}
	c.JSON(http.StatusOK, resp)
}

func (ctrl *MpDeviceController) BindDevice(c *gin.Context) {
	userID, _ := c.Get("user_id")
	currentUID, _ := userID.(uint)
	if currentUID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未认证"})
		return
	}

	var req struct {
		SN            string `json:"sn" binding:"required"`
		ChildNickName string `json:"child_nick_name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}

	device, found, err := ctrl.findDeviceBySN(req.SN)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询设备失败"})
		return
	}
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "设备未登记，请联系厂商"})
		return
	}
	if device.UserID != 0 && device.UserID != currentUID {
		c.JSON(http.StatusConflict, gin.H{"error": "设备已被其他家长绑定"})
		return
	}
	if device.AgentID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "设备未绑定智能体，请联系厂商"})
		return
	}

	var agent models.Agent
	if err := ctrl.DB.Where("id = ?", device.AgentID).First(&agent).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusBadRequest, gin.H{"error": "设备未绑定智能体，请联系厂商"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询智能体失败"})
		return
	}

	childNick := strings.TrimSpace(req.ChildNickName)
	if childNick == "" {
		childNick = strings.TrimSpace(device.NickName)
	}
	if childNick == "" {
		childNick = "小朋友"
	}

	updates := map[string]interface{}{
		"user_id":   currentUID,
		"activated": true,
		"nick_name": childNick,
	}
	if err := updateDeviceColumns(ctrl.DB, device.ID, updates); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "绑定设备失败"})
		return
	}

	deviceSvc := NewDeviceService(ctrl.DB)
	result, err := deviceSvc.Get(accessScope{ActorUserID: currentUID}, device.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取设备信息失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "绑定成功",
		"data": gin.H{
			"device": result,
			"agent": gin.H{
				"id":       agent.ID,
				"name":     agent.Name,
				"nickname": agent.Nickname,
			},
		},
	})
}

func (ctrl *MpDeviceController) ListDevices(c *gin.Context) {
	userID, _ := c.Get("user_id")
	currentUID, _ := userID.(uint)
	result, err := NewDeviceService(ctrl.DB).List(accessScope{ActorUserID: currentUID})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取设备列表失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": result})
}

func (ctrl *MpDeviceController) UnbindDevice(c *gin.Context) {
	userID, _ := c.Get("user_id")
	currentUID, _ := userID.(uint)
	deviceID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || deviceID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "设备 ID 无效"})
		return
	}

	if ctrl.ResetService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "解绑服务未配置"})
		return
	}

	if err := ctrl.ResetService.ResetToFactory(c.Request.Context(), uint(deviceID), currentUID); err != nil {
		switch {
		case errors.Is(err, device_reset.ErrDeviceNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "设备不存在"})
		case errors.Is(err, device_reset.ErrForbidden):
			c.JSON(http.StatusForbidden, gin.H{"error": "无权解绑该设备"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "解绑失败: " + err.Error()})
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "解绑成功"})
}

func (ctrl *MpDeviceController) findDeviceBySN(sn string) (models.Device, bool, error) {
	normalized := normalizeDeviceSN(sn)
	if normalized == "" {
		return models.Device{}, false, nil
	}
	var device models.Device
	err := ctrl.DB.Where("device_name = ?", normalized).First(&device).Error
	if err == gorm.ErrRecordNotFound {
		return models.Device{}, false, nil
	}
	if err != nil {
		return models.Device{}, false, err
	}
	return device, true, nil
}
