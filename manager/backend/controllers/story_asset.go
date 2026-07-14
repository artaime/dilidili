package controllers

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"dili/manager/backend/services/story_persist"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type StoryAssetController struct {
	DB *gorm.DB
}

func (ctrl *StoryAssetController) svc() *story_persist.Service {
	return story_persist.NewService(ctrl.DB)
}

func (ctrl *StoryAssetController) List(c *gin.Context) {
	q := story_persist.ListAssetsQuery{
		Q:        c.Query("q"),
		PoolKind: c.Query("pool_kind"),
	}
	if v := c.Query("page"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			q.Page = n
		}
	}
	if v := c.Query("page_size"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			q.PageSize = n
		}
	}
	if v := strings.TrimSpace(c.Query("shareable")); v != "" {
		b := v == "1" || strings.EqualFold(v, "true")
		q.Shareable = &b
	}
	result, err := ctrl.svc().ListAssets(c.Request.Context(), q)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	// 列表不返回超长全文，只保留预览
	for i := range result.Items {
		if n := len([]rune(result.Items[i].FullText)); n > 200 {
			result.Items[i].FullText = string([]rune(result.Items[i].FullText)[:200]) + "…"
		}
		result.Items[i].Segments = nil
	}
	c.JSON(http.StatusOK, gin.H{"data": result})
}

func (ctrl *StoryAssetController) Get(c *gin.Context) {
	view, err := ctrl.svc().GetAsset(c.Request.Context(), c.Param("storyId"))
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

func (ctrl *StoryAssetController) Create(c *gin.Context) {
	var req story_persist.UpsertAssetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求无效"})
		return
	}
	if strings.TrimSpace(req.StoryID) == "" {
		req.StoryID = uuid.NewString()
	}
	ctrl.normalizeUpsert(&req)
	if err := ctrl.svc().UpsertAsset(c.Request.Context(), req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	view, err := ctrl.svc().GetAsset(c.Request.Context(), req.StoryID)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"data": gin.H{"story_id": req.StoryID}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": view})
}

func (ctrl *StoryAssetController) Update(c *gin.Context) {
	storyID := strings.TrimSpace(c.Param("storyId"))
	if storyID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "story_id 无效"})
		return
	}
	if _, err := ctrl.svc().GetAsset(c.Request.Context(), storyID); err != nil {
		if errors.Is(err, story_persist.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "故事不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	var req story_persist.UpsertAssetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求无效"})
		return
	}
	req.StoryID = storyID
	ctrl.normalizeUpsert(&req)
	if err := ctrl.svc().UpsertAsset(c.Request.Context(), req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	view, err := ctrl.svc().GetAsset(c.Request.Context(), storyID)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"ok": true})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": view})
}

func (ctrl *StoryAssetController) Delete(c *gin.Context) {
	if err := ctrl.svc().DeleteAsset(c.Request.Context(), c.Param("storyId")); err != nil {
		if errors.Is(err, story_persist.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "故事不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (ctrl *StoryAssetController) Generate(c *gin.Context) {
	var req story_persist.GenerateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求无效"})
		return
	}
	result, err := story_persist.GenerateStoryText(c.Request.Context(), ctrl.DB, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": result})
}

func (ctrl *StoryAssetController) normalizeUpsert(req *story_persist.UpsertAssetRequest) {
	if req == nil {
		return
	}
	if strings.TrimSpace(req.FullText) != "" && len(req.Segments) == 0 {
		req.Segments = story_persist.SegmentText(req.FullText)
	}
	if !req.GenerationComplete && strings.TrimSpace(req.FullText) != "" {
		req.GenerationComplete = true
	}
	if strings.TrimSpace(req.ThemeKey) == "" {
		req.ThemeKey = strings.TrimSpace(req.CanonicalKey)
	}
	if strings.TrimSpace(req.Title) == "" && req.ThemeKey != "" {
		req.Title = req.ThemeKey
		if !strings.Contains(req.Title, "故事") {
			req.Title = req.ThemeKey + "的故事"
		}
	}
	if req.ParamsSnapshot == nil {
		req.ParamsSnapshot = map[string]any{}
	}
	if req.PoolKind != "" {
		req.ParamsSnapshot["pool_kind"] = req.PoolKind
	}
	if req.CanonicalKey != "" {
		req.ParamsSnapshot["canonical_key"] = req.CanonicalKey
	}
	if len(req.Aliases) > 0 {
		req.ParamsSnapshot["aliases"] = req.Aliases
	}
	if req.ThemeKey != "" {
		req.ParamsSnapshot["theme"] = req.ThemeKey
	}
	if req.NarrationMode != "" {
		req.ParamsSnapshot["narration_mode"] = req.NarrationMode
	}
}
