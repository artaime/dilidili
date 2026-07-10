package storage

import (
	"sync"
	"time"
)

const (
	NodeStatusOnline  = "online"
	NodeStatusOffline = "offline"
)

const defaultNodeOfflineAfter = 30 * time.Second

// HostMetricsSnapshot 主机指标快照。
type HostMetricsSnapshot struct {
	CPUPercent  float64 `json:"cpu_percent"`
	MemTotalMB  uint64  `json:"mem_total_mb"`
	MemUsedMB   uint64  `json:"mem_used_mb"`
	MemPercent  float64 `json:"mem_percent"`
	DiskTotalGB float64 `json:"disk_total_gb"`
	DiskUsedGB  float64 `json:"disk_used_gb"`
	DiskPercent float64 `json:"disk_percent"`
	NetRxBps    float64 `json:"net_rx_bps"`
	NetTxBps    float64 `json:"net_tx_bps"`
}

// AppMetricsSnapshot 应用指标快照。
type AppMetricsSnapshot struct {
	UptimeSec          int64  `json:"uptime_sec"`
	Goroutines         int    `json:"goroutines"`
	RSSMB              uint64 `json:"rss_mb"`
	ChatManagerCount   int    `json:"chat_manager_count"`
	ActiveSessionCount int    `json:"active_session_count"`
	TransportWS        int    `json:"transport_ws"`
	TransportMqttUdp   int    `json:"transport_mqtt_udp"`
}

// BuildInfoSnapshot 构建信息快照。
type BuildInfoSnapshot struct {
	Version   string `json:"version"`
	GoVersion string `json:"go_version"`
}

// NodeRuntimeSnapshot 单节点运行时快照。
type NodeRuntimeSnapshot struct {
	NodeID      string                 `json:"node_id"`
	NodeName    string                 `json:"node_name"`
	ReportedAt  time.Time              `json:"reported_at"`
	ReceivedAt  time.Time              `json:"received_at"`
	WSConnected bool                   `json:"ws_connected"`
	Status      string                 `json:"status"`
	Host        HostMetricsSnapshot    `json:"host"`
	App         AppMetricsSnapshot     `json:"app"`
	Pools       map[string]interface{} `json:"pools"`
	Build       BuildInfoSnapshot      `json:"build"`
}

// ClusterRuntimeSummary 集群运行时汇总。
type ClusterRuntimeSummary struct {
	TotalNodes         int `json:"total_nodes"`
	OnlineNodes        int `json:"online_nodes"`
	TotalChatManagers  int `json:"total_chat_managers"`
	TotalActiveSessions int `json:"total_active_sessions"`
	DevicesActive5m    int `json:"devices_active_5m"`
}

// RuntimeReportInput 主服务上报输入。
type RuntimeReportInput struct {
	NodeID     string
	NodeName   string
	ReportedAt time.Time
	Host       HostMetricsSnapshot
	App        AppMetricsSnapshot
	Pools      map[string]interface{}
	Build      BuildInfoSnapshot
}

// NodeRuntimeStore 按 node_id 保存各主服务节点最新运行时快照。
type NodeRuntimeStore struct {
	mu              sync.RWMutex
	nodes           map[string]*NodeRuntimeSnapshot
	offlineAfter    time.Duration
	devicesActive5m int
	devicesCachedAt time.Time
}

var (
	globalNodeRuntimeStore *NodeRuntimeStore
	nodeRuntimeStoreOnce   sync.Once
)

// GetNodeRuntimeStore 获取全局节点运行时存储（单例）。
func GetNodeRuntimeStore() *NodeRuntimeStore {
	nodeRuntimeStoreOnce.Do(func() {
		globalNodeRuntimeStore = &NodeRuntimeStore{
			nodes:        make(map[string]*NodeRuntimeSnapshot),
			offlineAfter: defaultNodeOfflineAfter,
		}
		go globalNodeRuntimeStore.offlineWatcher()
	})
	return globalNodeRuntimeStore
}

func (s *NodeRuntimeStore) offlineWatcher() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		s.refreshOfflineStatus()
	}
}

func (s *NodeRuntimeStore) refreshOfflineStatus() {
	now := time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, node := range s.nodes {
		if now.Sub(node.ReportedAt) > s.offlineAfter {
			node.Status = NodeStatusOffline
		}
	}
}

// Upsert 写入或更新节点快照（自动注册）。
func (s *NodeRuntimeStore) Upsert(input RuntimeReportInput, wsConnected bool) *NodeRuntimeSnapshot {
	now := time.Now()
	reportedAt := input.ReportedAt
	if reportedAt.IsZero() {
		reportedAt = now
	}
	status := NodeStatusOnline
	if now.Sub(reportedAt) > s.offlineAfter {
		status = NodeStatusOffline
	}

	snapshot := &NodeRuntimeSnapshot{
		NodeID:      input.NodeID,
		NodeName:    input.NodeName,
		ReportedAt:  reportedAt,
		ReceivedAt:  now,
		WSConnected: wsConnected,
		Status:      status,
		Host:        input.Host,
		App:         input.App,
		Pools:       input.Pools,
		Build:       input.Build,
	}
	if snapshot.NodeName == "" {
		snapshot.NodeName = snapshot.NodeID
	}
	if snapshot.Pools == nil {
		snapshot.Pools = map[string]interface{}{}
	}

	s.mu.Lock()
	s.nodes[input.NodeID] = snapshot
	s.mu.Unlock()
	return snapshot.clone()
}

// SetWSConnected 更新节点 WS 连通状态。
func (s *NodeRuntimeStore) SetWSConnected(nodeID string, connected bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if node, ok := s.nodes[nodeID]; ok {
		node.WSConnected = connected
	}
}

// List 返回所有节点快照副本。
func (s *NodeRuntimeStore) List() []NodeRuntimeSnapshot {
	s.refreshOfflineStatusLocked()
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]NodeRuntimeSnapshot, 0, len(s.nodes))
	for _, node := range s.nodes {
		out = append(out, *node.clone())
	}
	return out
}

// Get 返回单节点快照。
func (s *NodeRuntimeStore) Get(nodeID string) (*NodeRuntimeSnapshot, bool) {
	s.refreshOfflineStatusLocked()
	s.mu.RLock()
	defer s.mu.RUnlock()
	node, ok := s.nodes[nodeID]
	if !ok {
		return nil, false
	}
	clone := node.clone()
	return clone, true
}

func (s *NodeRuntimeStore) refreshOfflineStatusLocked() {
	now := time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, node := range s.nodes {
		if now.Sub(node.ReportedAt) > s.offlineAfter {
			node.Status = NodeStatusOffline
		} else {
			node.Status = NodeStatusOnline
		}
	}
}

// GetClusterSummary 聚合集群指标。
func (s *NodeRuntimeStore) GetClusterSummary() ClusterRuntimeSummary {
	s.refreshOfflineStatusLocked()
	s.mu.RLock()
	defer s.mu.RUnlock()

	summary := ClusterRuntimeSummary{TotalNodes: len(s.nodes)}
	for _, node := range s.nodes {
		if node.Status == NodeStatusOnline {
			summary.OnlineNodes++
		}
		summary.TotalChatManagers += node.App.ChatManagerCount
		summary.TotalActiveSessions += node.App.ActiveSessionCount
	}
	summary.DevicesActive5m = s.devicesActive5m
	return summary
}

// SetDevicesActive5m 设置业务活跃设备数缓存。
func (s *NodeRuntimeStore) SetDevicesActive5m(count int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.devicesActive5m = count
	s.devicesCachedAt = time.Now()
}

func (n *NodeRuntimeSnapshot) clone() *NodeRuntimeSnapshot {
	if n == nil {
		return nil
	}
	clone := *n
	if n.Pools != nil {
		clone.Pools = make(map[string]interface{}, len(n.Pools))
		for k, v := range n.Pools {
			clone.Pools[k] = v
		}
	}
	return &clone
}
