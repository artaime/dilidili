package controllers

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"dili/manager/backend/config"
	"dili/manager/backend/models"
	"dili/manager/backend/privacy"
	"dili/manager/backend/services/device_acl"
	"dili/manager/backend/storage"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const maxParentMessageDurationSec = 60

type MpMessageController struct {
	DB           *gorm.DB
	Cfg          *config.Config
	AudioStorage *storage.AudioStorage
	Privacy      *privacy.Service
}

func NewMpMessageController(db *gorm.DB, cfg *config.Config, priv *privacy.Service) *MpMessageController {
	basePath := cfg.ParentMessage.AudioBasePath
	maxSize := cfg.ParentMessage.MaxFileSize
	if basePath == "" {
			basePath = "./data/parent_messages/audio"
	}
	if maxSize <= 0 {
		maxSize = 10 * 1024 * 1024
	}
	return &MpMessageController{
		DB:           db,
		Cfg:          cfg,
		AudioStorage: storage.NewAudioStorage(basePath, maxSize),
		Privacy:      priv,
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
		Title       string `json:"title"`
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
	textContent := sanitizeMessageText(req.TextContent)
	ctrl.saveMessage(c, currentUID, saveMessageInput{
		DeviceID:    req.DeviceID,
		TextContent: textContent,
		Title:       req.Title,
		SourceType:  sourceType,
	})
}

type saveMessageInput struct {
	DeviceID         uint
	TextContent      string
	Title            string
	SourceType       string
	AudioPath        string
	AudioDurationSec int
}

func (ctrl *MpMessageController) createMessageMultipart(c *gin.Context, userID uint) {
	deviceIDStr := strings.TrimSpace(c.PostForm("device_id"))
	title := strings.TrimSpace(c.PostForm("title"))
	sourceType := strings.TrimSpace(c.PostForm("source_type"))
	if sourceType == "" {
		sourceType = "voice"
	}

	var deviceID uint
	if _, err := fmt.Sscanf(deviceIDStr, "%d", &deviceID); err != nil || deviceID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "device_id 无效"})
		return
	}

	audioDurationSec := 0
	if raw := strings.TrimSpace(c.PostForm("audio_duration_sec")); raw != "" {
		if v, err := strconv.Atoi(raw); err == nil {
			audioDurationSec = v
		}
	}
	if audioDurationSec > maxParentMessageDurationSec {
		c.JSON(http.StatusBadRequest, gin.H{"error": "录音时长不能超过 60 秒"})
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

	ctrl.saveMessage(c, userID, saveMessageInput{
		DeviceID:         deviceID,
		Title:            title,
		SourceType:       sourceType,
		AudioPath:        audioPath,
		AudioDurationSec: audioDurationSec,
	})
}

func (ctrl *MpMessageController) saveMessage(c *gin.Context, userID uint, input saveMessageInput) {
	sourceType := strings.TrimSpace(input.SourceType)
	if sourceType == "" {
		sourceType = "text"
	}

	switch sourceType {
	case "text":
		if input.TextContent == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "留言内容不能为空"})
			return
		}
		if len([]rune(input.TextContent)) > 500 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "留言内容不能超过 500 字"})
			return
		}
	case "voice":
		if strings.TrimSpace(input.AudioPath) == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "请上传音频文件"})
			return
		}
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的留言类型"})
		return
	}

	var device models.Device
	if err := ctrl.DB.Where("id = ?", input.DeviceID).First(&device).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusBadRequest, gin.H{"error": "设备不存在或不属于当前用户"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询设备失败"})
		return
	}
	if !device_acl.CanAccess(ctrl.DB, device.ID, userID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "设备不存在或不属于当前用户"})
		return
	}

	now := time.Now()
	textContent := input.TextContent
	if enc, err := ctrl.encryptParentText(device.ID, textContent); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "加密留言失败"})
		return
	} else {
		textContent = enc
	}

	if sourceType == "voice" && strings.TrimSpace(input.AudioPath) != "" {
		if err := ctrl.encryptParentAudioFile(device.ID, input.AudioPath); err != nil {
			_ = ctrl.AudioStorage.DeleteAudioFile(input.AudioPath)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "加密留言音频失败"})
			return
		}
	}

	msg := models.ParentMessage{
		UserID:           userID,
		DeviceID:         device.ID,
		Title:            resolveMessageTitle(input.Title, sourceType, now),
		TextContent:      textContent,
		AudioPath:        input.AudioPath,
		AudioDurationSec: input.AudioDurationSec,
		SourceType:       sourceType,
		Status:           "pending",
		CreatedAt:        now,
	}
	if err := ctrl.DB.Create(&msg).Error; err != nil {
		if input.AudioPath != "" {
			_ = ctrl.AudioStorage.DeleteAudioFile(input.AudioPath)
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建留言失败"})
		return
	}

	ctrl.decryptParentMessage(&msg)
	c.JSON(http.StatusOK, gin.H{
		"message": "留言已发送，设备上线后将通知孩子收听",
		"data":    enrichParentMessage(msg, device),
	})
}

func (ctrl *MpMessageController) ListMessages(c *gin.Context) {
	userID, _ := c.Get("user_id")
	currentUID, _ := userID.(uint)

	query := ctrl.DB.Model(&models.ParentMessage{}).Order("id DESC")
	if deviceID := strings.TrimSpace(c.Query("device_id")); deviceID != "" {
		var did uint
		if _, err := fmt.Sscanf(deviceID, "%d", &did); err != nil || did == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "device_id 无效"})
			return
		}
		if !device_acl.CanAccess(ctrl.DB, did, currentUID) {
			c.JSON(http.StatusNotFound, gin.H{"error": "设备不存在或不属于当前用户"})
			return
		}
		query = query.Where("device_id = ?", did)
	} else {
		accessibleIDs, err := device_acl.ListAccessibleDeviceIDs(ctrl.DB, currentUID)
		if err != nil || len(accessibleIDs) == 0 {
			c.JSON(http.StatusOK, gin.H{"data": []map[string]interface{}{}})
			return
		}
		query = query.Where("device_id IN ?", accessibleIDs)
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

	result := make([]map[string]interface{}, 0, len(messages))
	for i := range messages {
		ctrl.decryptParentMessage(&messages[i])
		device := deviceMap[messages[i].DeviceID]
		result = append(result, enrichParentMessage(messages[i], device))
	}
	c.JSON(http.StatusOK, gin.H{"data": result})
}

func (ctrl *MpMessageController) GetMessageAudio(c *gin.Context) {
	userID, _ := c.Get("user_id")
	currentUID, _ := userID.(uint)

	var msg models.ParentMessage
	if err := ctrl.DB.Where("id = ?", c.Param("id")).First(&msg).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "留言不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询留言失败"})
		return
	}
	if !device_acl.CanAccess(ctrl.DB, msg.DeviceID, currentUID) {
		c.JSON(http.StatusNotFound, gin.H{"error": "留言不存在"})
		return
	}
	if msg.SourceType != "voice" || strings.TrimSpace(msg.AudioPath) == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "音频不存在"})
		return
	}
	serveParentMessageAudio(c, ctrl.Privacy, msg.DeviceID, msg.AudioPath)
}

func (ctrl *MpMessageController) GetMessage(c *gin.Context) {
	userID, _ := c.Get("user_id")
	currentUID, _ := userID.(uint)

	var msg models.ParentMessage
	if err := ctrl.DB.Where("id = ?", c.Param("id")).First(&msg).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "留言不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询留言失败"})
		return
	}
	if !device_acl.CanAccess(ctrl.DB, msg.DeviceID, currentUID) {
		c.JSON(http.StatusNotFound, gin.H{"error": "留言不存在"})
		return
	}

	var device models.Device
	_ = ctrl.DB.Where("id = ?", msg.DeviceID).First(&device).Error
	ctrl.decryptParentMessage(&msg)
	c.JSON(http.StatusOK, gin.H{"data": enrichParentMessage(msg, device)})
}

func (ctrl *MpMessageController) DeleteMessage(c *gin.Context) {
	userID, _ := c.Get("user_id")
	currentUID, _ := userID.(uint)

	var msg models.ParentMessage
	if err := ctrl.DB.Where("id = ?", c.Param("id")).First(&msg).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "留言不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询留言失败"})
		return
	}
	if !device_acl.CanAccess(ctrl.DB, msg.DeviceID, currentUID) {
		c.JSON(http.StatusNotFound, gin.H{"error": "留言不存在"})
		return
	}
	if msg.AudioPath != "" {
		_ = ctrl.AudioStorage.DeleteAudioFile(msg.AudioPath)
	}
	if err := ctrl.DB.Delete(&msg).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除留言失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "留言已删除"})
}

func (ctrl *MpMessageController) encryptParentText(deviceDBID uint, text string) (string, error) {
	if ctrl.Privacy == nil {
		return text, nil
	}
	return ctrl.Privacy.EncryptText(deviceDBID, text)
}

func (ctrl *MpMessageController) encryptParentAudioFile(deviceDBID uint, audioPath string) error {
	if ctrl.Privacy == nil || !ctrl.Privacy.Enabled() {
		return nil
	}
	data, err := os.ReadFile(audioPath)
	if err != nil {
		return err
	}
	enc, err := ctrl.Privacy.EncryptFileBytes(deviceDBID, data)
	if err != nil {
		return err
	}
	return os.WriteFile(audioPath, enc, 0644)
}

func (ctrl *MpMessageController) decryptParentMessage(msg *models.ParentMessage) {
	if msg == nil || ctrl.Privacy == nil {
		return
	}
	plain, err := ctrl.Privacy.DecryptText(msg.DeviceID, msg.TextContent)
	if err != nil {
		log.Printf("解密留言失败 id=%d: %v", msg.ID, err)
		return
	}
	msg.TextContent = plain
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

func parentMessageStatusAllowed(status string) bool {
	switch status {
	case "notified", "played", "expired":
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

func findSelectableParentMessages(db *gorm.DB, deviceName string, start, end *time.Time, limit int) ([]models.ParentMessage, error) {
	if limit <= 0 {
		limit = 100
	}
	var device models.Device
	if err := db.Where("device_name = ?", normalizeDeviceSN(deviceName)).First(&device).Error; err != nil {
		return nil, err
	}
	query := db.Where("device_id = ? AND status IN ?", device.ID, []string{"pending", "notified", "played"})
	if start != nil {
		query = query.Where("created_at >= ?", *start)
	}
	if end != nil {
		query = query.Where("created_at <= ?", *end)
	}
	var messages []models.ParentMessage
	if err := query.Order("created_at DESC").Limit(limit).Find(&messages).Error; err != nil {
		return nil, err
	}
	return messages, nil
}

func findPlayedParentMessages(db *gorm.DB, deviceName string, limit int) ([]models.ParentMessage, error) {
	if limit <= 0 {
		limit = 20
	}
	var device models.Device
	if err := db.Where("device_name = ?", normalizeDeviceSN(deviceName)).First(&device).Error; err != nil {
		return nil, err
	}
	var messages []models.ParentMessage
	if err := db.Where("device_id = ? AND status = ?", device.ID, "played").
		Order("played_at DESC, id DESC").Limit(limit).Find(&messages).Error; err != nil {
		return nil, err
	}
	return messages, nil
}

func findPendingParentMessages(db *gorm.DB, deviceName string) ([]models.ParentMessage, error) {
	var device models.Device
	if err := db.Where("device_name = ?", normalizeDeviceSN(deviceName)).First(&device).Error; err != nil {
		return nil, err
	}
	var messages []models.ParentMessage
	if err := db.Where("device_id = ? AND status IN ?", device.ID, []string{"pending", "notified"}).
		Order("id ASC").Find(&messages).Error; err != nil {
		return nil, err
	}
	return messages, nil
}

func findPendingParentMessage(db *gorm.DB, deviceName string) (*models.ParentMessage, error) {
	messages, err := findPendingParentMessages(db, deviceName)
	if err != nil {
		return nil, err
	}
	if len(messages) == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	msg := messages[0]
	return &msg, nil
}

func serveParentMessageAudio(c *gin.Context, priv *privacy.Service, deviceDBID uint, audioPath string) {
	if strings.TrimSpace(audioPath) == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "音频不存在"})
		return
	}
	data, err := os.ReadFile(audioPath)
	if err != nil {
		if os.IsNotExist(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "音频不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取音频失败"})
		return
	}
	if priv != nil {
		dec, err := priv.DecryptFileBytes(deviceDBID, data)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "解密音频失败"})
			return
		}
		data = dec
	}
	c.Header("Content-Type", "audio/wav")
	c.Header("Content-Length", strconv.Itoa(len(data)))
	c.Data(http.StatusOK, "audio/wav", data)
}
