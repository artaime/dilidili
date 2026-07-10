package observability

import (
	"os"
	"runtime"
	"time"

	"github.com/shirou/gopsutil/v4/process"
)

var processStartAt = time.Now()

// CollectAppMetrics 从 AppMetricsProvider 与 runtime 采集应用指标。
func CollectAppMetrics(provider AppMetricsProvider) AppMetrics {
	metrics := AppMetrics{
		UptimeSec: int64(time.Since(processStartAt).Seconds()),
		Goroutines: runtime.NumGoroutine(),
	}
	if provider != nil {
		metrics.ChatManagerCount = provider.GetChatManagerCount()
		metrics.ActiveSessionCount = provider.GetActiveSessionCount()
		metrics.TransportWS, metrics.TransportMqttUdp = provider.GetTransportBreakdown()
	}
	if p, err := process.NewProcess(int32(os.Getpid())); err == nil {
		if info, err := p.MemoryInfo(); err == nil && info != nil {
			metrics.RSSMB = info.RSS / 1024 / 1024
		}
	}
	return metrics
}
