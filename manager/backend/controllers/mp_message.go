package controllers

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"xiaozhi/manager/backend/config"
	"xiaozhi/manager/backend/models"
	"xiaozhi/manager/backend/storage"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type MpMessageController struct {
	DB           *gorm.DB
	Cfg          *config.Config
	AudioStorage *storage.AudioStorage
}

func NewMpMessageController(db *gorm.DB, cfg *config.Config) *MpMessageController {
	basePath := cfg.ParentMessage.AudioBasePath
	maxSize := cfg.ParentMessage.MaxFileSize
	if basePath == "" {
		basePath = "./storage/parent_messages/audio"
	}
	if maxSize <= 0 {
		maxSize = 10 * 1024 * 1024
	}
	return &MpMessageController{
		DB:           db,
		Cfg:          cfg,
		AudioStorage: storage.NewAudioStorage(basePath, maxSize),
	}
}

func (ctrl *MpMessageController) CreateMessage(c *gin.Context) {
	userID, _ := c.Get("user_id")
	currentUID, _ := userID.(uint)

	contentType := c.GetHeader("Content-Type")
	if strings.Contains(contentType, "multipart/form-data") {
		ctrl.createMessageMultipart(c, currentUID)
		return
	}

	var req struct {
		DeviceID    uint   `json:"device_id" binding:"required"`
		TextContent string `json:"text_content" binding:"required"`
		SourceType  string `json:"source_type"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}
	sourceType := strings.TrimSpace(req.SourceType)
	if sourceType == "" {
		sourceType = "text"
	}
	ctrl.saveMessage(c, currentUID, req.DeviceID, strings.TrimSpace(req.TextContent), "", sourceType)
}

func (ctrl *MpMessageController) createMessageMultipart(c *gin.Context, userID uint) {
	deviceIDStr := strings.TrimSpace(c.PostForm("device_id"))
	textContent := strings.TrimSpace(c.PostForm("text_content"))
	sourceType := strings.TrimSpace(c.PostForm("source_type"))
	if sourceType == "" {
		sourceType = "voice"
	}

	var deviceID uint
	if _, err := fmt.Sscanf(deviceIDStr, "%d", &deviceID); err != nil || deviceID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "device_id 无效"})
		return
	}

	file, header, err := c.Request.FormFile("audio")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请上传音频文件"})
		return
	}
	defer file.Close()

	audioPath, _, err := ctrl.AudioStorage.SaveVoiceCloneAudioFile(userID, uuid.NewString(), header.Filename, file)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if textContent == "" {
		if transcribed, err := transcribeAudioFile(ctrl.Cfg.SpeakerService.URL, audioPath); err == nil {
			textContent = transcribed
		}
	}
	if textContent == "" {
		_ = ctrl.AudioStorage.DeleteAudioFile(audioPath)
		c.JSON(http.StatusBadRequest, gin.H{"error": "未能识别语音内容，请补充文字或重录"})
		return
	}

	ctrl.saveMessage(c, userID, deviceID, textContent, audioPath, sourceType)
}

func (ctrl *MpMessageController) saveMessage(c *gin.Context, userID, deviceID uint, textContent, audioPath, sourceType string) {
	if textContent == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "留言内容不能为空"})
		return
	}
	if len([]rune(textContent)) > 500 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "留言内容不能超过 500 字"})
		return
	}

	var device models.Device
	if err := ctrl.DB.Where("id = ? AND user_id = ?", deviceID, userID).First(&device).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusBadRequest, gin.H{"error": "设备不存在或不属于当前用户"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询设备失败"})
		return
	}

	msg := models.ParentMessage{
		UserID:      userID,
		DeviceID:    device.ID,
		TextContent: textContent,
		AudioPath:   audioPath,
		SourceType:  sourceType,
		Status:      "pending",
	}
	if err := ctrl.DB.Create(&msg).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建留言失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "留言已发送，设备上线后将通知孩子收听",
		"data":    enrichParentMessage(msg, device),
	})
}

func (ctrl *MpMessageController) ListMessages(c *gin.Context) {
	userID, _ := c.Get("user_id")
	currentUID, _ := userID.(uint)

	query := ctrl.DB.Where("user_id = ?", currentUID).Order("id DESC")
	if deviceID := strings.TrimSpace(c.Query("device_id")); deviceID != "" {
		query = query.Where("device_id = ?", deviceID)
	}

	var messages []models.ParentMessage
	if err := query.Find(&messages).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取留言列表失败"})
		return
	}

	deviceIDs := make([]uint, 0, len(messages))
	for _, msg := range messages {
		deviceIDs = append(deviceIDs, msg.DeviceID)
	}
	deviceMap := ctrl.loadDeviceMap(deviceIDs)

	result := make([]gin.H, 0, len(messages))
	for _, msg := range messages {
		device := deviceMap[msg.DeviceID]
		result = append(result, enrichParentMessage(msg, device))
	}
	c.JSON(http.StatusOK, gin.H{"data": result})
}

func (ctrl *MpMessageController) GetMessage(c *gin.Context) {
	userID, _ := c.Get("user_id")
	currentUID, _ := userID.(uint)

	var msg models.ParentMessage
	if err := ctrl.DB.Where("id = ? AND user_id = ?", c.Param("id"), currentUID).First(&msg).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "留言不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询留言失败"})
		return
	}

	var device models.Device
	_ = ctrl.DB.Where("id = ?", msg.DeviceID).First(&device).Error
	c.JSON(http.StatusOK, gin.H{"data": enrichParentMessage(msg, device)})
}

func (ctrl *MpMessageController) DeleteMessage(c *gin.Context) {
	userID, _ := c.Get("user_id")
	currentUID, _ := userID.(uint)

	var msg models.ParentMessage
	if err := ctrl.DB.Where("id = ? AND user_id = ?", c.Param("id"), currentUID).First(&msg).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "留言不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询留言失败"})
		return
	}
	if msg.Status != "pending" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "仅可撤回待播放的留言"})
		return
	}

	if msg.AudioPath != "" {
		_ = ctrl.AudioStorage.DeleteAudioFile(msg.AudioPath)
	}
	if err := ctrl.DB.Delete(&msg).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除留言失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "留言已撤回"})
}

func (ctrl *MpMessageController) loadDeviceMap(deviceIDs []uint) map[uint]models.Device {
	result := map[uint]models.Device{}
	if len(deviceIDs) == 0 {
		return result
	}
	unique := make([]uint, 0, len(deviceIDs))
	seen := map[uint]struct{}{}
	for _, id := range deviceIDs {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		unique = append(unique, id)
	}
	var devices []models.Device
	_ = ctrl.DB.Where("id IN ?", unique).Find(&devices).Error
	for _, device := range devices {
		result[device.ID] = device
	}
	return result
}

func enrichParentMessage(msg models.ParentMessage, device models.Device) gin.H {
	statusText := map[string]string{
		"pending":  "待播放",
		"notified": "已通知",
		"played":   "已播放",
		"skipped":  "已跳过",
		"expired":  "已过期",
	}
	return gin.H{
		"id":           msg.ID,
		"device_id":    msg.DeviceID,
		"device_name":  device.DeviceName,
		"device_nick":  device.NickName,
		"text_content": msg.TextContent,
		"source_type":  msg.SourceType,
		"status":       msg.Status,
		"status_text":  statusText[msg.Status],
		"has_audio":    msg.AudioPath != "",
		"created_at":   msg.CreatedAt,
		"played_at":    msg.PlayedAt,
	}
}

func parentMessageStatusAllowed(status string) bool {
	switch status {
	case "notified", "played", "skipped", "expired":
		return true
	default:
		return false
	}
}

func updateParentMessageStatus(db *gorm.DB, id uint, status string) error {
	updates := map[string]interface{}{"status": status}
	if status == "played" {
		now := time.Now()
		updates["played_at"] = &now
	}
	return db.Model(&models.ParentMessage{}).Where("id = ?", id).Updates(updates).Error
}

func findPendingParentMessage(db *gorm.DB, deviceName string) (*models.ParentMessage, error) {
	var device models.Device
	normalized := normalizeDeviceNameCandidate(deviceName)
	if err := db.Where("LOWER(REPLACE(device_name, '-', ':')) = ?", normalized).First(&device).Error; err != nil {
		return nil, err
	}
	var msg models.ParentMessage
	if err := db.Where("device_id = ? AND status = ?", device.ID, "pending").
		Order("id ASC").First(&msg).Error; err != nil {
		return nil, err
	}
	return &msg, nil
}

func serveParentMessageAudio(c *gin.Context, audioPath string) {
	if strings.TrimSpace(audioPath) == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "音频不存在"})
		return
	}
	if _, err := os.Stat(audioPath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "音频不存在"})
		return
	}
	c.File(audioPath)
}
