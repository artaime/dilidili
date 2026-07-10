package controllers

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"

	"dili/manager/backend/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type DeviceActivationController struct {
	DB *gorm.DB
}

// 生成UUID格式的挑战码
func generateChallenge() string {
	randomBytes := make([]byte, 16)
	rand.Read(randomBytes)

	// 设置版本 (4) 和变体位
	randomBytes[6] = (randomBytes[6] & 0x0f) | 0x40
	randomBytes[8] = (randomBytes[8] & 0x3f) | 0x80

	return fmt.Sprintf("%x-%x-%x-%x-%x",
		randomBytes[0:4],
		randomBytes[4:6],
		randomBytes[6:8],
		randomBytes[8:10],
		randomBytes[10:16])
}

// 1. 判断设备是否已激活
// GET /api/internal/device/check-activation?device_id=xxx&client_id=xxx
func (dac *DeviceActivationController) CheckDeviceActivation(c *gin.Context) {
	deviceId := normalizeDeviceSN(c.Query("device_id"))
	//clientId := c.Query("client_id")

	if deviceId == "" /*|| clientId == ""*/ {
		c.JSON(http.StatusOK, gin.H{
			"activated": false,
			"error":     "device_id参数必填",
		})
		return
	}

	var device models.Device
	// 使用device_id (对应device_name字段) 查找设备
	if err := dac.DB.Where("device_name = ?", deviceId).First(&device).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusOK, gin.H{
				"activated": false,
				"message":   "设备不存在",
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"activated": false,
			"error":     "查询设备失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"activated": device.Activated,
		"message": func() string {
			if device.Activated {
				return "设备已激活"
			}
			return "设备未激活"
		}(),
	})
}

// 2. 获取激活信息
// GET /api/internal/device/activation-info?device_id=xxx&client_id=xxx
func (dac *DeviceActivationController) GetActivationInfo(c *gin.Context) {
	deviceId := normalizeDeviceSN(c.Query("device_id"))
	//clientId := c.Query("client_id")

	if deviceId == "" /*|| clientId == ""*/ {
		c.JSON(http.StatusBadRequest, gin.H{"error": "device_id和client_id参数必填"})
		return
	}

	var device models.Device

	// 使用device_id (对应device_name字段) 查找设备
	if err := dac.DB.Where("device_name = ?", deviceId).First(&device).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusOK, gin.H{
				"registered": false,
				"activated":  false,
				"message":    "设备未登记，请联系厂商",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询设备失败"})
		return
	}

	// 如果设备已激活，直接返回状态
	if device.Activated {
		c.JSON(http.StatusOK, gin.H{
			"activated": true,
			"message":   "设备已激活",
		})
		return
	}

	// 未激活设备：仅生成挑战码（供设备端 HMAC 激活握手），不再生成激活码
	needUpdate := false
	if device.Challenge == "" {
		device.Challenge = generateChallenge()
		needUpdate = true
	}

	if needUpdate {
		if err := updateDeviceColumns(dac.DB, device.ID, map[string]interface{}{
			"challenge": device.Challenge,
		}); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "更新设备信息失败"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"activated": false,
		"challenge": device.Challenge,
		"message":   "请在小程序上绑定我，双击电源键进入配网模式",
	})
}

// 验证HMAC-SHA256
func verifyHMAC(challenge, secretKey, providedHmac string) bool {
	if secretKey == "" {
		return true // 如果pre_secret_key为空，直接通过验证
	}

	mac := hmac.New(sha256.New, []byte(secretKey))
	mac.Write([]byte(challenge))
	expectedHmac := hex.EncodeToString(mac.Sum(nil))

	return expectedHmac == providedHmac
}

// 3. 设备激活接口
// POST /api/internal/device/activate
func (dac *DeviceActivationController) ActivateDevice(c *gin.Context) {
	var req struct {
		DeviceId     string `json:"device_id" binding:"required"`
		ClientId     string `json:"client_id" binding:"required"`
		Challenge    string `json:"challenge" binding:"required"`
		Algorithm    string `json:"algorithm" binding:"required"`
		SerialNumber string `json:"serial_number" binding:"required"`
		Hmac         string `json:"hmac" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}

	deviceID := normalizeDeviceSN(req.DeviceId)
	serialNumber := normalizeDeviceSN(req.SerialNumber)
	if deviceID == "" || serialNumber == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "device_id 与 serial_number 不能为空",
		})
		return
	}
	if deviceID != serialNumber {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"error":   "serial_number 与 device_id 不一致",
		})
		return
	}

	var device models.Device
	// 使用 device_id（对应 device_name 字段，值为 SN）查找设备
	if err := dac.DB.Where("device_name = ?", deviceID).First(&device).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"error":   "设备不存在",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "查询设备失败",
		})
		return
	}

	// 检查设备是否已经激活
	if device.Activated {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "设备已激活",
		})
		return
	}

	if device.UserID == 0 {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"error":   "设备未绑定用户",
		})
		return
	}

	if device.Challenge != req.Challenge {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"error":   "挑战码错误",
		})
		return
	}

	// 验证HMAC（如果pre_secret_key为空则直接通过）
	if !verifyHMAC(req.Challenge, device.PreSecretKey, req.Hmac) {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"error":   "HMAC验证失败",
		})
		return
	}

	// 激活设备
	device.Activated = true
	if err := updateDeviceColumns(dac.DB, device.ID, map[string]interface{}{
		"activated": device.Activated,
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "激活设备失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "设备激活成功",
		"data": gin.H{
			"device_id": device.DeviceName,
			"activated": device.Activated,
		},
	})
}
