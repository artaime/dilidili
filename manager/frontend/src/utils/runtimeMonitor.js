export function formatPoolStats(stats) {
  if (!stats || typeof stats !== 'object') {
    return []
  }
  return Object.entries(stats).map(([poolKey, poolStats]) => ({
    poolKey,
    total: poolStats?.total_resources || 0,
    available: poolStats?.available_resources || 0,
    inUse: poolStats?.in_use_resources || 0,
    maxSize: poolStats?.max_size || 0,
    minSize: poolStats?.min_size || 0,
    maxIdle: poolStats?.max_idle || 0,
    isClosed: poolStats?.is_closed || false
  }))
}

export function formatBytesPerSec(value) {
  const n = Number(value) || 0
  if (n >= 1024 * 1024) {
    return `${(n / 1024 / 1024).toFixed(1)} MB/s`
  }
  if (n >= 1024) {
    return `${(n / 1024).toFixed(1)} KB/s`
  }
  return `${n.toFixed(1)} B/s`
}

export function formatPercent(value, digits = 1) {
  const n = Number(value) || 0
  return `${n.toFixed(digits)}%`
}

/** 指标超过阈值时显示警告图标（百分比类） */
export const METRIC_WARNING_THRESHOLDS = {
  cpu: 85,
  mem: 90,
  disk: 85
}

export function isMetricWarning(metric, value) {
  const threshold = METRIC_WARNING_THRESHOLDS[metric]
  if (threshold == null) return false
  const n = Number(value) || 0
  return n >= threshold
}

/** 资源池使用率 ≥90% 时警告 */
export function isPoolUsageWarning(inUse, maxSize) {
  const max = Number(maxSize) || 0
  if (max <= 0) return false
  return (Number(inUse) || 0) / max >= 0.9
}

export function progressBarColor(metric, value) {
  return isMetricWarning(metric, value) ? '#E6A23C' : ''
}

/** 进度条用：限制 0–100 并保留 1 位小数 */
export function clampPercent(value) {
  const n = Number(value) || 0
  return Math.min(100, Math.max(0, Math.round(n * 10) / 10))
}

/** el-progress 的 format 回调 */
export function progressFormat(percentage) {
  return `${Number(percentage).toFixed(1)}%`
}

export function formatTime(timestamp) {
  if (!timestamp) {
    return '-'
  }
  const date = new Date(timestamp)
  if (Number.isNaN(date.getTime())) {
    return '-'
  }
  return date.toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit'
  })
}

export function nodeStatusTag(status) {
  return status === 'online' ? 'success' : 'danger'
}

export function nodeStatusLabel(status) {
  return status === 'online' ? '在线' : '离线'
}
