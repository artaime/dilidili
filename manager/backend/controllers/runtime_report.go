package controllers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"dili/manager/backend/middleware"
	"dili/manager/backend/models"
	"dili/manager/backend/storage"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"gorm.io/gorm"
)

const (
	runtimeStreamTokenTTL   = 5 * time.Minute
	runtimeStreamInterval   = 5 * time.Second
	devicesActiveCacheTTL   = 30 * time.Second
)

type runtimeStreamClaims struct {
	Purpose string `json:"purpose"`
	UserID  uint   `json:"user_id"`
	jwt.RegisteredClaims
}

// RuntimeReportController 主服务运行时监控控制器。
type RuntimeReportController struct {
	DB                  *gorm.DB
	WebSocketController *WebSocketController
	store               *storage.NodeRuntimeStore

	devicesCacheMu sync.Mutex
	devicesCached  int
	devicesCachedAt time.Time
}

func NewRuntimeReportController(db *gorm.DB, wsController *WebSocketController) *RuntimeReportController {
	return &RuntimeReportController{
		DB:                  db,
		WebSocketController: wsController,
		store:               storage.GetNodeRuntimeStore(),
	}
}

type runtimeReportRequest struct {
	NodeID     string                 `json:"node_id" binding:"required"`
	NodeName   string                 `json:"node_name"`
	ReportedAt string                 `json:"reported_at"`
	Host       storage.HostMetricsSnapshot `json:"host"`
	App        storage.AppMetricsSnapshot  `json:"app"`
	Pools      map[string]interface{} `json:"pools"`
	Build      storage.BuildInfoSnapshot   `json:"build"`
}

// ReportRuntime 接收主服务运行时上报（内部接口）。
func (c *RuntimeReportController) ReportRuntime(ctx *gin.Context) {
	var req runtimeReportRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误: " + err.Error()})
		return
	}

	reportedAt := time.Now()
	if req.ReportedAt != "" {
		if parsed, err := time.Parse(time.RFC3339, req.ReportedAt); err == nil {
			reportedAt = parsed
		}
	}

	wsConnected := false
	if c.WebSocketController != nil {
		wsConnected = c.WebSocketController.IsClientConnected(req.NodeID)
	}

	snapshot := c.store.Upsert(storage.RuntimeReportInput{
		NodeID:     req.NodeID,
		NodeName:   req.NodeName,
		ReportedAt: reportedAt,
		Host:       req.Host,
		App:        req.App,
		Pools:      req.Pools,
		Build:      req.Build,
	}, wsConnected)

	ctx.JSON(http.StatusOK, gin.H{
		"message": "运行时数据上报成功",
		"data":    snapshot,
	})
}

// ListNodes 返回所有节点最新快照。
func (c *RuntimeReportController) ListNodes(ctx *gin.Context) {
	c.syncWSConnectedStatus()
	nodes := c.store.List()
	ctx.JSON(http.StatusOK, gin.H{"data": nodes, "count": len(nodes)})
}

// GetNode 返回单节点详情。
func (c *RuntimeReportController) GetNode(ctx *gin.Context) {
	nodeID := ctx.Param("node_id")
	c.syncWSConnectedStatus()
	node, ok := c.store.Get(nodeID)
	if !ok {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "节点不存在"})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"data": node})
}

// GetSummary 返回集群汇总与业务活跃设备数。
func (c *RuntimeReportController) GetSummary(ctx *gin.Context) {
	c.syncWSConnectedStatus()
	summary := c.store.GetClusterSummary()
	summary.DevicesActive5m = c.getDevicesActive5m()
	ctx.JSON(http.StatusOK, gin.H{"data": summary})
}

// IssueStreamToken 签发 SSE 短期访问 token。
func (c *RuntimeReportController) IssueStreamToken(ctx *gin.Context) {
	userID, _ := ctx.Get("user_id")
	uid, _ := userID.(uint)
	claims := runtimeStreamClaims{
		Purpose: "runtime-stream",
		UserID:  uid,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(runtimeStreamTokenTTL)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(middleware.JWTSecret())
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "签发 stream token 失败"})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"token":      tokenString,
			"expires_in": int(runtimeStreamTokenTTL.Seconds()),
		},
	})
}

// StreamNodes SSE 推送节点运行时快照。
func (c *RuntimeReportController) StreamNodes(ctx *gin.Context) {
	if !c.authorizeStream(ctx) {
		return
	}

	ctx.Writer.Header().Set("Content-Type", "text/event-stream")
	ctx.Writer.Header().Set("Cache-Control", "no-cache")
	ctx.Writer.Header().Set("Connection", "keep-alive")
	ctx.Writer.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := ctx.Writer.(http.Flusher)
	if !ok {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "SSE 不支持"})
		return
	}

	ticker := time.NewTicker(runtimeStreamInterval)
	defer ticker.Stop()

	c.writeStreamEvent(ctx.Writer, flusher)
	for {
		select {
		case <-ctx.Request.Context().Done():
			return
		case <-ticker.C:
			c.writeStreamEvent(ctx.Writer, flusher)
		}
	}
}

func (c *RuntimeReportController) writeStreamEvent(w io.Writer, flusher http.Flusher) {
	c.syncWSConnectedStatus()
	payload := gin.H{
		"nodes":   c.store.List(),
		"summary": c.buildSummary(),
		"ts":      time.Now().Format(time.RFC3339),
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return
	}
	fmt.Fprintf(w, "event: runtime\ndata: %s\n\n", data)
	flusher.Flush()
}

func (c *RuntimeReportController) buildSummary() storage.ClusterRuntimeSummary {
	summary := c.store.GetClusterSummary()
	summary.DevicesActive5m = c.getDevicesActive5m()
	return summary
}

// ListWSClients 返回 WS 客户端连接状态（调试用）。
func (c *RuntimeReportController) ListWSClients(ctx *gin.Context) {
	if c.WebSocketController == nil {
		ctx.JSON(http.StatusOK, gin.H{"data": gin.H{"clients": []any{}, "count": 0}})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"data": c.WebSocketController.GetClientConnectionStatus()})
}

func (c *RuntimeReportController) authorizeStream(ctx *gin.Context) bool {
	token := ctx.Query("token")
	if token == "" {
		authHeader := ctx.GetHeader("Authorization")
		if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
			token = authHeader[7:]
		}
	}
	if token == "" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "缺少 stream token"})
		return false
	}

	claims := &runtimeStreamClaims{}
	parsed, err := jwt.ParseWithClaims(token, claims, func(token *jwt.Token) (interface{}, error) {
		return middleware.JWTSecret(), nil
	})
	if err != nil || !parsed.Valid || claims.Purpose != "runtime-stream" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "无效的 stream token"})
		return false
	}
	return true
}

func (c *RuntimeReportController) syncWSConnectedStatus() {
	if c.WebSocketController == nil {
		return
	}
	status := c.WebSocketController.GetClientConnectionStatus()
	clients, _ := status["clients"].([]map[string]interface{})
	connected := map[string]bool{}
	for _, client := range clients {
		uuid, _ := client["uuid"].(string)
		isConnected, _ := client["connected"].(bool)
		if uuid != "" && isConnected {
			connected[uuid] = true
		}
	}
	for _, node := range c.store.List() {
		c.store.SetWSConnected(node.NodeID, connected[node.NodeID])
	}
}

func (c *RuntimeReportController) getDevicesActive5m() int {
	c.devicesCacheMu.Lock()
	defer c.devicesCacheMu.Unlock()
	if time.Since(c.devicesCachedAt) < devicesActiveCacheTTL {
		return c.devicesCached
	}
	if c.DB == nil {
		return 0
	}
	var count int64
	fiveMinutesAgo := time.Now().Add(-5 * time.Minute)
	if err := c.DB.Model(&models.Device{}).Where("last_active_at > ?", fiveMinutesAgo).Count(&count).Error; err != nil {
		return c.devicesCached
	}
	c.devicesCached = int(count)
	c.devicesCachedAt = time.Now()
	c.store.SetDevicesActive5m(c.devicesCached)
	return c.devicesCached
}
