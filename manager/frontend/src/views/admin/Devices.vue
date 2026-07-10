<template>
  <div class="admin-devices">
    <div class="toolbar">
      <el-select v-model="bindFilter" placeholder="绑定状态" style="width: 140px">
        <el-option label="全部" value="all" />
        <el-option label="已绑定" value="bound" />
        <el-option label="未绑定" value="unbound" />
        <el-option label="已激活" value="activated" />
      </el-select>
      <el-button type="primary" @click="openAddDialog">
        <el-icon><Plus /></el-icon>
        添加设备
      </el-button>
      <el-button @click="loadDevices">
        <el-icon><Refresh /></el-icon>
        刷新
      </el-button>
    </div>

    <el-table :data="filteredDevices" v-loading="loading" stripe>
      <el-table-column prop="id" label="ID" width="80" />
      <el-table-column label="设备昵称" min-width="170">
        <template #default="{ row }">
          <span :class="['device-nick-name', { 'is-empty': !getDeviceNickName(row) }]">
            {{ formatDeviceNickName(row) }}
          </span>
        </template>
      </el-table-column>
      <el-table-column label="设备ID" width="190">
        <template #default="{ row }">
          <span class="device-id-text">{{ row.device_name || '-' }}</span>
        </template>
      </el-table-column>
      <el-table-column label="绑定用户" min-width="160">
        <template #default="{ row }">
          <template v-if="row.user_id > 0">
            <span class="bind-user">{{ row.username || `用户 #${row.user_id}` }}</span>
            <div class="bind-meta">ID: {{ row.user_id }}</div>
          </template>
          <el-tag v-else type="info" size="small">未绑定</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="关联智能体" width="150">
        <template #default="{ row }">
          <span v-if="row.agent_id > 0">
            {{ row.agent_name || `智能体 ${row.agent_id}` }}
          </span>
          <el-tag v-else type="info" size="small">未分配</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="激活状态" width="100">
        <template #default="{ row }">
          <el-tag :type="row.activated ? 'success' : 'warning'">
            {{ row.activated ? '已激活' : '未激活' }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column label="在线状态" width="100">
        <template #default="{ row }">
          <el-tag :type="isDeviceOnline(row.last_active_at) ? 'success' : 'danger'">
            {{ isDeviceOnline(row.last_active_at) ? '在线' : '离线' }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="last_active_at" label="最后活跃时间" width="180">
        <template #default="{ row }">
          {{ row.last_active_at ? new Date(row.last_active_at).toLocaleString() : '从未活跃' }}
        </template>
      </el-table-column>
      <el-table-column prop="created_at" label="创建时间" width="180">
        <template #default="{ row }">
          {{ new Date(row.created_at).toLocaleString() }}
        </template>
      </el-table-column>
      <el-table-column label="操作" width="320" fixed="right">
        <template #default="{ row }">
          <el-button size="small" @click="editDevice(row)">
            编辑
          </el-button>
          <el-button
            size="small"
            type="warning"
            :disabled="!row.user_id"
            @click="factoryResetDevice(row)"
          >
            出厂重置
          </el-button>
          <el-dropdown trigger="click" @command="(cmd) => handleMoreAction(cmd, row)">
            <el-button size="small">
              更多
              <el-icon class="el-icon--right"><ArrowDown /></el-icon>
            </el-button>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item command="conversation" :disabled="!row.agent_id || !row.device_name">
                  对话记录
                </el-dropdown-item>
                <el-dropdown-item command="memory" :disabled="!row.agent_id || !row.device_name">
                  设备记忆
                </el-dropdown-item>
                <el-dropdown-item command="stories" :disabled="!row.device_name">
                  故事
                </el-dropdown-item>
                <el-dropdown-item command="mcp">
                  MCP
                </el-dropdown-item>
                <el-dropdown-item command="delete" divided>
                  <span class="danger-text">删除</span>
                </el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </template>
      </el-table-column>
    </el-table>

    <!-- 添加/编辑设备对话框 -->

    <el-dialog v-model="showMcpDialog" title="设备MCP工具" width="760px">
      <div v-loading="mcpLoading">
        <div class="mcp-tools-header">
          <el-button size="small" type="primary" @click="refreshDeviceMcpTools" :loading="toolsLoading">刷新工具列表</el-button>
        </div>

        <div v-if="mcpTools.length === 0" class="tools-empty">暂无工具数据</div>
        <div v-else class="tools-tags">
          <el-tag v-for="tool in mcpTools" :key="tool.name" class="tool-tag">{{ tool.name }}</el-tag>
        </div>

        <el-divider />

        <el-form :model="mcpCallForm" label-width="90px">
          <el-form-item label="工具">
            <el-select v-model="mcpCallForm.tool_name" placeholder="请选择工具" style="width:100%" @change="handleMcpToolChange">
              <el-option v-for="tool in mcpTools" :key="tool.name" :label="tool.name" :value="tool.name" />
            </el-select>
          </el-form-item>
          <el-form-item label="参数JSON">
            <el-input v-model="mcpCallForm.argumentsText" type="textarea" :rows="6" placeholder='例如: {"query":"hello"}' />
          </el-form-item>
        </el-form>

        <el-button type="primary" @click="callDeviceMcpTool" :loading="callingTool">调用工具</el-button>

        <el-divider />
        <div class="endpoint-content">{{ mcpCallResult || '暂无调用结果' }}</div>
      </div>
    </el-dialog>

    <el-dialog
      v-model="showAddDialog"
      :title="editingDevice ? '编辑设备' : '添加设备'"
      width="500px"
    >
      <DeviceForm
        ref="deviceFormRef"
        v-model="deviceForm"
        is-admin
        :mode="editingDevice ? 'edit' : 'create'"
      />
      <template #footer>
        <el-button @click="showAddDialog = false">取消</el-button>
        <el-button type="primary" @click="saveDevice" :loading="saving">
          {{ editingDevice ? '更新' : '添加' }}
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus, Refresh, ArrowDown } from '@element-plus/icons-vue'
import api from '../../utils/api'
import DeviceForm from '../../components/common/DeviceForm.vue'
import { createDefaultDeviceForm, deviceToForm } from '../../composables/useAgentFormOptions'
import { formatDeviceNickName, getDeviceNickName, getDeviceReferenceLabel } from '../../utils/iotDevice'

const router = useRouter()

const devices = ref([])
const bindFilter = ref('all')
const loading = ref(false)
const showAddDialog = ref(false)
const editingDevice = ref(null)
const saving = ref(false)
const deviceFormRef = ref()

const showMcpDialog = ref(false)
const mcpLoading = ref(false)
const toolsLoading = ref(false)
const callingTool = ref(false)
const currentDeviceId = ref(null)
const mcpTools = ref([])
const mcpCallResult = ref('')
const mcpCallForm = ref({ tool_name: '', argumentsText: '{}' })
const deviceForm = ref(createDefaultDeviceForm({ isAdmin: true }))

const filteredDevices = computed(() => {
  const list = devices.value || []
  switch (bindFilter.value) {
    case 'bound':
      return list.filter((item) => item.user_id > 0)
    case 'unbound':
      return list.filter((item) => !item.user_id)
    case 'activated':
      return list.filter((item) => item.activated)
    default:
      return list
  }
})

const loadDevices = async () => {
  loading.value = true
  try {
    const response = await api.get('/admin/devices')
    devices.value = response.data.data || []
  } catch (error) {
    ElMessage.error('加载设备列表失败')
    console.error('Error loading devices:', error)
  } finally {
    loading.value = false
  }
}

const openAddDialog = () => {
  editingDevice.value = null
  deviceForm.value = createDefaultDeviceForm({ isAdmin: true })
  showAddDialog.value = true
}

const openConversationRecords = (device) => {
  if (!device.agent_id || !device.device_name) {
    ElMessage.warning('设备须绑定智能体且具备 SN 方可查看对话记录')
    return
  }
  router.push({ name: 'AdminDeviceConversationRecords', params: { id: device.id } })
}

const openDeviceMemory = (device) => {
  if (!device.agent_id || !device.device_name) {
    ElMessage.warning('设备须绑定智能体且具备 SN 方可查看设备记忆')
    return
  }
  router.push({ name: 'AdminDeviceMemory', params: { id: device.id } })
}

const openDeviceStories = (device) => {
  if (!device.device_name) {
    ElMessage.warning('设备须具备 SN 方可查看故事记录')
    return
  }
  router.push({ name: 'AdminDeviceStories', params: { id: device.id } })
}

const editDevice = (device) => {
  editingDevice.value = device
  deviceForm.value = deviceToForm(device, { isAdmin: true })
  showAddDialog.value = true
}

const saveDevice = async () => {
  if (!deviceFormRef.value) return
  
  const valid = await deviceFormRef.value.validate().catch(() => false)
  if (!valid) return

  saving.value = true
  try {
    const payload = deviceFormRef.value.buildPayload()
    if (editingDevice.value) {
      await api.put(`/admin/devices/${editingDevice.value.id}`, payload, { silentError: true })
      ElMessage.success('设备更新成功')
    } else {
      const response = await api.post('/admin/devices', payload, { silentError: true })
      // 根据后端返回的消息显示不同的提示
      const message = response.data.message || '设备添加成功'
      ElMessage.success(message)
    }
    showAddDialog.value = false
    resetForm()
    loadDevices()
  } catch (error) {
    const errorMessage = error.response?.data?.error || (editingDevice.value ? '设备更新失败' : '设备添加失败')
    ElMessage.error(errorMessage)
    console.error('Error saving device:', error)
  } finally {
    saving.value = false
  }
}

const handleMoreAction = (command, device) => {
  switch (command) {
    case 'conversation':
      openConversationRecords(device)
      break
    case 'memory':
      openDeviceMemory(device)
      break
    case 'stories':
      openDeviceStories(device)
      break
    case 'mcp':
      showDeviceMcp(device)
      break
    case 'delete':
      deleteDevice(device)
      break
    default:
      break
  }
}

const factoryResetDevice = async (device) => {
  try {
    await ElMessageBox.confirm(
      '将永久删除该设备的 Memobase 记忆、Redis 缓存、对话记录、家长留言与音频，并解除用户绑定。设备出厂登记信息保留。此操作不可恢复。',
      '确认出厂重置',
      {
        confirmButtonText: '出厂重置',
        cancelButtonText: '取消',
        type: 'warning',
        confirmButtonClass: 'el-button--danger'
      }
    )

    await api.post(`/admin/devices/${device.id}/factory-reset`)
    ElMessage.success('设备已出厂重置')
    loadDevices()
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error(error.response?.data?.error || '出厂重置失败')
      console.error('Error factory resetting device:', error)
    }
  }
}

const deleteDevice = async (device) => {
  try {
    await ElMessageBox.confirm(
      `将先执行出厂重置清理全部业务数据，再删除设备「${getDeviceReferenceLabel(device)}」的登记记录。不可恢复。`,
      '确认删除',
      {
        confirmButtonText: '删除',
        cancelButtonText: '取消',
        type: 'warning',
        confirmButtonClass: 'el-button--danger'
      }
    )
    
    await api.delete(`/admin/devices/${device.id}`)
    ElMessage.success('设备删除成功')
    loadDevices()
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error(error.response?.data?.error || '设备删除失败')
      console.error('Error deleting device:', error)
    }
  }
}



const showDeviceMcp = async (device) => {
  currentDeviceId.value = device.id
  showMcpDialog.value = true
  mcpLoading.value = true
  mcpCallResult.value = ''
  mcpCallForm.value = { tool_name: '', argumentsText: '{}' }
  try {
    await refreshDeviceMcpTools()
  } finally {
    mcpLoading.value = false
  }
}

const refreshDeviceMcpTools = async () => {
  if (!currentDeviceId.value) return
  toolsLoading.value = true
  try {
    const response = await api.get(`/admin/devices/${currentDeviceId.value}/mcp-tools`)
    mcpTools.value = response.data.data?.tools || []
    if (!mcpCallForm.value.tool_name && mcpTools.value.length > 0) {
      mcpCallForm.value.tool_name = mcpTools.value[0].name
    }
  } catch (error) {
    ElMessage.error('获取设备MCP工具失败')
    mcpTools.value = []
  } finally {
    toolsLoading.value = false
  }
}



const buildExampleFromSchema = (schema = {}) => {
  if (!schema || typeof schema !== 'object') return {}
  if (Array.isArray(schema.enum) && schema.enum.length > 0) return schema.enum[0]

  const type = schema.type || 'object'
  if (type === 'object') {
    const props = schema.properties || {}
    const result = {}
    Object.keys(props).sort().forEach((key) => {
      result[key] = buildExampleFromSchema(props[key])
    })
    return result
  }
  if (type === 'array') {
    return [buildExampleFromSchema(schema.items || {})]
  }
  if (type === 'number') return 0.1
  if (type === 'integer') return 0
  if (type === 'boolean') return false
  return ''
}

const updateMcpExampleByTool = (toolName) => {
  const selectedTool = mcpTools.value.find(item => item.name === toolName)
  if (!selectedTool) return

  const example = buildExampleFromSchema(selectedTool.input_schema || {})
  mcpCallForm.value.argumentsText = JSON.stringify(example ?? {}, null, 2)
}

const handleMcpToolChange = (toolName) => {
  updateMcpExampleByTool(toolName)
}

const formatMcpCallResult = (payload) => {
  const MAX_PARSE_DEPTH = 8

  const tryParseJSONString = (value) => {
    if (typeof value !== 'string') return { parsed: false, value }
    let text = value.trim()
    if (!text) return { parsed: false, value }

    const fenced = text.match(/^```(?:json)?\s*([\s\S]*?)\s*```$/i)
    if (fenced) {
      text = fenced[1].trim()
    }

    const looksLikeJSON =
      (text.startsWith('{') && text.endsWith('}')) ||
      (text.startsWith('[') && text.endsWith(']'))
    if (!looksLikeJSON) return { parsed: false, value }

    try {
      return { parsed: true, value: JSON.parse(text) }
    } catch (_) {
      return { parsed: false, value }
    }
  }

  const deepParseJSONStrings = (value, depth = 0) => {
    if (depth >= MAX_PARSE_DEPTH || value == null) return value

    if (typeof value === 'string') {
      const parsed = tryParseJSONString(value)
      if (!parsed.parsed) return value
      return deepParseJSONStrings(parsed.value, depth + 1)
    }

    if (Array.isArray(value)) {
      return value.map((item) => deepParseJSONStrings(item, depth + 1))
    }

    if (typeof value === 'object') {
      const out = {}
      Object.keys(value).forEach((key) => {
        out[key] = deepParseJSONStrings(value[key], depth + 1)
      })

      if (Array.isArray(out.content) && out.content.length === 1) {
        const first = out.content[0]
        if (first && typeof first === 'object' && !Array.isArray(first) && first.type === 'text' && Object.prototype.hasOwnProperty.call(first, 'text')) {
          const textValue = first.text
          if (textValue && typeof textValue === 'object') {
            return textValue
          }
        }
      }

      return out
    }

    return value
  }

  const data = payload ?? {}
  const raw = (data && typeof data === 'object' && !Array.isArray(data) && Object.prototype.hasOwnProperty.call(data, 'result'))
    ? data.result
    : data

  return JSON.stringify(deepParseJSONStrings(raw), null, 2)
}

const callDeviceMcpTool = async () => {
  if (!currentDeviceId.value || !mcpCallForm.value.tool_name) {
    ElMessage.warning('请选择工具')
    return
  }

  let argumentsObj = {}
  try {
    argumentsObj = mcpCallForm.value.argumentsText ? JSON.parse(mcpCallForm.value.argumentsText) : {}
  } catch (e) {
    ElMessage.error('参数JSON格式错误')
    return
  }

  callingTool.value = true
  try {
    const response = await api.post(`/admin/devices/${currentDeviceId.value}/mcp-call`, {
      tool_name: mcpCallForm.value.tool_name,
      arguments: argumentsObj
    })
    mcpCallResult.value = formatMcpCallResult(response.data.data || {})
    ElMessage.success('MCP工具调用成功')
  } catch (error) {
    mcpCallResult.value = JSON.stringify(error.response?.data || { error: error.message }, null, 2)
    ElMessage.error('MCP工具调用失败')
  } finally {
    callingTool.value = false
  }
}

const resetForm = () => {
  editingDevice.value = null
  deviceForm.value = createDefaultDeviceForm({ isAdmin: true })
  if (deviceFormRef.value) {
    deviceFormRef.value.resetFields()
  }
}

// 判断设备是否在线（基于最后活跃时间）
const isDeviceOnline = (lastActiveAt) => {
  if (!lastActiveAt) return false
  const now = new Date()
  const lastActive = new Date(lastActiveAt)
  // 5分钟内有活动认为在线
  return (now - lastActive) < 5 * 60 * 1000
}

onMounted(() => {
  loadDevices()
})
</script>

<style scoped>
.admin-devices {
  padding: 20px;
}

.toolbar {
  margin-bottom: 20px;
  display: flex;
  gap: 12px;
  justify-content: flex-end;
  flex-wrap: wrap;
}

.device-nick-name {
  font-weight: 700;
  color: var(--apple-text, #1d1d1f);
}

.device-nick-name.is-empty {
  font-weight: 500;
  color: #909399;
}

.device-id-text {
  color: rgba(107, 114, 128, 0.72);
  font-family: monospace;
  font-size: 12px;
}

.bind-user {
  font-weight: 600;
  color: var(--apple-text, #1d1d1f);
}

.bind-meta {
  margin-top: 2px;
  font-size: 12px;
  color: #909399;
}

.danger-text {
  color: var(--el-color-danger);
}

.form-tip {
  margin-top: 6px;
  color: rgba(107, 114, 128, 0.78);
  font-size: 12px;
  line-height: 1.5;
}

.tools-tags { display:flex; flex-wrap:wrap; gap:8px; margin-bottom:12px; }
.tools-empty { color:#909399; margin: 8px 0 16px; }
.endpoint-content {
  white-space: pre-wrap;
  font-family: monospace;
  background: #f8fafc;
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  padding: 10px;
  min-height: 80px;
}
.mcp-tools-header { margin-bottom: 12px; }
</style>
