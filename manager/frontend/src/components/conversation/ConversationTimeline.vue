<template>
  <div class="conversation-timeline" v-loading="loading && records.length === 0">
    <div
      ref="scrollContainerRef"
      class="chat-messages"
      @scroll="handleScroll"
    >
      <div v-if="loadingOlder" class="edge-hint">加载更早记录...</div>
      <div v-else-if="hasOlder && records.length" class="edge-hint subtle">继续下拉加载更早</div>

      <template v-if="records.length">
        <div
          v-for="(record, index) in records"
          :key="recordKey(record)"
          class="message-wrapper"
          :class="wrapperClass(record)"
        >
          <div v-if="shouldShowTime(record, index)" class="message-time-divider">
            {{ formatTimeShort(record.sort_time) }}
          </div>

          <div
            v-if="record.type === 'parent_message'"
            class="message-row message-row-center"
          >
            <div class="parent-bubble">
              <span class="parent-label">家长留言</span>
              <div v-if="record.title" class="parent-title">{{ record.title }}</div>
              <div class="message-text">{{ record.content }}</div>
            </div>
            <el-button
              v-if="record.has_audio"
              class="play-btn-outside"
              :icon="playingKey === recordKey(record) ? VideoPause : VideoPlay"
              circle
              size="small"
              :title="playingKey === recordKey(record) ? '播放中' : '播放留言'"
              @click="emitPlay(record)"
            />
          </div>

          <div
            v-else
            class="message-row"
            :class="record.role === 'assistant' ? 'message-row-left' : 'message-row-right'"
          >
            <div
              class="message-bubble"
              :class="record.role === 'assistant' ? 'message-bubble-left' : 'message-bubble-right'"
            >
              <div v-if="record.content" class="message-text">{{ record.content }}</div>
            </div>
            <el-button
              v-if="record.has_audio"
              class="play-btn-outside"
              :icon="playingKey === recordKey(record) ? VideoPause : VideoPlay"
              circle
              size="small"
              :title="playingKey === recordKey(record) ? '播放中' : '播放'"
              @click="emitPlay(record)"
            />
          </div>
        </div>
      </template>

      <div v-if="loadingNewer" class="edge-hint">加载更多...</div>
      <div v-else-if="hasNewer && records.length" class="edge-hint subtle">继续上滑加载更新</div>

      <div v-if="!loading && records.length === 0" class="empty-state">
        <el-empty description="暂无对话记录" />
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { VideoPlay, VideoPause } from '@element-plus/icons-vue'

const props = defineProps({
  records: { type: Array, default: () => [] },
  loading: { type: Boolean, default: false },
  loadingOlder: { type: Boolean, default: false },
  loadingNewer: { type: Boolean, default: false },
  hasOlder: { type: Boolean, default: false },
  hasNewer: { type: Boolean, default: false },
  playingKey: { type: String, default: '' }
})

const emit = defineEmits(['load-older', 'load-newer', 'play-audio'])

const scrollContainerRef = ref(null)

const recordKey = (record) => `${record.type}_${record.id}`

const wrapperClass = (record) => {
  if (record.type === 'parent_message') return 'message-center'
  return record.role === 'assistant' ? 'message-left' : 'message-right'
}

const formatTimeShort = (dateString) => {
  const date = new Date(dateString)
  const now = new Date()
  const today = new Date(now.getFullYear(), now.getMonth(), now.getDate())
  const msgDate = new Date(date.getFullYear(), date.getMonth(), date.getDate())

  if (msgDate.getTime() === today.getTime()) {
    return date.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' })
  }

  const yesterday = new Date(today)
  yesterday.setDate(yesterday.getDate() - 1)
  if (msgDate.getTime() === yesterday.getTime()) {
    return `昨天 ${date.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' })}`
  }

  if (date.getFullYear() === now.getFullYear()) {
    return `${date.getMonth() + 1}月${date.getDate()}日 ${date.toLocaleTimeString('zh-CN', {
      hour: '2-digit',
      minute: '2-digit'
    })}`
  }

  return date.toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit'
  })
}

const shouldShowTime = (record, index) => {
  if (index === 0) return true
  const currentTime = new Date(record.sort_time).getTime()
  const prevTime = new Date(props.records[index - 1].sort_time).getTime()
  return currentTime - prevTime > 5 * 60 * 1000
}

const emitPlay = (record) => {
  emit('play-audio', record)
}

const handleScroll = (event) => {
  const el = event.target
  if (!el) return
  if (el.scrollTop <= 40 && props.hasOlder && !props.loadingOlder && !props.loading) {
    emit('load-older')
  }
  if (el.scrollHeight - el.scrollTop - el.clientHeight <= 40 && props.hasNewer && !props.loadingNewer && !props.loading) {
    emit('load-newer')
  }
}

const scrollToBottom = () => {
  const el = scrollContainerRef.value
  if (el) {
    el.scrollTop = el.scrollHeight
  }
}

defineExpose({ scrollToBottom, scrollContainerRef })
</script>

<style scoped>
.conversation-timeline {
  min-height: 400px;
}

.chat-messages {
  padding: 20px;
  max-height: 70vh;
  overflow-y: auto;
  background: rgba(248, 250, 252, 0.92);
  border: 1px solid rgba(229, 229, 234, 0.72);
  border-radius: 22px;
}

.edge-hint {
  text-align: center;
  font-size: 12px;
  color: var(--apple-text-secondary);
  padding: 8px 0 12px;
}

.edge-hint.subtle {
  color: var(--apple-text-tertiary);
}

.message-wrapper {
  display: flex;
  flex-direction: column;
  margin-bottom: 16px;
}

.message-center {
  align-items: center;
}

.message-left {
  align-items: flex-start;
}

.message-right {
  align-items: flex-end;
}

.message-time-divider {
  text-align: center;
  margin: 16px 0;
  font-size: 12px;
  color: var(--apple-text-tertiary);
}

.message-row {
  display: flex;
  align-items: center;
  gap: 10px;
  max-width: 78%;
}

.message-row-left {
  align-self: flex-start;
}

.message-row-right {
  align-self: flex-end;
  flex-direction: row-reverse;
}

.message-row-center {
  align-self: center;
}

.message-bubble {
  padding: 10px 14px;
  border-radius: 18px;
  word-break: break-word;
  box-shadow: 0 8px 16px rgba(15, 23, 42, 0.05);
  max-width: 100%;
  flex: 1;
  min-width: 0;
}

.message-bubble-left {
  background: rgba(255, 255, 255, 0.94);
  border-top-left-radius: 8px;
}

.message-bubble-right {
  background: rgba(0, 122, 255, 0.12);
  border: 1px solid rgba(0, 122, 255, 0.16);
  border-top-right-radius: 8px;
}

.parent-bubble {
  padding: 12px 16px;
  border-radius: 16px;
  background: #fff8e1;
  border: 1px solid #ffe082;
  text-align: center;
  flex: 1;
  min-width: 0;
}

.parent-label {
  display: block;
  font-size: 12px;
  color: #f57c00;
  margin-bottom: 6px;
}

.parent-title {
  font-weight: 600;
  margin-bottom: 6px;
}

.message-text {
  line-height: 1.5;
  white-space: pre-wrap;
  font-size: 14px;
}

.play-btn-outside {
  flex-shrink: 0;
}

.empty-state {
  padding: 48px 0;
}
</style>
