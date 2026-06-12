package controllers

import (
	"fmt"
	"net/http"
	"strings"

	"xiaozhi/manager/backend/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ParentMessageInternalController struct {
	DB *gorm.DB
}

func (ctrl *ParentMessageInternalController) GetPendingMessage(c *gin.Context) {
	deviceName := strings.TrimSpace(c.Param("device_name"))
	if deviceName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "device_name 必填"})
		return
	}

	messages, err := findPendingParentMessages(ctrl.DB, deviceName)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusOK, gin.H{"data": []interface{}{}})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询留言失败"})
		return
	}
	if len(messages) == 0 {
		c.JSON(http.StatusOK, gin.H{"data": []interface{}{}})
		return
	}

	userIDs := make([]uint, 0, len(messages))
	for _, msg := range messages {
		userIDs = append(userIDs, msg.UserID)
	}
	userMap := loadUserFamilyRoleMap(ctrl.DB, userIDs)

	result := make([]gin.H, 0, len(messages))
	for _, msg := range messages {
		result = append(result, enrichPendingParentMessage(msg, userMap[msg.UserID]))
	}
	c.JSON(http.StatusOK, gin.H{"data": result})
}

func (ctrl *ParentMessageInternalController) GetMessageAudio(c *gin.Context) {
	var msg models.ParentMessage
	if err := ctrl.DB.Where("id = ?", c.Param("id")).First(&msg).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "留言不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询留言失败"})
		return
	}
	if msg.SourceType != "voice" || strings.TrimSpace(msg.AudioPath) == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "音频不存在"})
		return
	}
	serveParentMessageAudio(c, msg.AudioPath)
}

func (ctrl *ParentMessageInternalController) UpdateMessageStatus(c *gin.Context) {
	var req struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}
	status := strings.TrimSpace(req.Status)
	if !parentMessageStatusAllowed(status) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的状态"})
		return
	}

	var msg models.ParentMessage
	if err := ctrl.DB.Where("id = ?", c.Param("id")).First(&msg).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "留言不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询留言失败"})
		return
	}

	if err := updateParentMessageStatus(ctrl.DB, msg.ID, status); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新状态失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "状态已更新"})
}

func loadUserFamilyRoleMap(db *gorm.DB, userIDs []uint) map[uint]models.User {
	result := map[uint]models.User{}
	if len(userIDs) == 0 {
		return result
	}
	unique := make([]uint, 0, len(userIDs))
	seen := map[uint]struct{}{}
	for _, id := range userIDs {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		unique = append(unique, id)
	}
	var users []models.User
	_ = db.Where("id IN ?", unique).Find(&users).Error
	for _, user := range users {
		result[user.ID] = user
	}
	return result
}

func enrichPendingParentMessage(msg models.ParentMessage, user models.User) gin.H {
	familyRole := normalizeFamilyRole(user.FamilyRole)
	title := strings.TrimSpace(msg.Title)
	if title == "" {
		title = autoGenerateTitle(msg.SourceType, msg.CreatedAt)
	}
	item := gin.H{
		"id":           msg.ID,
		"user_id":      msg.UserID,
		"device_id":    msg.DeviceID,
		"title":        title,
		"text_content": msg.TextContent,
		"source_type":  msg.SourceType,
		"status":       msg.Status,
		"family_role":  familyRole,
		"created_at":   msg.CreatedAt,
	}
	if msg.SourceType == "voice" && strings.TrimSpace(msg.AudioPath) != "" {
		item["audio_url"] = fmt.Sprintf("/api/internal/parent-messages/%d/audio", msg.ID)
	}
	return item
}
