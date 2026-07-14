package controllers

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"dili/manager/backend/models"
	"dili/manager/backend/services/story_persist"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type StoryInternalController struct {
	DB *gorm.DB
}

func (ctrl *StoryInternalController) svc() *story_persist.Service {
	return story_persist.NewService(ctrl.DB)
}

func (ctrl *StoryInternalController) UpsertAsset(c *gin.Context) {
	var req story_persist.UpsertAssetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求无效"})
		return
	}
	if err := ctrl.resolveCreatorUser(&req); err != nil {
		// 忽略解析用户失败，仍可落库
	}
	if err := ctrl.svc().UpsertAsset(c.Request.Context(), req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (ctrl *StoryInternalController) UpsertPlayback(c *gin.Context) {
	var req story_persist.UpsertPlaybackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求无效"})
		return
	}
	if req.UserID == 0 && strings.TrimSpace(req.DeviceSN) != "" {
		var device models.Device
		if err := ctrl.DB.Where("device_name = ?", strings.TrimSpace(req.DeviceSN)).First(&device).Error; err == nil {
			req.UserID = device.UserID
		}
	}
	if err := ctrl.svc().UpsertPlayback(c.Request.Context(), req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (ctrl *StoryInternalController) GetAsset(c *gin.Context) {
	view, err := ctrl.svc().GetAsset(c.Request.Context(), c.Param("id"))
	if err != nil {
		if errors.Is(err, story_persist.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "故事不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": view})
}

func (ctrl *StoryInternalController) FindShareable(c *gin.Context) {
	q := story_persist.FindShareableQuery{
		PoolKind: c.Query("pool_kind"),
		Theme:    c.Query("theme"),
		AgeBand:  c.Query("age_band"),
		DeviceSN: c.Query("device_sn"),
	}
	if q.Theme == "" {
		q.Theme = c.Query("theme_key") // 兼容旧参数
	}
	if v := c.Query("exclude_days"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			q.ExcludeDays = n
		}
	}
	if v := c.Query("top_k"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			q.TopK = n
		}
	}
	view, err := ctrl.svc().FindShareable(c.Request.Context(), q)
	if err != nil {
		if errors.Is(err, story_persist.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "无可用共享故事"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": view})
}

func (ctrl *StoryInternalController) resolveCreatorUser(req *story_persist.UpsertAssetRequest) error {
	if req == nil || req.CreatorUserID != 0 || strings.TrimSpace(req.CreatorDeviceSN) == "" {
		return nil
	}
	var device models.Device
	if err := ctrl.DB.Where("device_name = ?", strings.TrimSpace(req.CreatorDeviceSN)).First(&device).Error; err != nil {
		return err
	}
	req.CreatorUserID = device.UserID
	return nil
}
