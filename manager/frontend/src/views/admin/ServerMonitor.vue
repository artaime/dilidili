<template>
  <div class="server-monitor">
    <el-card class="summary-card">
      <template #header>
        <div class="card-header">
          <span>服务节点监控</span>
          <div class="header-actions">
            <el-tag size="small" :type="connectionTagType">{{ connectionLabel }}</el-tag>
            <span class="updated-at">更新于 {{ formatTime(lastUpdatedAt) }}</span>
            <el-button type="primary" size="small" @click="refresh">
              <el-icon><Refresh /></el-icon>
              刷新
            </el-button>
          </div>
        </div>
      </template>

      <el-row :gutter="16">
        <el-col :span="6">
          <el-statistic title="在线节点" :value="summary.online_nodes">
            <template #suffix>/ {{ summary.total_nodes }}</template>
          </el-statistic>
        </el-col>
        <el-col :span="6">
          <el-statistic title="连接在线" :value="summary.total_chat_managers" />
        </el-col>
        <el-col :span="6">
          <el-statistic title="会话活跃" :value="summary.total_active_sessions" />
        </el-col>
        <el-col :span="6">
          <el-statistic title="设备活跃(5min)" :value="summary.devices_active_5m" />
        </el-col>
      </el-row>
    </el-card>

    <el-row :gutter="16" class="node-grid">
      <el-col
        v-for="node in nodes"
        :key="node.node_id"
        :xs="24"
        :sm="12"
        :lg="8"
      >
        <el-card
          class="node-card"
          :class="{ selected: selectedNodeId === node.node_id }"
          shadow="hover"
          @click="selectNode(node.node_id)"
        >
          <div class="node-card-header">
            <div>
              <h3>{{ node.node_name || node.node_id }}</h3>
              <p class="node-id">{{ node.node_id }}</p>
            </div>
            <div class="node-tags">
              <el-tag :type="nodeStatusTag(node.status)" size="small">{{ nodeStatusLabel(node.status) }}</el-tag>
              <el-tag :type="node.ws_connected ? 'success' : 'info'" size="small">
                WS {{ node.ws_connected ? '已连' : '未连' }}
              </el-tag>
            </div>
          </div>

          <div class="node-metrics">
            <div class="metric-row">
              <span>连接 / 活跃</span>
              <strong>{{ node.app?.chat_manager_count || 0 }} / {{ node.app?.active_session_count || 0 }}</strong>
            </div>
            <div class="metric-row metric-row-progress">
              <span>CPU</span>
              <div class="metric-progress-wrap">
                <el-progress
                  class="metric-progress"
                  :percentage="clampPercent(node.host?.cpu_percent)"
                  :stroke-width="8"
                  :show-text="false"
                  :color="progressBarColor('cpu', node.host?.cpu_percent)"
                />
                <span class="metric-value" :class="{ 'is-warning': isMetricWarning('cpu', node.host?.cpu_percent) }">
                  {{ formatPercent(node.host?.cpu_percent) }}
                  <el-icon v-if="isMetricWarning('cpu', node.host?.cpu_percent)" class="warn-icon"><WarningFilled /></el-icon>
                </span>
              </div>
            </div>
            <div class="metric-row metric-row-progress">
              <span>内存</span>
              <div class="metric-progress-wrap">
                <el-progress
                  class="metric-progress"
                  :percentage="clampPercent(node.host?.mem_percent)"
                  :stroke-width="8"
                  :show-text="false"
                  :color="progressBarColor('mem', node.host?.mem_percent)"
                />
                <span class="metric-value" :class="{ 'is-warning': isMetricWarning('mem', node.host?.mem_percent) }">
                  {{ formatPercent(node.host?.mem_percent) }}
                  <el-icon v-if="isMetricWarning('mem', node.host?.mem_percent)" class="warn-icon"><WarningFilled /></el-icon>
                </span>
              </div>
            </div>
            <div class="metric-row metric-row-progress">
              <span>磁盘</span>
              <div class="metric-progress-wrap">
                <el-progress
                  class="metric-progress"
                  :percentage="clampPercent(node.host?.disk_percent)"
                  :stroke-width="8"
                  :show-text="false"
                  :color="progressBarColor('disk', node.host?.disk_percent)"
                />
                <span class="metric-value" :class="{ 'is-warning': isMetricWarning('disk', node.host?.disk_percent) }">
                  {{ formatPercent(node.host?.disk_percent) }}
                  <el-icon v-if="isMetricWarning('disk', node.host?.disk_percent)" class="warn-icon"><WarningFilled /></el-icon>
                </span>
              </div>
            </div>
            <div class="metric-row muted">
              <span>上报时间</span>
              <span>{{ formatTime(node.reported_at) }}</span>
            </div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <el-empty v-if="!nodes.length" description="暂无主服务节点上报，请确认主服务已配置 node_id 并连接管理端" />

    <el-drawer
      v-model="drawerVisible"
      :title="selectedNode ? (selectedNode.node_name || selectedNode.node_id) : '节点详情'"
      size="520px"
    >
      <template v-if="selectedNode">
        <el-descriptions :column="1" border>
          <el-descriptions-item label="节点 ID">{{ selectedNode.node_id }}</el-descriptions-item>
          <el-descriptions-item label="状态">
            <el-tag :type="nodeStatusTag(selectedNode.status)">{{ nodeStatusLabel(selectedNode.status) }}</el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="WS 连接">{{ selectedNode.ws_connected ? '已连接' : '未连接' }}</el-descriptions-item>
          <el-descriptions-item label="运行时长">{{ formatUptime(selectedNode.app?.uptime_sec) }}</el-descriptions-item>
          <el-descriptions-item label="Goroutine">{{ selectedNode.app?.goroutines || 0 }}</el-descriptions-item>
          <el-descriptions-item label="进程内存">{{ selectedNode.app?.rss_mb || 0 }} MB</el-descriptions-item>
          <el-descriptions-item label="CPU 使用">
            <span class="metric-value-inline" :class="{ 'is-warning': isMetricWarning('cpu', selectedNode.host?.cpu_percent) }">
              {{ formatPercent(selectedNode.host?.cpu_percent) }}
              <el-icon v-if="isMetricWarning('cpu', selectedNode.host?.cpu_percent)" class="warn-icon"><WarningFilled /></el-icon>
            </span>
          </el-descriptions-item>
          <el-descriptions-item label="内存使用">
            <span class="metric-value-inline" :class="{ 'is-warning': isMetricWarning('mem', selectedNode.host?.mem_percent) }">
              {{ formatPercent(selectedNode.host?.mem_percent) }}
              ({{ (selectedNode.host?.mem_used_mb || 0).toFixed(1) }} / {{ (selectedNode.host?.mem_total_mb || 0).toFixed(1) }} MB)
              <el-icon v-if="isMetricWarning('mem', selectedNode.host?.mem_percent)" class="warn-icon"><WarningFilled /></el-icon>
            </span>
          </el-descriptions-item>
          <el-descriptions-item label="磁盘使用">
            <span class="metric-value-inline" :class="{ 'is-warning': isMetricWarning('disk', selectedNode.host?.disk_percent) }">
              {{ formatPercent(selectedNode.host?.disk_percent) }}
              ({{ (selectedNode.host?.disk_used_gb || 0).toFixed(1) }} / {{ (selectedNode.host?.disk_total_gb || 0).toFixed(1) }} GB)
              <el-icon v-if="isMetricWarning('disk', selectedNode.host?.disk_percent)" class="warn-icon"><WarningFilled /></el-icon>
            </span>
          </el-descriptions-item>
          <el-descriptions-item label="网络下行">{{ formatBytesPerSec(selectedNode.host?.net_rx_bps) }}</el-descriptions-item>
          <el-descriptions-item label="网络上行">{{ formatBytesPerSec(selectedNode.host?.net_tx_bps) }}</el-descriptions-item>
          <el-descriptions-item label="协议分布">
            WebSocket {{ selectedNode.app?.transport_ws || 0 }} · MQTT-UDP {{ selectedNode.app?.transport_mqtt_udp || 0 }}
          </el-descriptions-item>
          <el-descriptions-item label="版本">{{ selectedNode.build?.version || '-' }} ({{ selectedNode.build?.go_version || '-' }})</el-descriptions-item>
        </el-descriptions>

        <el-divider>资源池</el-divider>
        <el-table :data="formatPoolStats(selectedNode.pools)" border stripe>
          <el-table-column prop="poolKey" label="资源池" min-width="160" />
          <el-table-column label="使用中" width="110">
            <template #default="{ row }">
              <span class="metric-value-inline" :class="{ 'is-warning': isPoolUsageWarning(row.inUse, row.maxSize) }">
                {{ row.inUse }}
                <el-icon v-if="isPoolUsageWarning(row.inUse, row.maxSize)" class="warn-icon"><WarningFilled /></el-icon>
              </span>
            </template>
          </el-table-column>
          <el-table-column prop="available" label="可用" width="90" />
          <el-table-column prop="maxSize" label="最大容量" width="100" />
        </el-table>
      </template>
    </el-drawer>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { Refresh, WarningFilled } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { useRuntimeMonitor } from '@/composables/useRuntimeMonitor'
import {
  clampPercent,
  formatBytesPerSec,
  formatPercent,
  formatPoolStats,
  formatTime,
  isMetricWarning,
  isPoolUsageWarning,
  nodeStatusLabel,
  nodeStatusTag,
  progressBarColor
} from '@/utils/runtimeMonitor'

const {
  nodes,
  summary,
  selectedNode,
  selectedNodeId,
  connectionState,
  lastUpdatedAt,
  connect,
  loadSnapshot,
  selectNode
} = useRuntimeMonitor()

const drawerVisible = computed({
  get: () => !!selectedNodeId.value,
  set: (visible) => {
    if (!visible) {
      selectNode('')
    }
  }
})

const connectionLabel = computed(() => {
  switch (connectionState.value) {
    case 'connected':
      return 'SSE 已连接'
    case 'connecting':
      return '连接中'
    case 'reconnecting':
      return '重连中'
    case 'polling':
      return '轮询模式'
    case 'error':
      return '连接异常'
    default:
      return '未连接'
  }
})

const connectionTagType = computed(() => {
  if (connectionState.value === 'connected') return 'success'
  if (connectionState.value === 'error') return 'danger'
  if (connectionState.value === 'polling') return 'warning'
  return 'info'
})

const formatUptime = (seconds) => {
  const total = Number(seconds) || 0
  const days = Math.floor(total / 86400)
  const hours = Math.floor((total % 86400) / 3600)
  const minutes = Math.floor((total % 3600) / 60)
  if (days > 0) return `${days}天 ${hours}小时`
  if (hours > 0) return `${hours}小时 ${minutes}分钟`
  return `${minutes}分钟`
}

const refresh = async () => {
  try {
    if (connectionState.value === 'connected' || connectionState.value === 'reconnecting') {
      await connect()
    } else {
      await loadSnapshot()
    }
    ElMessage.success('已刷新')
  } catch (error) {
    console.error('刷新运行时监控失败:', error)
    ElMessage.error('刷新失败')
  }
}
</script>

<style scoped>
.server-monitor {
  padding: 20px;
}

.card-header,
.node-card-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 12px;
}

.header-actions {
  display: flex;
  align-items: center;
  gap: 10px;
}

.updated-at {
  color: #909399;
  font-size: 12px;
}

.summary-card {
  margin-bottom: 16px;
}

.node-grid {
  margin-top: 4px;
}

.node-card {
  margin-bottom: 16px;
  cursor: pointer;
}

.node-card.selected {
  border-color: var(--el-color-primary);
}

.node-card h3 {
  margin: 0;
  font-size: 16px;
}

.node-id {
  margin: 4px 0 0;
  color: #909399;
  font-size: 12px;
}

.node-tags {
  display: flex;
  gap: 6px;
  flex-wrap: wrap;
}

.node-metrics {
  margin-top: 14px;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.metric-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
}

.metric-row.muted {
  color: #909399;
  font-size: 12px;
}

.metric-row-progress {
  align-items: center;
}

.metric-row-progress > span {
  flex: 0 0 40px;
}

.metric-progress-wrap {
  flex: 1;
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 8px;
}

.metric-progress {
  flex: 1;
  min-width: 0;
}

.metric-value,
.metric-value-inline {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  flex-shrink: 0;
  font-size: 13px;
  color: #606266;
}

.metric-value.is-warning,
.metric-value-inline.is-warning {
  color: #E6A23C;
  font-weight: 600;
}

.warn-icon {
  color: #E6A23C;
  font-size: 14px;
}
</style>
