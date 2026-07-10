package observability

import "time"

// RuntimeReport 主服务运行时上报载荷。
type RuntimeReport struct {
	NodeID     string                 `json:"node_id"`
	NodeName   string                 `json:"node_name"`
	ReportedAt time.Time              `json:"reported_at"`
	Host       HostMetrics            `json:"host"`
	App        AppMetrics             `json:"app"`
	Pools      map[string]interface{} `json:"pools"`
	Build      BuildInfo              `json:"build"`
}

type HostMetrics struct {
	CPUPercent   float64 `json:"cpu_percent"`
	MemTotalMB   uint64  `json:"mem_total_mb"`
	MemUsedMB    uint64  `json:"mem_used_mb"`
	MemPercent   float64 `json:"mem_percent"`
	DiskTotalGB  float64 `json:"disk_total_gb"`
	DiskUsedGB   float64 `json:"disk_used_gb"`
	DiskPercent  float64 `json:"disk_percent"`
	NetRxBps     float64 `json:"net_rx_bps"`
	NetTxBps     float64 `json:"net_tx_bps"`
}

type AppMetrics struct {
	UptimeSec          int64 `json:"uptime_sec"`
	Goroutines         int   `json:"goroutines"`
	RSSMB              uint64 `json:"rss_mb"`
	ChatManagerCount   int   `json:"chat_manager_count"`
	ActiveSessionCount int   `json:"active_session_count"`
	TransportWS        int   `json:"transport_ws"`
	TransportMqttUdp   int   `json:"transport_mqtt_udp"`
}

type BuildInfo struct {
	Version    string `json:"version"`
	GoVersion  string `json:"go_version"`
}

// AppMetricsProvider 由主服务 App 实现，供采集器读取应用指标。
type AppMetricsProvider interface {
	GetChatManagerCount() int
	GetActiveSessionCount() int
	GetTransportBreakdown() (ws, mqttUdp int)
}
