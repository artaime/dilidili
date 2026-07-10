package observability

import (
	"sync"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/shirou/gopsutil/v4/net"
	log "dili-esp32-server-golang/logger"
)

// HostCollector 采集主机 CPU/内存/磁盘/网络指标。
type HostCollector struct {
	diskPath string

	mu           sync.Mutex
	lastNetAt    time.Time
	lastNetRx    uint64
	lastNetTx    uint64
}

func NewHostCollector(diskPath string) *HostCollector {
	if diskPath == "" {
		diskPath = "/"
	}
	return &HostCollector{diskPath: diskPath}
}

func (c *HostCollector) Collect() HostMetrics {
	metrics := HostMetrics{}
	if percents, err := cpu.Percent(0, false); err == nil && len(percents) > 0 {
		metrics.CPUPercent = percents[0]
	} else if err != nil {
		log.Warnf("采集 CPU 指标失败: %v", err)
	}

	if vm, err := mem.VirtualMemory(); err == nil {
		metrics.MemTotalMB = vm.Total / 1024 / 1024
		metrics.MemUsedMB = vm.Used / 1024 / 1024
		metrics.MemPercent = vm.UsedPercent
	} else {
		log.Warnf("采集内存指标失败: %v", err)
	}

	if usage, err := disk.Usage(c.diskPath); err == nil {
		metrics.DiskTotalGB = float64(usage.Total) / 1024 / 1024 / 1024
		metrics.DiskUsedGB = float64(usage.Used) / 1024 / 1024 / 1024
		metrics.DiskPercent = usage.UsedPercent
	} else {
		log.Warnf("采集磁盘指标失败 path=%s: %v", c.diskPath, err)
	}

	rxBps, txBps := c.collectNetworkBps()
	metrics.NetRxBps = rxBps
	metrics.NetTxBps = txBps
	return metrics
}

func (c *HostCollector) collectNetworkBps() (rxBps, txBps float64) {
	counters, err := net.IOCounters(false)
	if err != nil || len(counters) == 0 {
		if err != nil {
			log.Warnf("采集网络指标失败: %v", err)
		}
		return 0, 0
	}

	var totalRx, totalTx uint64
	for _, counter := range counters {
		totalRx += counter.BytesRecv
		totalTx += counter.BytesSent
	}

	now := time.Now()
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.lastNetAt.IsZero() {
		elapsed := now.Sub(c.lastNetAt).Seconds()
		if elapsed > 0 {
			rxBps = float64(totalRx-c.lastNetRx) / elapsed
			txBps = float64(totalTx-c.lastNetTx) / elapsed
		}
	}
	c.lastNetAt = now
	c.lastNetRx = totalRx
	c.lastNetTx = totalTx
	return rxBps, txBps
}
