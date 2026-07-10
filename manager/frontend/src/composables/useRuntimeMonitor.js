import { onMounted, onUnmounted, ref } from 'vue'
import api from '@/utils/api'

export function useRuntimeMonitor(options = {}) {
  const autoConnect = options.autoConnect !== false
  const nodes = ref([])
  const summary = ref({
    total_nodes: 0,
    online_nodes: 0,
    total_chat_managers: 0,
    total_active_sessions: 0,
    devices_active_5m: 0
  })
  const connectionState = ref('idle')
  const selectedNodeId = ref('')
  const lastUpdatedAt = ref('')

  let eventSource = null
  let reconnectTimer = null

  const selectedNode = ref(null)

  const applyPayload = (payload) => {
    nodes.value = payload?.nodes || []
    summary.value = payload?.summary || summary.value
    lastUpdatedAt.value = payload?.ts || new Date().toISOString()
    if (selectedNodeId.value) {
      selectedNode.value = nodes.value.find((item) => item.node_id === selectedNodeId.value) || null
    }
  }

  const loadSnapshot = async () => {
    const [nodesResp, summaryResp] = await Promise.all([
      api.get('/admin/runtime/nodes'),
      api.get('/admin/runtime/summary')
    ])
    applyPayload({
      nodes: nodesResp.data?.data || [],
      summary: summaryResp.data?.data || summary.value,
      ts: new Date().toISOString()
    })
  }

  const disconnect = () => {
    if (reconnectTimer) {
      clearTimeout(reconnectTimer)
      reconnectTimer = null
    }
    if (eventSource) {
      eventSource.close()
      eventSource = null
    }
    connectionState.value = 'disconnected'
  }

  const connect = async () => {
    disconnect()
    connectionState.value = 'connecting'
    try {
      const tokenResp = await api.get('/admin/runtime/stream-token')
      const token = tokenResp.data?.data?.token
      if (!token) {
        throw new Error('未获取到 stream token')
      }
      eventSource = new EventSource(`/api/admin/runtime/stream?token=${encodeURIComponent(token)}`)
      eventSource.addEventListener('runtime', (event) => {
        try {
          const payload = JSON.parse(event.data)
          applyPayload(payload)
          connectionState.value = 'connected'
        } catch (error) {
          console.error('解析运行时 SSE 数据失败:', error)
        }
      })
      eventSource.onerror = () => {
        connectionState.value = 'reconnecting'
        disconnect()
        reconnectTimer = setTimeout(() => {
          connect().catch((error) => {
            console.error('运行时 SSE 重连失败:', error)
            connectionState.value = 'error'
          })
        }, 3000)
      }
    } catch (error) {
      console.error('建立运行时 SSE 失败，回退轮询:', error)
      connectionState.value = 'polling'
      await loadSnapshot()
    }
  }

  const selectNode = (nodeId) => {
    selectedNodeId.value = nodeId
    selectedNode.value = nodes.value.find((item) => item.node_id === nodeId) || null
  }

  if (autoConnect) {
    onMounted(() => {
      connect().catch(async (error) => {
        console.error('初始化运行时监控失败:', error)
        connectionState.value = 'polling'
        await loadSnapshot()
      })
    })
    onUnmounted(() => {
      disconnect()
    })
  }

  return {
    nodes,
    summary,
    selectedNode,
    selectedNodeId,
    connectionState,
    lastUpdatedAt,
    connect,
    disconnect,
    loadSnapshot,
    selectNode
  }
}
