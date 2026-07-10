<template>
  <div class="device-memory-page">
    <div class="page-header">
      <div class="header-left">
        <el-button @click="$router.back()" :icon="ArrowLeft" circle size="large" />
        <div class="header-context">
          <span class="context-label">设备长期记忆</span>
          <strong class="context-value">{{ deviceLabel }}</strong>
          <p class="context-meta" v-if="deviceSN">SN: {{ deviceSN }}</p>
        </div>
      </div>
      <div class="header-right">
        <el-button :icon="Refresh" @click="loadMemory" :loading="loading">刷新</el-button>
        <el-button
          type="danger"
          plain
          :disabled="!memoryData || loading || !!loadError"
          @click="confirmClearMemory"
        >
          清空 Memobase 长期记忆
        </el-button>
      </div>
    </div>

    <el-alert
      type="info"
      title="此页仅清理 Memobase Profile/Event/Context，不含对话记录、Redis 故事缓存等。全量清理请使用设备列表「出厂重置」。"
      show-icon
      :closable="false"
      class="memory-alert"
    />

    <el-alert
      v-if="loadError"
      type="warning"
      :title="loadError"
      show-icon
      :closable="false"
      class="memory-alert"
    />

    <el-alert
      v-if="!loadError && memoryData?.using_legacy"
      type="info"
      title="当前展示的是历史记忆键（修复前 double-UUID 写入的数据）。新会话写入将使用新键。"
      show-icon
      :closable="false"
      class="memory-alert"
    />

    <div v-loading="loading" class="memory-body">
      <el-empty
        v-if="!loading && loadError"
        description="无法加载设备记忆"
      />

      <el-tabs v-else-if="memoryData" v-model="activeTab" class="memory-tabs">
        <el-tab-pane label="用户画像" name="profiles">
          <el-table :data="memoryData.profiles || []" stripe empty-text="暂无画像数据">
            <el-table-column prop="topic" label="主题" width="120" />
            <el-table-column prop="sub_topic" label="子主题" width="140" />
            <el-table-column prop="content" label="内容" min-width="240" show-overflow-tooltip />
            <el-table-column prop="updated_at" label="更新时间" width="180" />
          </el-table>
        </el-tab-pane>

        <el-tab-pane label="相关事件" name="events">
          <el-table :data="memoryData.events || []" stripe empty-text="暂无事件数据">
            <el-table-column prop="event_tip" label="事件摘要" min-width="280" show-overflow-tooltip />
            <el-table-column label="标签" min-width="200">
              <template #default="{ row }">
                <el-tag
                  v-for="tag in row.tags || []"
                  :key="tag"
                  size="small"
                  class="event-tag"
                >
                  {{ tag }}
                </el-tag>
                <span v-if="!(row.tags || []).length" class="text-muted">—</span>
              </template>
            </el-table-column>
            <el-table-column prop="created_at" label="创建时间" width="180" />
          </el-table>
        </el-tab-pane>

        <el-tab-pane label="上下文预览" name="context">
          <p class="context-hint">
            Memobase 注入 LLM 的上下文原文，便于排查「为何未主动提及记忆」等问题。
          </p>
          <el-input
            :model-value="memoryData.context || ''"
            type="textarea"
            :rows="18"
            readonly
            placeholder="暂无上下文"
          />
        </el-tab-pane>
      </el-tabs>
    </div>
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
const memoryData = ref(null)
const activeTab = ref('profiles')
const deviceLabel = ref('设备')
const deviceSN = ref('')

const loadDeviceMeta = async () => {
  try {
    const response = await api.get('/admin/devices')
    const list = response.data?.data || response.data || []
    const device = list.find((item) => String(item.id) === String(deviceId.value))
    if (device) {
      deviceLabel.value = formatDeviceNickName(device)
      deviceSN.value = device.device_name || ''
    }
  } catch (error) {
    // 记忆接口也会返回 device_sn
  }
}

const loadMemory = async () => {
  loading.value = true
  loadError.value = ''
  try {
    const response = await api.get(`/admin/devices/${deviceId.value}/memory`)
    memoryData.value = response.data?.data || null
    if (memoryData.value?.device_sn) {
      deviceSN.value = memoryData.value.device_sn
    }
  } catch (error) {
    memoryData.value = null
    loadError.value = error.response?.data?.error || '加载设备记忆失败'
  } finally {
    loading.value = false
  }
}

const confirmClearMemory = async () => {
  try {
    await ElMessageBox.confirm(
      '将永久删除该设备在 Memobase 中的 Profile、Event 与 Context，且不可恢复。不含对话记录、Redis 与故事；全量清理请使用设备列表「出厂重置」。确定继续？',
      '清空 Memobase 长期记忆',
      {
        confirmButtonText: '清空',
        cancelButtonText: '取消',
        type: 'warning',
        confirmButtonClass: 'el-button--danger'
      }
    )
  } catch {
    return
  }

  loading.value = true
  try {
    await api.delete(`/admin/devices/${deviceId.value}/memory`)
    ElMessage.success('Memobase 长期记忆已清空')
    await loadMemory()
  } catch (error) {
    ElMessage.error(error.response?.data?.error || '清空失败')
  } finally {
    loading.value = false
  }
}

onMounted(async () => {
  await loadDeviceMeta()
  await loadMemory()
})
</script>

<style scoped>
.device-memory-page {
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

.memory-alert {
  margin-bottom: 16px;
}

.memory-body {
  padding: 0 4px 24px;
  min-height: 320px;
}

.memory-tabs {
  background: rgba(255, 255, 255, 0.88);
  border: 1px solid rgba(255, 255, 255, 0.9);
  border-radius: var(--apple-radius-lg);
  box-shadow: var(--apple-shadow-md);
  padding: 16px 20px 20px;
}

.context-hint {
  margin: 0 0 12px;
  color: var(--apple-text-secondary);
  font-size: 13px;
}

.event-tag {
  margin-right: 6px;
  margin-bottom: 4px;
}

.text-muted {
  color: var(--apple-text-secondary);
}
</style>
