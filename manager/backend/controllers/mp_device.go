package controllers

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"dili/manager/backend/models"
	"dili/manager/backend/services/device_acl"
	"dili/manager/backend/services/device_reset"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type MpDeviceController struct {
	DB           *gorm.DB
	ResetService *device_reset.Service
}

type mpDeviceListItem struct {
	DeviceResponse
	MyRole string `json:"my_role"`
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
	isMember := device_acl.CanAccess(ctrl.DB, device.ID, currentUID) && device.UserID != currentUID
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
		"is_member":   isMember,
	}
	if !hasAgent {
		resp["message"] = "设备未绑定智能体，请联系厂商"
	} else if isMember {
		resp["bindable"] = false
		resp["message"] = "已是家庭成员，无需重复绑定"
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
		if device_acl.CanAccess(ctrl.DB, device.ID, currentUID) {
			c.JSON(http.StatusConflict, gin.H{"error": "已是家庭成员，无需重复绑定"})
			return
		}
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

	err = ctrl.DB.Transaction(func(tx *gorm.DB) error {
		updates := map[string]interface{}{
			"user_id":   currentUID,
			"activated": true,
			"nick_name": childNick,
		}
		if err := updateDeviceColumns(tx, device.ID, updates); err != nil {
			return err
		}
		return device_acl.EnsureOwnerMember(tx, device.ID, currentUID)
	})
	if err != nil {
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

	ids, err := device_acl.ListAccessibleDeviceIDs(ctrl.DB, currentUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取设备列表失败"})
		return
	}
	if len(ids) == 0 {
		c.JSON(http.StatusOK, gin.H{"data": []mpDeviceListItem{}})
		return
	}

	var devices []models.Device
	if err := ctrl.DB.Where("id IN ?", ids).Order("id DESC").Find(&devices).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取设备列表失败"})
		return
	}

	enriched, err := NewDeviceService(ctrl.DB).enrichDevices(accessScope{ActorUserID: currentUID, IsAdmin: true}, devices)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取设备列表失败"})
		return
	}

	result := make([]mpDeviceListItem, 0, len(enriched))
	for _, d := range enriched {
		result = append(result, mpDeviceListItem{
			DeviceResponse: d,
			MyRole:         device_acl.MemberRole(ctrl.DB, d.ID, currentUID),
		})
	}
	c.JSON(http.StatusOK, gin.H{"data": result})
}

func (ctrl *MpDeviceController) UpdateDevice(c *gin.Context) {
	userID, _ := c.Get("user_id")
	currentUID, _ := userID.(uint)
	deviceID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || deviceID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "设备 ID 无效"})
		return
	}
	if err := device_acl.AssertManage(ctrl.DB, uint(deviceID), currentUID); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "仅属主可修改孩子昵称"})
		return
	}

	var req struct {
		NickName string `json:"nick_name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}
	nickName, err := normalizeDeviceNickName(req.NickName)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if nickName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "孩子昵称不能为空"})
		return
	}
	if err := updateDeviceColumns(ctrl.DB, uint(deviceID), map[string]interface{}{"nick_name": nickName}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失败"})
		return
	}

	result, err := NewDeviceService(ctrl.DB).Get(accessScope{ActorUserID: currentUID, IsAdmin: true}, uint(deviceID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取设备信息失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": "已更新",
		"data": mpDeviceListItem{
			DeviceResponse: *result,
			MyRole:         device_acl.RoleOwner,
		},
	})
}

func (ctrl *MpDeviceController) CreateInvite(c *gin.Context) {
	userID, _ := c.Get("user_id")
	currentUID, _ := userID.(uint)
	deviceID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || deviceID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "设备 ID 无效"})
		return
	}

	invite, err := device_acl.CreateInvite(ctrl.DB, uint(deviceID), currentUID)
	if err != nil {
		writeDeviceACLError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"code":       invite.Code,
			"expires_at": invite.ExpiresAt,
			"max_uses":   invite.MaxUses,
			"used_count": invite.UsedCount,
		},
	})
}

func (ctrl *MpDeviceController) JoinDevice(c *gin.Context) {
	userID, _ := c.Get("user_id")
	currentUID, _ := userID.(uint)
	if currentUID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未认证"})
		return
	}
	var req struct {
		Code string `json:"code" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请输入邀请码"})
		return
	}

	device, err := device_acl.JoinByCode(ctrl.DB, currentUID, req.Code)
	if err != nil {
		writeDeviceACLError(c, err)
		return
	}
	result, err := NewDeviceService(ctrl.DB).Get(accessScope{ActorUserID: currentUID, IsAdmin: true}, device.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "加入成功但获取设备失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": "已加入家庭",
		"data": mpDeviceListItem{
			DeviceResponse: *result,
			MyRole:         device_acl.RoleMember,
		},
	})
}

func (ctrl *MpDeviceController) ListMembers(c *gin.Context) {
	userID, _ := c.Get("user_id")
	currentUID, _ := userID.(uint)
	deviceID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || deviceID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "设备 ID 无效"})
		return
	}
	if err := device_acl.AssertAccess(ctrl.DB, uint(deviceID), currentUID); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权查看该设备成员"})
		return
	}

	members, err := device_acl.ListMembers(ctrl.DB, uint(deviceID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取成员列表失败"})
		return
	}

	userIDs := make([]uint, 0, len(members))
	for _, m := range members {
		userIDs = append(userIDs, m.UserID)
	}
	userMap := map[uint]models.User{}
	if len(userIDs) > 0 {
		var users []models.User
		_ = ctrl.DB.Where("id IN ?", userIDs).Find(&users).Error
		for _, u := range users {
			userMap[u.ID] = u
		}
	}

	result := make([]gin.H, 0, len(members))
	for _, m := range members {
		u := userMap[m.UserID]
		displayName := strings.TrimSpace(u.Nickname)
		if displayName == "" {
			displayName = u.Username
		}
		result = append(result, gin.H{
			"user_id":     m.UserID,
			"role":        m.Role,
			"nickname":    displayName,
			"avatar_url":  u.AvatarURL,
			"family_role": u.FamilyRole,
			"joined_at":   m.JoinedAt,
			"is_self":     m.UserID == currentUID,
		})
	}
	c.JSON(http.StatusOK, gin.H{
		"data":     result,
		"my_role":  device_acl.MemberRole(ctrl.DB, uint(deviceID), currentUID),
		"max":      device_acl.MaxMembersPerDevice,
		"count":    len(result),
	})
}

func (ctrl *MpDeviceController) RemoveMember(c *gin.Context) {
	userID, _ := c.Get("user_id")
	currentUID, _ := userID.(uint)
	deviceID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || deviceID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "设备 ID 无效"})
		return
	}
	targetID, err := strconv.ParseUint(c.Param("userId"), 10, 64)
	if err != nil || targetID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "用户 ID 无效"})
		return
	}
	if err := device_acl.RevokeMember(ctrl.DB, uint(deviceID), currentUID, uint(targetID)); err != nil {
		writeDeviceACLError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "已移除该成员"})
}

func (ctrl *MpDeviceController) LeaveDevice(c *gin.Context) {
	userID, _ := c.Get("user_id")
	currentUID, _ := userID.(uint)
	deviceID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || deviceID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "设备 ID 无效"})
		return
	}
	if err := device_acl.LeaveDevice(ctrl.DB, uint(deviceID), currentUID); err != nil {
		writeDeviceACLError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "已退出家庭"})
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

func writeDeviceACLError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, device_acl.ErrForbidden),
		errors.Is(err, device_acl.ErrOwnerOnly):
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
	case errors.Is(err, device_acl.ErrNotMember),
		errors.Is(err, device_acl.ErrInviteInvalid):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case errors.Is(err, device_acl.ErrAlreadyMember),
		errors.Is(err, device_acl.ErrMemberFull),
		errors.Is(err, device_acl.ErrCannotKickOwner),
		errors.Is(err, device_acl.ErrCannotLeaveOwner):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	case errors.Is(err, gorm.ErrRecordNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "设备不存在"})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
}
