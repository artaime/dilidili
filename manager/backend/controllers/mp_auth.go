package controllers

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"dili/manager/backend/config"
	"dili/manager/backend/middleware"
	"dili/manager/backend/models"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type MpAuthController struct {
	DB  *gorm.DB
	Cfg *config.Config
}

type wxCode2SessionResponse struct {
	OpenID     string `json:"openid"`
	SessionKey string `json:"session_key"`
	UnionID    string `json:"unionid"`
	ErrCode    int    `json:"errcode"`
	ErrMsg     string `json:"errmsg"`
}

func (ctrl *MpAuthController) Login(c *gin.Context) {
	var req struct {
		Code       string `json:"code" binding:"required"`
		Nickname   string `json:"nickname"`
		Avatar     string `json:"avatar_url"`
		FamilyRole string `json:"family_role"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}

	appID := strings.TrimSpace(ctrl.Cfg.WeChat.MiniProgram.AppID)
	appSecret := strings.TrimSpace(ctrl.Cfg.WeChat.MiniProgram.AppSecret)
	if appID == "" || appSecret == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "微信小程序未配置，请联系管理员"})
		return
	}

	wxResp, err := ctrl.exchangeWxCode(appID, appSecret, strings.TrimSpace(req.Code))
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	if wxResp.ErrCode != 0 || wxResp.OpenID == "" {
		msg := wxResp.ErrMsg
		if msg == "" {
			msg = "微信登录失败"
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": msg})
		return
	}

	var user models.User
	openID := wxResp.OpenID
	err = ctrl.DB.Where("wx_openid = ?", openID).First(&user).Error
	if err == gorm.ErrRecordNotFound {
		user, err = ctrl.createMiniProgramUser(wxResp, req.Nickname, req.Avatar, req.FamilyRole)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询用户失败"})
		return
	} else {
		updates := map[string]interface{}{}
		if req.Nickname != "" {
			updates["nickname"] = strings.TrimSpace(req.Nickname)
		}
		if req.Avatar != "" {
			updates["avatar_url"] = strings.TrimSpace(req.Avatar)
		}
		if req.FamilyRole != "" {
			updates["family_role"] = normalizeFamilyRole(req.FamilyRole)
		}
		if wxResp.UnionID != "" {
			unionID := wxResp.UnionID
			updates["wx_unionid"] = &unionID
		}
		if len(updates) > 0 {
			_ = ctrl.DB.Model(&user).Updates(updates).Error
			_ = ctrl.DB.Where("id = ?", user.ID).First(&user).Error
		}
	}

	token, err := middleware.GenerateToken(user.ID, user.Username, user.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "生成 token 失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"user": gin.H{
			"id":          user.ID,
			"username":    user.Username,
			"nickname":    user.Nickname,
			"avatar_url":  user.AvatarURL,
			"family_role": normalizeFamilyRole(user.FamilyRole),
			"role":        user.Role,
			"source":      user.Source,
		},
	})
}

func (ctrl *MpAuthController) Profile(c *gin.Context) {
	userID, _ := c.Get("user_id")
	var user models.User
	if err := ctrl.DB.Where("id = ?", userID).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"id":          user.ID,
			"username":    user.Username,
			"nickname":    user.Nickname,
			"avatar_url":  user.AvatarURL,
			"phone":       user.Phone,
			"family_role": normalizeFamilyRole(user.FamilyRole),
			"source":      user.Source,
		},
	})
}

func (ctrl *MpAuthController) UpdateProfile(c *gin.Context) {
	userID, _ := c.Get("user_id")
	currentUID, _ := userID.(uint)

	var req struct {
		Nickname   string `json:"nickname"`
		AvatarURL  string `json:"avatar_url"`
		FamilyRole string `json:"family_role"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}

	updates := map[string]interface{}{}
	if nickname := strings.TrimSpace(req.Nickname); nickname != "" {
		updates["nickname"] = nickname
	}
	if avatar := strings.TrimSpace(req.AvatarURL); avatar != "" {
		updates["avatar_url"] = avatar
	}
	if req.FamilyRole != "" {
		if !isValidFamilyRole(req.FamilyRole) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的亲属角色"})
			return
		}
		updates["family_role"] = strings.TrimSpace(req.FamilyRole)
	}
	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "没有可更新的字段"})
		return
	}

	if err := ctrl.DB.Model(&models.User{}).Where("id = ?", currentUID).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新资料失败"})
		return
	}

	var user models.User
	if err := ctrl.DB.Where("id = ?", currentUID).First(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询用户失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": "资料已更新",
		"data": gin.H{
			"id":          user.ID,
			"username":    user.Username,
			"nickname":    user.Nickname,
			"avatar_url":  user.AvatarURL,
			"phone":       user.Phone,
			"family_role": normalizeFamilyRole(user.FamilyRole),
			"source":      user.Source,
		},
	})
}

func (ctrl *MpAuthController) exchangeWxCode(appID, appSecret, code string) (*wxCode2SessionResponse, error) {
	endpoint := fmt.Sprintf(
		"https://api.weixin.qq.com/sns/jscode2session?appid=%s&secret=%s&js_code=%s&grant_type=authorization_code",
		url.QueryEscape(appID),
		url.QueryEscape(appSecret),
		url.QueryEscape(code),
	)
	resp, err := http.Get(endpoint)
	if err != nil {
		return nil, fmt.Errorf("微信服务请求失败")
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取微信响应失败")
	}
	var wxResp wxCode2SessionResponse
	if err := json.Unmarshal(body, &wxResp); err != nil {
		return nil, fmt.Errorf("解析微信响应失败")
	}
	return &wxResp, nil
}

func (ctrl *MpAuthController) createMiniProgramUser(wxResp *wxCode2SessionResponse, nickname, avatar, familyRole string) (models.User, error) {
	prefix := wxResp.OpenID
	if len(prefix) > 8 {
		prefix = prefix[:8]
	}
	username := fmt.Sprintf("mp_%s", prefix)
	for i := 0; i < 5; i++ {
		var count int64
		if err := ctrl.DB.Model(&models.User{}).Where("username = ?", username).Count(&count).Error; err != nil {
			return models.User{}, err
		}
		if count == 0 {
			break
		}
		username = fmt.Sprintf("mp_%s_%d", prefix, i+1)
	}

	randomSecret := make([]byte, 16)
	if _, err := rand.Read(randomSecret); err != nil {
		return models.User{}, fmt.Errorf("生成用户凭证失败")
	}
	hashed, err := bcrypt.GenerateFromPassword([]byte(hex.EncodeToString(randomSecret)), bcrypt.DefaultCost)
	if err != nil {
		return models.User{}, fmt.Errorf("生成用户凭证失败")
	}

	openID := wxResp.OpenID
	var unionIDPtr *string
	if wxResp.UnionID != "" {
		unionID := wxResp.UnionID
		unionIDPtr = &unionID
	}
	user := models.User{
		Username:   username,
		Password:   string(hashed),
		Email:      fmt.Sprintf("%s@miniprogram.local", username),
		Role:       "user",
		WxOpenid:   &openID,
		WxUnionid:  unionIDPtr,
		Nickname:   strings.TrimSpace(nickname),
		AvatarURL:  strings.TrimSpace(avatar),
		FamilyRole: normalizeFamilyRole(familyRole),
		Source:     "miniprogram",
	}
	if err := ctrl.DB.Create(&user).Error; err != nil {
		return models.User{}, fmt.Errorf("创建用户失败")
	}
	return user, nil
}
