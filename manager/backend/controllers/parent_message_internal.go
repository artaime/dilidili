package controllers

import (
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

	msg, err := findPendingParentMessage(ctrl.DB, deviceName)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusOK, gin.H{"data": nil})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询留言失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"id":           msg.ID,
			"user_id":      msg.UserID,
			"device_id":    msg.DeviceID,
			"text_content": msg.TextContent,
			"source_type":  msg.SourceType,
			"status":       msg.Status,
		},
	})
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
