package controllers

import (
	"fmt"
	"net/http"
	"strings"

	"xiaozhi/manager/backend/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type MpDeviceController struct {
	DB *gorm.DB
}

func (ctrl *MpDeviceController) CheckDevice(c *gin.Context) {
	mac := strings.TrimSpace(c.Query("mac"))
	if mac == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "mac 参数必填"})
		return
	}

	device, found, err := ctrl.findDeviceByMAC(mac)
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
	bindable := !bound || device.UserID == currentUID

	resp := gin.H{
		"registered":  true,
		"bound":       bound,
		"bindable":    bindable,
		"device_id":   device.ID,
		"device_name": device.DeviceName,
		"nick_name":   device.NickName,
		"activated":   device.Activated,
	}
	if bound && device.UserID != currentUID {
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
		MAC           string `json:"mac" binding:"required"`
		ChildNickName string `json:"child_nick_name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}

	device, found, err := ctrl.findDeviceByMAC(req.MAC)
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

	childNick := strings.TrimSpace(req.ChildNickName)
	if childNick == "" {
		childNick = strings.TrimSpace(device.NickName)
	}
	if childNick == "" {
		childNick = "小朋友"
	}

	agent, err := ctrl.ensureDefaultAgent(currentUID, childNick)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	updates := map[string]interface{}{
		"user_id":   currentUID,
		"agent_id":  agent.ID,
		"activated": true,
	}
	if childNick != "" {
		updates["nick_name"] = childNick
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
	deviceID := c.Param("id")

	var device models.Device
	if err := ctrl.DB.Where("id = ? AND user_id = ?", deviceID, currentUID).First(&device).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "设备不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询设备失败"})
		return
	}

	updates := map[string]interface{}{
		"user_id":   0,
		"agent_id":  0,
		"activated": false,
	}
	if err := updateDeviceColumns(ctrl.DB, device.ID, updates); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "解绑失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "解绑成功"})
}

func (ctrl *MpDeviceController) findDeviceByMAC(mac string) (models.Device, bool, error) {
	normalized := normalizeDeviceNameCandidate(mac)
	var device models.Device
	err := ctrl.DB.Where("LOWER(REPLACE(device_name, '-', ':')) = ?", normalized).First(&device).Error
	if err == gorm.ErrRecordNotFound {
		return models.Device{}, false, nil
	}
	if err != nil {
		return models.Device{}, false, err
	}
	return device, true, nil
}

func (ctrl *MpDeviceController) ensureDefaultAgent(userID uint, childNick string) (models.Agent, error) {
	var existing models.Agent
	agentName := fmt.Sprintf("%s的小伙伴", childNick)
	if err := ctrl.DB.Where("user_id = ? AND name = ?", userID, agentName).First(&existing).Error; err == nil {
		return existing, nil
	} else if err != gorm.ErrRecordNotFound {
		return models.Agent{}, err
	}

	llmID, ttsID := getDefaultConfigIDs(ctrl.DB)
	llmPtr := stringPtrOrNil(llmID)
	ttsPtr := stringPtrOrNil(ttsID)

	agentSvc := NewAgentService(ctrl.DB)
	resp, err := agentSvc.Create(accessScope{ActorUserID: userID}, AgentPayload{
		UserID:      userID,
		Name:        agentName,
		Nickname:    &agentName,
		CustomPrompt: "你是一个温暖、有趣的儿童 AI 伙伴，用简短、亲切的语言和孩子交流。",
		LLMConfigID: llmPtr,
		TTSConfigID: ttsPtr,
	})
	if err != nil {
		return models.Agent{}, err
	}
	return resp.Agent, nil
}

func getDefaultConfigIDs(db *gorm.DB) (llmID, ttsID string) {
	var llmCfg models.Config
	if err := db.Where("type = ? AND is_default = ? AND enabled = ?", "llm", true, true).First(&llmCfg).Error; err == nil {
		llmID = llmCfg.ConfigID
	}
	var ttsCfg models.Config
	if err := db.Where("type = ? AND is_default = ? AND enabled = ?", "tts", true, true).First(&ttsCfg).Error; err == nil {
		ttsID = ttsCfg.ConfigID
	}
	return llmID, ttsID
}

func stringPtrOrNil(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}
