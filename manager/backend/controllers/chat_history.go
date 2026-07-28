package controllers

import (
	"crypto/md5"
	"dili/manager/backend/models"
	"dili/manager/backend/privacy"
	"dili/manager/backend/services/device_acl"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ChatHistoryController struct {
	DB            *gorm.DB
	AudioBasePath string // 音频存储基础路径
	MaxFileSize   int64  // 最大文件大小（10MB）
	Privacy       *privacy.Service
}

// SaveMessageRequest 保存消息请求
type SaveMessageRequest struct {
	MessageID     string                 `json:"message_id" binding:"required"`
	DeviceID      string                 `json:"device_id" binding:"required"`
	AgentID       string                 `json:"agent_id" binding:"required"`
	SessionID     string                 `json:"session_id,omitempty"`
	Role          string                 `json:"role" binding:"required,oneof=user assistant system tool"`
	Content       string                 `json:"content" binding:"required"`
	ToolCallID    string                 `json:"tool_call_id,omitempty"`    // 工具调用ID（Tool角色使用）
	ToolCallsJSON *string                `json:"tool_calls_json,omitempty"` // 工具调用列表JSON（Assistant角色使用），nil 表示 NULL
	AudioData     string                 `json:"audio_data,omitempty"`      // base64编码
	AudioFormat   string                 `json:"audio_format,omitempty"`    // 音频格式（客户端传入，后端固定使用wav）
	AudioDuration int                    `json:"audio_duration,omitempty"`
	AudioSize     int                    `json:"audio_size,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// SaveMessage 保存消息
func (c *ChatHistoryController) SaveMessage(ctx *gin.Context) {
	var req SaveMessageRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 验证设备存在（使用device_name字段查询）
	var device models.Device
	if err := c.DB.Where("device_name = ?", req.DeviceID).First(&device).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "设备不存在"})
			return
		}
		// 其他数据库错误
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "查询设备失败: " + err.Error()})
		return
	}

	// 如果请求中没有提供 AgentID，使用设备关联的 AgentID
	agentID := req.AgentID
	if agentID == "" && device.AgentID > 0 {
		agentID = fmt.Sprintf("%d", device.AgentID)
	}

	// 如果 AgentID 仍然为空，跳过保存
	if agentID == "" {
		ctx.JSON(http.StatusOK, gin.H{"message": "跳过保存: 没有关联的 AgentID"})
		return
	}

	content := req.Content
	if enc, err := c.encryptContent(device.ID, content); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "加密消息失败: " + err.Error()})
		return
	} else {
		content = enc
	}

	message := &models.ChatMessage{
		MessageID:     req.MessageID,
		DeviceID:      req.DeviceID,
		AgentID:       agentID,
		UserID:        device.UserID,
		SessionID:     req.SessionID,
		Role:          req.Role,
		Content:       content,
		ToolCallID:    req.ToolCallID,
		ToolCallsJSON: req.ToolCallsJSON,
		Metadata:      req.Metadata,
	}

	// 检查消息是否已存在（避免重复创建）
	var existingMessage models.ChatMessage
	err := c.DB.Where("message_id = ?", req.MessageID).First(&existingMessage).Error
	if err == nil {
		// 消息已存在，更新音频数据（如果提供了）
		if req.AudioData != "" {
			audioPath, err := c.saveAudioFile(device.ID, req.MessageID, req.AudioData)
			if err != nil {
				ctx.JSON(http.StatusInternalServerError, gin.H{"error": "保存音频文件失败: " + err.Error()})
				return
			}

			// 如果之前有音频文件，先删除
			if existingMessage.AudioPath != "" {
				c.deleteAudioFile(existingMessage.AudioPath)
			}

			// 更新消息
			updates := map[string]interface{}{
				"audio_path":   audioPath,
				"audio_format": "wav",
			}
			if req.AudioSize > 0 {
				updates["audio_size"] = req.AudioSize
			}
			if req.AudioDuration > 0 {
				updates["audio_duration"] = req.AudioDuration
			}

			// 更新 metadata（合并）
			if existingMessage.Metadata == nil {
				existingMessage.Metadata = make(map[string]interface{})
			}
			if req.Metadata != nil {
				for k, v := range req.Metadata {
					existingMessage.Metadata[k] = v
				}
			}
			// 手动序列化 metadata 到 MetadataJSON（因为 Updates 不会触发 BeforeSave hook）
			metadataJSONBytes, err := json.Marshal(existingMessage.Metadata)
			if err != nil {
				ctx.JSON(http.StatusInternalServerError, gin.H{"error": "序列化 metadata 失败: " + err.Error()})
				return
			}
			updates["metadata"] = string(metadataJSONBytes)

			if err := c.DB.Model(&existingMessage).Updates(updates).Error; err != nil {
				ctx.JSON(http.StatusInternalServerError, gin.H{"error": "更新消息失败"})
				return
			}
			ctx.JSON(http.StatusOK, existingMessage)
			return
		}
		// 消息已存在且没有音频数据，直接返回
		ctx.JSON(http.StatusOK, existingMessage)
		return
	} else if err != gorm.ErrRecordNotFound {
		// 查询出错（非"记录不存在"）
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "查询消息失败: " + err.Error()})
		return
	}

	// 消息不存在，创建新消息
	// 处理音频数据 - 保存到文件系统（固定为wav格式，两级hash打散）
	if req.AudioData != "" {
		audioPath, err := c.saveAudioFile(device.ID, req.MessageID, req.AudioData)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "保存音频文件失败: " + err.Error()})
			return
		}
		message.AudioPath = audioPath
		message.AudioFormat = "wav" // 固定为wav格式
		if req.AudioSize > 0 {
			message.AudioSize = &req.AudioSize
		}
		if req.AudioDuration > 0 {
			message.AudioDuration = &req.AudioDuration
		}
	}

	if err := c.DB.Create(message).Error; err != nil {
		// 如果数据库保存失败，删除已保存的音频文件
		if message.AudioPath != "" {
			c.deleteAudioFile(message.AudioPath)
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "保存消息失败: " + err.Error()})
		return
	}

	ctx.JSON(http.StatusCreated, message)
}

// GetMessages 获取消息列表（按agentId汇总）
func (c *ChatHistoryController) GetMessages(ctx *gin.Context) {
	userID, exists := ctx.Get("user_id")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "未授权"})
		return
	}

	agentID := ctx.Query("agent_id")
	deviceID := ctx.Query("device_id")
	sessionID := ctx.Query("session_id")
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("page_size", "50"))
	role := ctx.Query("role") // user/assistant

	// 构建查询
	query := c.DB.Model(&models.ChatMessage{}).
		Where("user_id = ? AND is_deleted = ?", userID, false)

	if agentID != "" {
		query = query.Where("agent_id = ?", agentID)
	}
	if deviceID != "" {
		query = query.Where("device_id = ?", deviceID)
	}
	if sessionID != "" {
		query = query.Where("session_id = ?", sessionID)
	}
	if role != "" {
		query = query.Where("role = ?", role)
	}

	var total int64
	query.Count(&total)

	var messages []models.ChatMessage
	offset := (page - 1) * pageSize
	if err := query.Order("created_at DESC").
		Limit(pageSize).Offset(offset).
		Find(&messages).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}

	c.decryptChatMessages(messages)

	ctx.JSON(http.StatusOK, gin.H{
		"total":     total,
		"page":      page,
		"page_size": pageSize,
		"data":      messages,
	})
}

// DeleteMessage 删除消息（软删除，立即删除音频文件）
func (c *ChatHistoryController) DeleteMessage(ctx *gin.Context) {
	id := ctx.Param("id")

	userID, exists := ctx.Get("user_id")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "未授权"})
		return
	}

	// 获取消息信息
	var message models.ChatMessage
	if err := c.DB.Where("id = ? AND user_id = ?", id, userID).First(&message).Error; err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "消息不存在"})
		return
	}

	// 先删除音频文件（如果存在）
	if message.AudioPath != "" {
		if err := c.deleteAudioFile(message.AudioPath); err != nil {
			// 记录日志，但不影响删除操作
			log.Printf("删除音频文件失败: %v", err)
		}
	}

	// 软删除消息
	if err := c.DB.Model(&models.ChatMessage{}).
		Where("id = ?", id).
		Update("is_deleted", true).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

// GetMessagesByAgent 按AgentID获取消息汇总（支持筛选）
func (c *ChatHistoryController) GetMessagesByAgent(ctx *gin.Context) {
	userID, exists := ctx.Get("user_id")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "未授权"})
		return
	}

	agentID := ctx.Param("agent_id")
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("page_size", "50"))
	role := ctx.Query("role")            // user/assistant
	deviceID := ctx.Query("device_id")   // 设备ID筛选
	startDate := ctx.Query("start_date") // 开始日期 YYYY-MM-DD
	endDate := ctx.Query("end_date")     // 结束日期 YYYY-MM-DD

	// 构建查询
	query := c.DB.Model(&models.ChatMessage{}).
		Where("user_id = ? AND agent_id = ? AND is_deleted = ?", userID, agentID, false)

	// 角色筛选
	if role != "" {
		query = query.Where("role = ?", role)
	}

	// 设备筛选
	if deviceID != "" {
		query = query.Where("device_id = ?", deviceID)
	}

	// 日期范围筛选
	if startDate != "" {
		if startTime, err := time.Parse("2006-01-02", startDate); err == nil {
			query = query.Where("created_at >= ?", startTime)
		}
	}
	if endDate != "" {
		if endTime, err := time.Parse("2006-01-02", endDate); err == nil {
			// 结束日期包含整天
			endTime = endTime.Add(24 * time.Hour)
			query = query.Where("created_at < ?", endTime)
		}
	}

	// 计算总数
	var total int64
	query.Count(&total)

	// 分页查询（按时间倒序，最新的在前，前端会反转数组使最新的在底部）
	var messages []models.ChatMessage
	offset := (page - 1) * pageSize
	if err := query.Order("created_at DESC").
		Limit(pageSize).Offset(offset).
		Find(&messages).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}

	c.decryptChatMessages(messages)

	ctx.JSON(http.StatusOK, gin.H{
		"total":     total,
		"page":      page,
		"page_size": pageSize,
		"data":      messages,
	})
}

// ExportMessages 导出聊天记录（JSON格式）
func (c *ChatHistoryController) ExportMessages(ctx *gin.Context) {
	userID, exists := ctx.Get("user_id")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "未授权"})
		return
	}

	agentID := ctx.Query("agent_id")
	deviceID := ctx.Query("device_id")
	startDate := ctx.Query("start_date")
	endDate := ctx.Query("end_date")

	// 构建查询
	query := c.DB.Model(&models.ChatMessage{}).
		Where("user_id = ? AND is_deleted = ?", userID, false)

	if agentID != "" {
		query = query.Where("agent_id = ?", agentID)
	}
	if deviceID != "" {
		query = query.Where("device_id = ?", deviceID)
	}
	if startDate != "" {
		if startTime, err := time.Parse("2006-01-02", startDate); err == nil {
			query = query.Where("created_at >= ?", startTime)
		}
	}
	if endDate != "" {
		if endTime, err := time.Parse("2006-01-02", endDate); err == nil {
			// 结束日期包含整天
			endTime = endTime.Add(24 * time.Hour)
			query = query.Where("created_at < ?", endTime)
		}
	}

	var messages []models.ChatMessage
	if err := query.Order("created_at ASC").Find(&messages).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "导出失败"})
		return
	}

	c.decryptChatMessages(messages)

	// 设置响应头，提示下载
	ctx.Header("Content-Type", "application/json")
	ctx.Header("Content-Disposition", "attachment; filename=chat_history_"+time.Now().Format("20060102_150405")+".json")
	ctx.JSON(http.StatusOK, gin.H{
		"export_time": time.Now().Format("2006-01-02 15:04:05"),
		"total":       len(messages),
		"messages":    messages,
	})
}

// saveAudioFile 保存音频文件到文件系统（两级hash打散）；deviceDBID 用于落盘加密。
func (c *ChatHistoryController) saveAudioFile(deviceDBID uint, messageID, audioDataBase64 string) (string, error) {
	audioData, err := base64.StdEncoding.DecodeString(audioDataBase64)
	if err != nil {
		return "", fmt.Errorf("解码音频数据失败: %v", err)
	}
	if int64(len(audioData)) > c.MaxFileSize {
		return "", fmt.Errorf("音频文件大小超过限制: %d > %d", len(audioData), c.MaxFileSize)
	}
	if c.Privacy != nil {
		enc, err := c.Privacy.EncryptFileBytes(deviceDBID, audioData)
		if err != nil {
			return "", fmt.Errorf("加密音频失败: %w", err)
		}
		audioData = enc
	}
	fileNameHash := fmt.Sprintf("%x", md5.Sum([]byte(messageID)))
	hash1 := fileNameHash[0:2]
	hash2 := fileNameHash[2:4]
	relativePath := fmt.Sprintf("%s/%s/%s.wav", hash1, hash2, fileNameHash)
	fullPath := filepath.Join(c.AudioBasePath, relativePath)
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("创建目录失败: %v", err)
	}
	if err := os.WriteFile(fullPath, audioData, 0644); err != nil {
		return "", fmt.Errorf("写入文件失败: %v", err)
	}
	return relativePath, nil
}

func (c *ChatHistoryController) deleteAudioFile(relativePath string) error {
	fullPath := filepath.Join(c.AudioBasePath, relativePath)
	return os.Remove(fullPath)
}

func (c *ChatHistoryController) encryptContent(deviceDBID uint, content string) (string, error) {
	if c.Privacy == nil {
		return content, nil
	}
	return c.Privacy.EncryptText(deviceDBID, content)
}

func (c *ChatHistoryController) decryptChatMessages(messages []models.ChatMessage) {
	if c.Privacy == nil || len(messages) == 0 {
		return
	}
	deviceIDs := make(map[string]uint)
	for i := range messages {
		sn := messages[i].DeviceID
		deviceDBID, ok := deviceIDs[sn]
		if !ok {
			var device models.Device
			if err := c.DB.Select("id").Where("device_name = ?", sn).First(&device).Error; err != nil {
				log.Printf("解密对话失败：找不到设备 %s: %v", sn, err)
				continue
			}
			deviceDBID = device.ID
			deviceIDs[sn] = deviceDBID
		}
		plain, err := c.Privacy.DecryptText(deviceDBID, messages[i].Content)
		if err != nil {
			log.Printf("解密对话 content 失败 message_id=%s: %v", messages[i].MessageID, err)
			continue
		}
		messages[i].Content = plain
	}
}

func (c *ChatHistoryController) GetAudioFile(ctx *gin.Context) {
	userID, exists := ctx.Get("user_id")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "未授权"})
		return
	}
	uid, _ := userID.(uint)
	serveChatMessageAudio(ctx, c.DB, c.Privacy, c.AudioBasePath, ctx.Param("id"), &uid)
}

func serveChatMessageAudio(ctx *gin.Context, db *gorm.DB, priv *privacy.Service, audioBasePath string, id string, userID *uint) {
	var message models.ChatMessage
	if err := db.Where("id = ? AND is_deleted = ?", id, false).First(&message).Error; err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "消息不存在"})
		return
	}
	var device models.Device
	if err := db.Where("device_name = ?", message.DeviceID).First(&device).Error; err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "消息不存在"})
		return
	}
	if userID != nil {
		if !device_acl.CanAccess(db, device.ID, *userID) {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "消息不存在"})
			return
		}
	} else {
		log.Printf("audit admin chat_audio message_id=%d device_id=%d", message.ID, device.ID)
	}
	if message.AudioPath == "" {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "音频文件不存在"})
		return
	}
	fullPath := filepath.Join(audioBasePath, message.AudioPath)
	audioData, err := os.ReadFile(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "音频文件不存在"})
		} else {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "读取音频文件失败"})
		}
		return
	}
	if priv != nil {
		dec, err := priv.DecryptFileBytes(device.ID, audioData)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "解密音频失败"})
			return
		}
		audioData = dec
	}
	ctx.Header("Content-Type", "audio/wav")
	ctx.Header("Content-Length", strconv.Itoa(len(audioData)))
	ctx.Header("Content-Disposition", fmt.Sprintf("inline; filename=%s", filepath.Base(message.AudioPath)))
	ctx.Data(http.StatusOK, "audio/wav", audioData)
}

func (c *ChatHistoryController) GetMessagesForInit(ctx *gin.Context) {
	deviceID := ctx.Query("device_id")
	agentID := ctx.Query("agent_id")
	sessionID := ctx.Query("session_id")
	userID, _ := strconv.ParseUint(ctx.Query("user_id"), 10, 64)
	limit, _ := strconv.Atoi(ctx.DefaultQuery("limit", "20"))

	if deviceID == "" || agentID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "device_id 和 agent_id 不能为空"})
		return
	}
	if userID == 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "user_id 必须为正整数"})
		return
	}
	if limit <= 0 {
		limit = 20
	}

	query := c.DB.Model(&models.ChatMessage{}).
		Where("user_id = ? AND device_id = ? AND agent_id = ? AND is_deleted = ?", uint(userID), deviceID, agentID, false)
	if sessionID != "" {
		query = query.Where("session_id = ?", sessionID)
	}

	var messages []models.ChatMessage
	if err := query.Order("created_at DESC").Order("id DESC").Limit(limit).Find(&messages).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}
	c.decryptChatMessages(messages)

	messageItems := make([]map[string]interface{}, 0, len(messages))
	for _, msg := range messages {
		item := map[string]interface{}{
			"message_id": msg.MessageID,
			"role":       msg.Role,
			"content":    msg.Content,
			"created_at": msg.CreatedAt.Format(time.RFC3339),
		}
		if msg.ToolCallID != "" {
			item["tool_call_id"] = msg.ToolCallID
		}
		if msg.ToolCallsJSON != nil && *msg.ToolCallsJSON != "" {
			var toolCalls []interface{}
			if err := json.Unmarshal([]byte(*msg.ToolCallsJSON), &toolCalls); err == nil {
				item["tool_calls"] = toolCalls
			}
		}
		messageItems = append(messageItems, item)
	}
	ctx.JSON(http.StatusOK, gin.H{"messages": messageItems})
}

type UpdateMessageAudioRequest struct {
	AudioData   string                 `json:"audio_data" binding:"required"`
	AudioFormat string                 `json:"audio_format"`
	AudioSize   int                    `json:"audio_size"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

func (c *ChatHistoryController) UpdateMessageAudio(ctx *gin.Context) {
	messageID := ctx.Param("message_id")
	if messageID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "message_id 不能为空"})
		return
	}
	var req UpdateMessageAudioRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var message models.ChatMessage
	if err := c.DB.Where("message_id = ?", messageID).First(&message).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			ctx.JSON(http.StatusOK, gin.H{"message": "跳过更新: 消息不存在"})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "查询消息失败"})
		return
	}
	if message.AgentID == "" {
		ctx.JSON(http.StatusOK, gin.H{"message": "跳过更新: 没有关联的 AgentID"})
		return
	}
	var device models.Device
	deviceDBID := uint(0)
	if err := c.DB.Select("id").Where("device_name = ?", message.DeviceID).First(&device).Error; err == nil {
		deviceDBID = device.ID
	}
	if req.AudioData != "" {
		audioPath, err := c.saveAudioFile(deviceDBID, messageID, req.AudioData)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "保存音频文件失败: " + err.Error()})
			return
		}
		if message.AudioPath != "" {
			c.deleteAudioFile(message.AudioPath)
		}
		updates := map[string]interface{}{
			"audio_path":   audioPath,
			"audio_format": "wav",
		}
		if req.AudioSize > 0 {
			updates["audio_size"] = req.AudioSize
		}
		if message.Metadata == nil {
			message.Metadata = make(map[string]interface{})
		}
		for k, v := range req.Metadata {
			message.Metadata[k] = v
		}
		metadataJSONBytes, err := json.Marshal(message.Metadata)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "序列化 metadata 失败: " + err.Error()})
			return
		}
		updates["metadata"] = string(metadataJSONBytes)
		if err := c.DB.Model(&message).Updates(updates).Error; err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "更新消息失败"})
			return
		}
	}
	ctx.JSON(http.StatusOK, message)
}
