<template>
  <div class="device-conversation-page">
    <div class="page-header">
      <div class="header-left">
        <el-button @click="$router.back()" :icon="ArrowLeft" circle size="large" />
        <div class="header-context">
          <span class="context-label">设备对话记录</span>
          <strong class="context-value">{{ deviceLabel }}</strong>
          <p class="context-meta" v-if="deviceSN">SN: {{ deviceSN }}</p>
        </div>
      </div>
      <div class="header-right">
        <el-date-picker
          v-model="selectedDate"
          type="date"
          placeholder="按日期"
          format="YYYY-MM-DD"
          value-format="YYYY-MM-DD"
          clearable
          :teleported="true"
          @change="handleDateChange"
        />
      </div>
    </div>

    <ConversationTimeline
      ref="timelineRef"
      :records="records"
      :loading="loading"
      :loading-older="loadingOlder"
      :loading-newer="loadingNewer"
      :has-older="hasOlder"
      :has-newer="hasNewer"
      :playing-key="playingKey"
      @load-older="loadOlder"
      @load-newer="loadNewer"
      @play-audio="handlePlayAudio"
      @open-story="openStoryDetail"
    />

    <el-drawer
      v-model="storyDetailVisible"
      :title="storyDetail?.title || '故事详情'"
      size="520px"
      destroy-on-close
    >
      <div v-loading="storyDetailLoading">
        <template v-if="storyDetail">
          <p class="story-meta">字数：{{ storyDetail.text_length || (storyDetail.full_text || '').length }}</p>
          <el-input
            :model-value="storyDetail.full_text || ''"
            type="textarea"
            :rows="18"
            readonly
            placeholder="暂无正文"
          />
        </template>
      </div>
    </el-drawer>
  </div>
</template>

<script setup>
import { computed, nextTick, onBeforeUnmount, onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage } from 'element-plus'
import { ArrowLeft } from '@element-plus/icons-vue'
import api from '../../utils/api'
import ConversationTimeline from '../../components/conversation/ConversationTimeline.vue'
import { formatDeviceNickName } from '../../utils/iotDevice'

const route = useRoute()
const deviceId = computed(() => route.params.id)

const timelineRef = ref(null)
const records = ref([])
const loading = ref(false)
const loadingOlder = ref(false)
const loadingNewer = ref(false)
const hasOlder = ref(false)
const hasNewer = ref(false)
const selectedDate = ref('')
const deviceLabel = ref('设备')
const deviceSN = ref('')
const playingKey = ref('')
const audioEl = ref(null)
const audioBlobUrls = ref({})
const storyDetailVisible = ref(false)
const storyDetailLoading = ref(false)
const storyDetail = ref(null)

const recordKey = (record) => `${record.type}_${record.id}`

const openStoryDetail = async ({ storyId }) => {
  if (!storyId || !deviceId.value) return
  storyDetailVisible.value = true
  storyDetailLoading.value = true
  storyDetail.value = null
  try {
    const response = await api.get(`/admin/devices/${deviceId.value}/stories/${storyId}`)
    storyDetail.value = response.data?.data || response.data || null
  } catch (error) {
    ElMessage.error(error.response?.data?.error || '加载故事详情失败')
    storyDetailVisible.value = false
  } finally {
    storyDetailLoading.value = false
  }
}

const buildParams = (extra = {}) => {
  const params = { limit: 20, ...extra }
  if (selectedDate.value && !extra.before_sort_time && !extra.after_sort_time) {
    params.date = selectedDate.value
  }
  return params
}

const fetchRecords = async (extra = {}) => {
  const response = await api.get(`/admin/devices/${deviceId.value}/conversation-records`, {
    params: buildParams(extra)
  })
  return response.data || {}
}

const loadInitial = async () => {
  loading.value = true
  records.value = []
  try {
    const res = await fetchRecords()
    records.value = res.data || []
    hasOlder.value = !!res.has_older
    hasNewer.value = !!res.has_newer
    deviceSN.value = res.device_name || deviceSN.value
    await nextTick()
    timelineRef.value?.scrollToBottom()
  } catch (error) {
    ElMessage.error(error.response?.data?.error || '加载对话记录失败')
  } finally {
    loading.value = false
  }
}

const loadOlder = async () => {
  if (!hasOlder.value || loadingOlder.value || loading.value || !records.value.length) return
  loadingOlder.value = true
  const first = records.value[0]
  try {
    const res = await fetchRecords({
      before_sort_time: first.sort_time,
      before_type: first.type,
      before_id: first.id
    })
    const older = res.data || []
    if (!older.length) {
      hasOlder.value = false
      return
    }
    records.value = older.concat(records.value)
    hasOlder.value = !!res.has_older
  } catch (error) {
    ElMessage.error(error.response?.data?.error || '加载失败')
  } finally {
    loadingOlder.value = false
  }
}

const loadNewer = async () => {
  if (!hasNewer.value || loadingNewer.value || loading.value || !records.value.length) return
  loadingNewer.value = true
  const last = records.value[records.value.length - 1]
  try {
    const res = await fetchRecords({
      after_sort_time: last.sort_time,
      after_type: last.type,
      after_id: last.id
    })
    const newer = res.data || []
    if (!newer.length) {
      hasNewer.value = false
      return
    }
    records.value = records.value.concat(newer)
    hasNewer.value = !!res.has_newer
  } catch (error) {
    ElMessage.error(error.response?.data?.error || '加载失败')
  } finally {
    loadingNewer.value = false
  }
}

const handleDateChange = () => {
  loadInitial()
}

const getAudioUrl = async (record) => {
  const key = recordKey(record)
  if (audioBlobUrls.value[key]) {
    return audioBlobUrls.value[key]
  }
  let path
  if (record.type === 'chat') {
    path = `/admin/conversation-records/chat/${record.id}/audio`
  } else if (record.chat_audio_id) {
    path = `/admin/conversation-records/chat/${record.chat_audio_id}/audio`
  } else {
    path = `/admin/conversation-records/parent/${record.id}/audio`
  }
  const response = await api.get(path, { responseType: 'blob' })
  const blobUrl = URL.createObjectURL(response.data)
  audioBlobUrls.value[key] = blobUrl
  return blobUrl
}

const stopAudio = () => {
  if (audioEl.value) {
    audioEl.value.pause()
    audioEl.value = null
  }
  playingKey.value = ''
}

const handlePlayAudio = async (record) => {
  const key = recordKey(record)
  if (playingKey.value === key) {
    stopAudio()
    return
  }
  stopAudio()
  try {
    const url = await getAudioUrl(record)
    const audio = new Audio(url)
    audioEl.value = audio
    audio.onended = () => {
      playingKey.value = ''
      audioEl.value = null
    }
    audio.onerror = () => {
      playingKey.value = ''
      audioEl.value = null
      ElMessage.warning('音频播放失败')
    }
    await audio.play()
    playingKey.value = key
  } catch (error) {
    ElMessage.warning('音频加载失败')
  }
}

const loadDeviceMeta = async () => {
  try {
    const response = await api.get(`/admin/devices/${deviceId.value}`)
    const device = response.data?.data
    if (device) {
      deviceLabel.value = formatDeviceNickName(device)
      deviceSN.value = device.device_name || ''
    }
  } catch (error) {
    // 列表接口失败时仍可通过 conversation-records 返回 device_name
  }
}

onMounted(async () => {
  await loadDeviceMeta()
  await loadInitial()
})

onBeforeUnmount(() => {
  stopAudio()
  Object.values(audioBlobUrls.value).forEach((url) => URL.revokeObjectURL(url))
})
</script>

<style scoped>
.device-conversation-page {
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

.story-meta {
  margin: 0 0 12px;
  color: var(--apple-text-secondary);
  font-size: 13px;
}
</style>
