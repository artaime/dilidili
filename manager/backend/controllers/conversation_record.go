package controllers

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ConversationRecordController struct {
	DB                    *gorm.DB
	ChatHistoryController *ChatHistoryController
}

func (ctrl *ConversationRecordController) parseListQuery(c *gin.Context) (conversationRecordQuery, error) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	q := conversationRecordQuery{Limit: limit}

	if dateStr := strings.TrimSpace(c.Query("date")); dateStr != "" {
		day, err := time.ParseInLocation("2006-01-02", dateStr, time.Local)
		if err != nil {
			return q, err
		}
		q.Date = &day
		return q, nil
	}

	beforeID, _ := strconv.ParseUint(c.Query("before_id"), 10, 64)
	if beforeTime := strings.TrimSpace(c.Query("before_sort_time")); beforeTime != "" {
		cursor, err := parseConversationCursor(beforeTime, c.Query("before_type"), uint(beforeID))
		if err != nil {
			return q, err
		}
		q.Before = cursor
		return q, nil
	}

	afterID, _ := strconv.ParseUint(c.Query("after_id"), 10, 64)
	if afterTime := strings.TrimSpace(c.Query("after_sort_time")); afterTime != "" {
		cursor, err := parseConversationCursor(afterTime, c.Query("after_type"), uint(afterID))
		if err != nil {
			return q, err
		}
		q.After = cursor
		return q, nil
	}

	return q, nil
}

func (ctrl *ConversationRecordController) listByDevice(c *gin.Context, userID *uint) {
	deviceID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || deviceID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "设备 ID 无效"})
		return
	}

	device, err := loadDeviceForConversation(ctrl.DB, uint(deviceID), userID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "设备不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询设备失败"})
		return
	}
	if strings.TrimSpace(device.DeviceName) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "设备缺少 SN，无法查询对话记录"})
		return
	}

	query, err := ctrl.parseListQuery(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	items, hasOlder, hasNewer, err := listConversationRecords(ctrl.DB, device.DeviceName, device.ID, device.UserID, query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询对话记录失败"})
		return
	}

	resp := gin.H{
		"data":       items,
		"has_older":  hasOlder,
		"has_newer":  hasNewer,
		"device_id":  device.ID,
		"device_name": device.DeviceName,
	}
	if len(items) > 0 {
		first := items[0]
		last := items[len(items)-1]
		resp["oldest_cursor"] = gin.H{"sort_time": first.SortTime, "type": first.Type, "id": first.ID}
		resp["newest_cursor"] = gin.H{"sort_time": last.SortTime, "type": last.Type, "id": last.ID}
	}
	c.JSON(http.StatusOK, resp)
}

func (ctrl *ConversationRecordController) MpList(c *gin.Context) {
	userIDVal, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权"})
		return
	}
	uid, _ := userIDVal.(uint)
	ctrl.listByDevice(c, &uid)
}

func (ctrl *ConversationRecordController) AdminList(c *gin.Context) {
	ctrl.listByDevice(c, nil)
}

func (ctrl *ConversationRecordController) MpGetChatAudio(c *gin.Context) {
	userIDVal, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权"})
		return
	}
	uid, _ := userIDVal.(uint)
	serveChatMessageAudio(c, ctrl.DB, ctrl.ChatHistoryController.AudioBasePath, c.Param("id"), &uid)
}

func (ctrl *ConversationRecordController) AdminGetChatAudio(c *gin.Context) {
	serveChatMessageAudio(c, ctrl.DB, ctrl.ChatHistoryController.AudioBasePath, c.Param("id"), nil)
}

func (ctrl *ConversationRecordController) AdminGetParentAudio(c *gin.Context) {
	var msg struct {
		ID uint
	}
	if err := ctrl.DB.Table("parent_messages").Select("id").Where("id = ?", c.Param("id")).Scan(&msg).Error; err != nil || msg.ID == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "留言不存在"})
		return
	}
	var full struct {
		AudioPath  string
		SourceType string
	}
	if err := ctrl.DB.Table("parent_messages").Select("audio_path, source_type").Where("id = ?", msg.ID).Scan(&full).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "留言不存在"})
		return
	}
	if full.SourceType != "voice" || strings.TrimSpace(full.AudioPath) == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "音频不存在"})
		return
	}
	serveParentMessageAudio(c, full.AudioPath)
}
