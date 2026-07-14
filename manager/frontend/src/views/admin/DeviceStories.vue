<template>
  <div class="device-stories-page">
    <div class="page-header">
      <div class="header-left">
        <el-button @click="$router.back()" :icon="ArrowLeft" circle size="large" />
        <div class="header-context">
          <span class="context-label">设备故事</span>
          <strong class="context-value">{{ deviceLabel }}</strong>
          <p class="context-meta" v-if="deviceSN">SN: {{ deviceSN }}</p>
        </div>
      </div>
      <div class="header-right">
        <el-button :icon="Refresh" @click="loadStories" :loading="loading">刷新</el-button>
        <el-button
          type="danger"
          plain
          :disabled="!storyData?.total"
          :loading="clearing"
          @click="handleClearAll"
        >
          清空故事
        </el-button>
      </div>
    </div>

    <el-alert
      v-if="loadError"
      type="warning"
      :title="loadError"
      show-icon
      :closable="false"
      class="stories-alert"
    />

    <div v-loading="loading" class="stories-body">
      <el-empty v-if="!loading && loadError" description="无法加载设备故事" />

      <div v-else class="stories-panel">
        <div class="summary-row" v-if="storyData">
          <el-tag type="info">共 {{ storyData.total || 0 }} 篇</el-tag>
          <span class="summary-hint">本机播放记录（MySQL playback + Redis 热缓存）；删除不影响共享故事库</span>
        </div>

        <el-table
          :data="storyData?.items || []"
          stripe
          empty-text="暂无故事记录"
          @row-click="openDetail"
          class="stories-table"
        >
          <el-table-column prop="title" label="标题" min-width="160" show-overflow-tooltip />
          <el-table-column label="播放进度" min-width="180">
            <template #default="{ row }">
              <div v-if="row.generation_complete && row.last_position?.progress_available !== false" class="progress-cell">
                <el-progress
                  :percentage="row.last_position?.progress_percent ?? 0"
                  :stroke-width="8"
                  :status="progressStatus(row)"
                />
                <span class="progress-meta">
                  {{ formatProgressMeta(row) }}
                </span>
              </div>
              <el-tag v-else size="small" type="warning">生成中断</el-tag>
            </template>
          </el-table-column>
          <el-table-column label="长度" width="100">
            <template #default="{ row }">
              {{ row.text_length }} 字
            </template>
          </el-table-column>
          <el-table-column label="题材" min-width="100" show-overflow-tooltip>
            <template #default="{ row }">
              {{ row.genre || row.theme || '—' }}
            </template>
          </el-table-column>
          <el-table-column label="年龄段" width="110">
            <template #default="{ row }">
              {{ ageBandLabel(row.age_band) }}
            </template>
          </el-table-column>
          <el-table-column label="状态" width="100">
            <template #default="{ row }">
              <el-tag size="small" :type="statusTagType(row.last_play_status)">
                {{ playStatusLabel(row.last_play_status) }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column label="播放" width="80" align="center">
            <template #default="{ row }">
              {{ row.play_count }}
            </template>
          </el-table-column>
          <el-table-column label="最近播放" width="170">
            <template #default="{ row }">
              {{ formatTime(row.last_played_at) }}
            </template>
          </el-table-column>
          <el-table-column label="操作" width="160" fixed="right">
            <template #default="{ row }">
              <el-button link type="primary" @click.stop="openDetail(row)">详情</el-button>
              <el-button link type="danger" @click.stop="handleDelete(row)">删除</el-button>
            </template>
          </el-table-column>
        </el-table>
      </div>
    </div>

    <el-drawer
      v-model="detailVisible"
      :title="detail?.title || '故事详情'"
      size="520px"
      destroy-on-close
    >
      <div v-loading="detailLoading">
        <template v-if="detail">
          <el-descriptions :column="1" border size="small" class="detail-desc">
            <el-descriptions-item label="故事 ID">{{ detail.story_id }}</el-descriptions-item>
            <el-descriptions-item label="播放进度">
              <template v-if="detail.generation_complete && detail.last_position?.progress_available !== false">
                {{ detail.last_position?.progress_percent ?? 0 }}%
              </template>
              <el-tag v-else size="small" type="warning">生成中断</el-tag>
            </el-descriptions-item>
            <el-descriptions-item label="生成状态">
              {{ detail.generation_complete ? '已完整生成' : '未完成生成' }}
            </el-descriptions-item>
            <el-descriptions-item label="字数">{{ detail.text_length }} 字</el-descriptions-item>
            <el-descriptions-item label="题材">{{ detail.genre || detail.theme || '—' }}</el-descriptions-item>
            <el-descriptions-item label="主题">{{ detail.theme || '—' }}</el-descriptions-item>
            <el-descriptions-item label="风格">{{ detail.style || '—' }}</el-descriptions-item>
            <el-descriptions-item label="年龄段">{{ ageBandLabel(detail.age_band) }}</el-descriptions-item>
            <el-descriptions-item label="模式">{{ detail.mode || '—' }}</el-descriptions-item>
            <el-descriptions-item label="标签">
              <el-tag v-for="tag in detail.tags || []" :key="tag" size="small" class="detail-tag">{{ tag }}</el-tag>
              <span v-if="!(detail.tags || []).length">—</span>
            </el-descriptions-item>
            <el-descriptions-item label="播放次数">{{ detail.play_count }}（完播 {{ detail.complete_count }}）</el-descriptions-item>
            <el-descriptions-item label="创建时间">{{ formatTime(detail.created_at) }}</el-descriptions-item>
            <el-descriptions-item label="最近播放">{{ formatTime(detail.last_played_at) }}</el-descriptions-item>
          </el-descriptions>

          <h4 class="detail-section-title">正文</h4>
          <el-input
            :model-value="detail.full_text || ''"
            type="textarea"
            :rows="16"
            readonly
            placeholder="暂无正文"
          />
          <div class="detail-actions">
            <el-button type="danger" plain :loading="deleting" @click="handleDelete(detail)">
              删除本机记录
            </el-button>
          </div>
        </template>
      </div>
    </el-drawer>
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { ArrowLeft, Refresh } from '@element-plus/icons-vue'
import api from '../../utils/api'
import { formatDeviceNickName } from '../../utils/iotDevice'

const route = useRoute()
const deviceId = computed(() => route.params.id)

const loading = ref(false)
const loadError = ref('')
const storyData = ref(null)
const deviceLabel = ref('设备')
const deviceSN = ref('')

const detailVisible = ref(false)
const detailLoading = ref(false)
const detail = ref(null)
const clearing = ref(false)
const deleting = ref(false)

const ageBandMap = {
  preschool: '学龄前 (3-6)',
  primary_low: '小学低段 (7-9)',
  primary_high: '小学高段 (10-12)',
  junior_high: '初中 (13-15)'
}

const playStatusMap = {
  playing: '播放中',
  completed: '已听完',
  interrupted: '已打断',
  abandoned: '生成失败'
}

const ageBandLabel = (band) => ageBandMap[band] || band || '—'
const playStatusLabel = (status) => playStatusMap[status] || status || '—'

const statusTagType = (status) => {
  switch (status) {
    case 'completed':
      return 'success'
    case 'interrupted':
      return 'warning'
    case 'abandoned':
      return 'danger'
    case 'playing':
      return 'primary'
    default:
      return 'info'
  }
}

const progressStatus = (row) => {
  if (row.last_play_status === 'completed') return 'success'
  if (row.last_play_status === 'interrupted') return 'warning'
  return undefined
}

const formatProgressMeta = (row) => {
  const pos = row.last_position || {}
  return `${pos.progress_percent ?? 0}%`
}

const formatTime = (value) => {
  if (!value) return '—'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return date.toLocaleString('zh-CN', { hour12: false })
}

const loadDeviceMeta = async () => {
  try {
    const response = await api.get('/admin/devices')
    const list = response.data?.data || response.data || []
    const device = list.find((item) => String(item.id) === String(deviceId.value))
    if (device) {
      deviceLabel.value = formatDeviceNickName(device)
      deviceSN.value = device.device_name || ''
    }
  } catch {
    // 列表接口失败时仍尝试加载故事
  }
}

const loadStories = async () => {
  loading.value = true
  loadError.value = ''
  try {
    const response = await api.get(`/admin/devices/${deviceId.value}/stories`, { params: { limit: 50 } })
    storyData.value = response.data?.data || null
    if (storyData.value?.device_sn) {
      deviceSN.value = storyData.value.device_sn
    }
  } catch (error) {
    storyData.value = null
    loadError.value = error.response?.data?.error || '加载设备故事失败'
  } finally {
    loading.value = false
  }
}

const openDetail = async (row) => {
  if (!row?.story_id) return
  detailVisible.value = true
  detailLoading.value = true
  detail.value = null
  try {
    const response = await api.get(`/admin/devices/${deviceId.value}/stories/${row.story_id}`)
    detail.value = response.data?.data || null
  } catch (error) {
    ElMessage.error(error.response?.data?.error || '加载故事详情失败')
    detailVisible.value = false
  } finally {
    detailLoading.value = false
  }
}

const handleDelete = async (row) => {
  if (!row?.story_id) return
  try {
    await ElMessageBox.confirm(
      `确定删除「${row.title || row.story_id}」的本机播放记录？\n不会删除共享故事库中的正文。`,
      '删除确认',
      { type: 'warning', confirmButtonText: '删除', cancelButtonText: '取消' }
    )
  } catch {
    return
  }
  deleting.value = true
  try {
    await api.delete(`/admin/devices/${deviceId.value}/stories/${row.story_id}`)
    ElMessage.success('已删除本机记录')
    detailVisible.value = false
    await loadStories()
  } catch (error) {
    ElMessage.error(error.response?.data?.error || '删除失败')
  } finally {
    deleting.value = false
  }
}

const handleClearAll = async () => {
  const total = storyData.value?.total || 0
  try {
    await ElMessageBox.confirm(
      `确定清空该设备全部 ${total} 条故事播放记录？\n仅清除本机进度与缓存，不影响共享故事库；清空后近 7 天共享排斥也会重置。`,
      '清空确认',
      { type: 'warning', confirmButtonText: '全部清空', cancelButtonText: '取消' }
    )
  } catch {
    return
  }
  clearing.value = true
  try {
    const { data } = await api.delete(`/admin/devices/${deviceId.value}/stories`)
    const n = data?.data?.playback_deleted ?? 0
    ElMessage.success(`已清空（playback ${n} 条）`)
    detailVisible.value = false
    await loadStories()
  } catch (error) {
    ElMessage.error(error.response?.data?.error || '清空失败')
  } finally {
    clearing.value = false
  }
}

onMounted(async () => {
  await loadDeviceMeta()
  await loadStories()
})
</script>

<style scoped>
.device-stories-page {
  padding: 0;
  min-height: 100%;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
  padding: 20px;
  background: rgba(255, 255, 255, 0.88);
  border: 1px solid rgba(255, 255, 255, 0.9);
  border-radius: var(--apple-radius-lg);
  box-shadow: var(--apple-shadow-md);
}

.header-left {
  display: flex;
  align-items: center;
  gap: 16px;
}

.header-right {
  display: flex;
  align-items: center;
  gap: 12px;
}

.header-context {
  display: grid;
  gap: 4px;
}

.context-label {
  color: var(--apple-text-secondary);
  font-size: 12px;
  font-weight: 600;
}

.context-value {
  color: var(--apple-text);
  font-size: 16px;
}

.context-meta {
  margin: 0;
  color: var(--apple-text-secondary);
  font-size: 13px;
}

.stories-alert {
  margin-bottom: 16px;
}

.stories-body {
  padding: 0 4px 24px;
  min-height: 320px;
}

.stories-panel {
  background: rgba(255, 255, 255, 0.88);
  border: 1px solid rgba(255, 255, 255, 0.9);
  border-radius: var(--apple-radius-lg);
  box-shadow: var(--apple-shadow-md);
  padding: 16px 20px 20px;
}

.summary-row {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 16px;
}

.summary-hint {
  color: var(--apple-text-secondary);
  font-size: 13px;
}

.stories-table :deep(.el-table__row) {
  cursor: pointer;
}

.progress-cell {
  display: grid;
  gap: 4px;
}

.progress-meta {
  font-size: 12px;
  color: var(--apple-text-secondary);
}

.text-muted {
  color: var(--apple-text-secondary);
  font-size: 12px;
}

.detail-actions {
  margin-top: 16px;
  display: flex;
  justify-content: flex-end;
}

.detail-desc {
  margin-bottom: 16px;
}

.detail-tag {
  margin-right: 6px;
}

.detail-section-title {
  margin: 0 0 8px;
  font-size: 14px;
  color: var(--apple-text);
}
</style>
