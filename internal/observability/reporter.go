package observability

import (
	"context"
	"runtime"
	"sync"
	"time"

	"dili-esp32-server-golang/internal/components/http"
	"dili-esp32-server-golang/internal/pool"
	"dili-esp32-server-golang/internal/util"
	log "dili-esp32-server-golang/logger"

	"github.com/spf13/viper"
)

var (
	globalReporter     *RuntimeReporter
	globalReporterOnce sync.Once
)

// StartRuntimeReporter 启动运行时指标定时上报。
func StartRuntimeReporter(ctx context.Context, provider AppMetricsProvider) {
	globalReporterOnce.Do(func() {
		globalReporter = newRuntimeReporter(provider)
	})
	if globalReporter == nil || !globalReporter.enabled {
		log.Info("运行时监控上报已禁用")
		return
	}
	globalReporter.start(ctx)
}

type RuntimeReporter struct {
	client        *http.ManagerClient
	enabled       bool
	interval      time.Duration
	hostCollector *HostCollector
	provider      AppMetricsProvider
}

func newRuntimeReporter(provider AppMetricsProvider) *RuntimeReporter {
	baseURL := util.GetBackendURL()
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}
	enabled := true
	if viper.IsSet("runtime_report.enabled") {
		enabled = viper.GetBool("runtime_report.enabled")
	}
	interval := viper.GetDuration("runtime_report.interval")
	if interval <= 0 {
		interval = 5 * time.Second
	}
	diskPath := viper.GetString("runtime_report.disk_path")
	if diskPath == "" {
		diskPath = "/"
	}
	return &RuntimeReporter{
		client: http.NewManagerClient(http.ManagerClientConfig{
			BaseURL:    baseURL,
			AuthToken:  util.GetManagerAuthToken(),
			Timeout:    5 * time.Second,
			MaxRetries: 2,
		}),
		enabled:       enabled,
		interval:      interval,
		hostCollector: NewHostCollector(diskPath),
		provider:      provider,
	}
}

func (r *RuntimeReporter) start(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(r.interval)
		defer ticker.Stop()
		log.Infof("运行时监控上报已启动 interval=%s node_id=%s", r.interval, util.GetNodeID())
		for {
			select {
			case <-ctx.Done():
				log.Debugf("运行时监控上报已停止")
				return
			case <-ticker.C:
				r.report(ctx)
			}
		}
	}()
}

func (r *RuntimeReporter) report(ctx context.Context) {
	report := RuntimeReport{
		NodeID:     util.GetNodeID(),
		NodeName:   util.GetNodeName(),
		ReportedAt: time.Now(),
		Host:       r.hostCollector.Collect(),
		App:        CollectAppMetrics(r.provider),
		Pools:      pool.GetStats(),
		Build: BuildInfo{
			Version:   viper.GetString("server.version"),
			GoVersion: runtime.Version(),
		},
	}
	if report.Build.Version == "" {
		report.Build.Version = "unknown"
	}

	body := map[string]interface{}{
		"node_id":     report.NodeID,
		"node_name":   report.NodeName,
		"reported_at": report.ReportedAt.Format(time.RFC3339),
		"host":        report.Host,
		"app":         report.App,
		"pools":       report.Pools,
		"build":       report.Build,
	}
	if err := r.client.DoRequest(ctx, http.RequestOptions{
		Method: "POST",
		Path:   "/api/internal/runtime/report",
		Body:   body,
	}); err != nil {
		log.Warnf("运行时监控上报失败: %v", err)
		return
	}

	// 兼容旧资源池上报接口（双写一个版本周期）
	if len(report.Pools) > 0 {
		_ = r.client.DoRequest(ctx, http.RequestOptions{
			Method: "POST",
			Path:   "/api/internal/pool/stats",
			Body: map[string]interface{}{
				"stats": report.Pools,
			},
		})
	}
}

func (r *RuntimeReporter) BuildReport() RuntimeReport {
	return RuntimeReport{
		NodeID:     util.GetNodeID(),
		NodeName:   util.GetNodeName(),
		ReportedAt: time.Now(),
		Host:       r.hostCollector.Collect(),
		App:        CollectAppMetrics(r.provider),
		Pools:      pool.GetStats(),
		Build: BuildInfo{
			Version:   viper.GetString("server.version"),
			GoVersion: runtime.Version(),
		},
	}
}
