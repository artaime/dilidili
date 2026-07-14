<template>
  <div class="config-page story-assets-page">
    <div class="page-actions">
      <el-input
        v-model="filters.q"
        placeholder="搜索标题 / 主题 / 规范名 / ID"
        style="width: 260px"
        clearable
        @keyup.enter="reload"
      />
      <el-select v-model="filters.pool_kind" placeholder="共享池" clearable style="width: 140px" @change="reload">
        <el-option label="点名池 named" value="named" />
        <el-option label="开放池 open" value="open" />
        <el-option label="睡前 bedtime" value="bedtime" />
      </el-select>
      <el-select v-model="filters.shareable" placeholder="是否共享" clearable style="width: 130px" @change="reload">
        <el-option label="可共享" value="true" />
        <el-option label="不可共享" value="false" />
      </el-select>
      <el-button @click="reload" :loading="loading">刷新</el-button>
      <el-button type="success" @click="openAiCreate">AI 新增</el-button>
      <el-button type="primary" @click="openCreate">
        <el-icon><Plus /></el-icon>
        手动新增
      </el-button>
    </div>

    <el-table :data="items" v-loading="loading" stripe style="width: 100%">
      <el-table-column prop="title" label="标题" min-width="160" show-overflow-tooltip />
      <el-table-column label="共享池" width="110">
        <template #default="{ row }">
          <el-tag size="small" :type="poolTagType(row.pool_kind)">{{ poolLabel(row.pool_kind) }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="canonical_key" label="规范名" min-width="120" show-overflow-tooltip />
      <el-table-column prop="theme_key" label="主题" min-width="110" show-overflow-tooltip />
      <el-table-column label="共享" width="80" align="center">
        <template #default="{ row }">
          <el-tag size="small" :type="row.shareable ? 'success' : 'info'">
            {{ row.shareable ? '是' : '否' }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="reuse_count" label="复用" width="70" align="center" />
      <el-table-column label="字数" width="80" align="center">
        <template #default="{ row }">{{ row.text_length || 0 }}</template>
      </el-table-column>
      <el-table-column label="更新时间" width="170">
        <template #default="{ row }">{{ formatTime(row.updated_at) }}</template>
      </el-table-column>
      <el-table-column label="操作" width="200" fixed="right">
        <template #default="{ row }">
          <el-button size="small" @click="openEdit(row)">编辑</el-button>
          <el-button size="small" type="danger" @click="handleDelete(row)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>

    <div class="pager">
      <el-pagination
        background
        layout="total, prev, pager, next"
        :total="total"
        :page-size="pageSize"
        :current-page="page"
        @current-change="onPageChange"
      />
    </div>

    <!-- 编辑 / 手动新增 -->
    <el-dialog
      v-model="dialogVisible"
      :title="isEdit ? '编辑故事资产' : '新增故事资产'"
      width="720px"
      destroy-on-close
      @closed="resetForm"
    >
      <el-form ref="formRef" :model="form" :rules="formRules" label-width="100px">
        <el-form-item v-if="isEdit" label="故事 ID">
          <el-input v-model="form.story_id" disabled />
        </el-form-item>
        <el-form-item label="标题" prop="title">
          <el-input v-model="form.title" placeholder="故事标题" />
        </el-form-item>
        <el-form-item label="主题" prop="theme_key">
          <el-input v-model="form.theme_key" placeholder="如：后羿射日" />
        </el-form-item>
        <el-form-item label="规范名">
          <el-input v-model="form.canonical_key" placeholder="点名池建议填写通行名" />
        </el-form-item>
        <el-form-item label="别名">
          <el-input
            v-model="aliasesText"
            placeholder="逗号分隔，如：后裔射太阳,后羿射太阳"
          />
        </el-form-item>
        <el-row :gutter="12">
          <el-col :span="8">
            <el-form-item label="共享池" prop="pool_kind">
              <el-select v-model="form.pool_kind" style="width: 100%">
                <el-option label="点名 named" value="named" />
                <el-option label="开放 open" value="open" />
                <el-option label="睡前 bedtime" value="bedtime" />
                <el-option label="不入池" value="" />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="讲述">
              <el-select v-model="form.narration_mode" style="width: 100%">
                <el-option label="正篇 canonical" value="canonical" />
                <el-option label="原创 creative" value="creative" />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="类型">
              <el-select v-model="form.mode" style="width: 100%" clearable>
                <el-option label="神话 myth" value="myth" />
                <el-option label="经典 classic" value="classic" />
                <el-option label="寓言 fable" value="fable" />
                <el-option label="童话 fairy_tale" value="fairy_tale" />
                <el-option label="睡前 bedtime" value="bedtime" />
                <el-option label="原创 original" value="original" />
              </el-select>
            </el-form-item>
          </el-col>
        </el-row>
        <el-form-item label="年龄档">
          <el-select v-model="form.age_band" clearable style="width: 100%">
            <el-option label="学前 preschool" value="preschool" />
            <el-option label="小学低 primary_low" value="primary_low" />
            <el-option label="小学高 primary_high" value="primary_high" />
            <el-option label="初中 junior_high" value="junior_high" />
          </el-select>
        </el-form-item>
        <el-form-item label="正文" prop="full_text">
          <div class="text-toolbar">
            <el-button size="small" type="success" :loading="generating" @click="generateIntoForm">
              AI 生成正文
            </el-button>
          </div>
          <el-input
            v-model="form.full_text"
            type="textarea"
            :rows="12"
            placeholder="故事全文"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="submitting" @click="submitForm">保存</el-button>
      </template>
    </el-dialog>

    <!-- AI 新增向导 -->
    <el-dialog v-model="aiDialogVisible" title="AI 新增故事" width="520px" destroy-on-close>
      <el-form :model="aiForm" label-width="100px">
        <el-form-item label="主题" required>
          <el-input v-model="aiForm.theme" placeholder="如：后羿射日 / 讲个森林冒险" />
        </el-form-item>
        <el-form-item label="共享池">
          <el-select v-model="aiForm.pool_kind" style="width: 100%">
            <el-option label="点名 named" value="named" />
            <el-option label="开放 open" value="open" />
            <el-option label="睡前 bedtime" value="bedtime" />
          </el-select>
        </el-form-item>
        <el-form-item label="讲述">
          <el-select v-model="aiForm.narration_mode" style="width: 100%">
            <el-option label="正篇 canonical" value="canonical" />
            <el-option label="原创 creative" value="creative" />
          </el-select>
        </el-form-item>
        <el-form-item label="LLM 配置">
          <el-select v-model="aiForm.llm_config_id" clearable placeholder="默认 LLM" style="width: 100%">
            <el-option
              v-for="cfg in llmConfigs"
              :key="cfg.config_id"
              :label="cfg.name || cfg.config_id"
              :value="cfg.config_id"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="补充要求">
          <el-input v-model="aiForm.extra_prompt" type="textarea" :rows="3" placeholder="可选" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="aiDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="generating" @click="runAiCreate">生成并编辑</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus } from '@element-plus/icons-vue'
import api from '../../utils/api'

const loading = ref(false)
const submitting = ref(false)
const generating = ref(false)
const items = ref([])
const total = ref(0)
const page = ref(1)
const pageSize = 20
const filters = reactive({ q: '', pool_kind: '', shareable: '' })

const dialogVisible = ref(false)
const aiDialogVisible = ref(false)
const isEdit = ref(false)
const formRef = ref()
const aliasesText = ref('')
const llmConfigs = ref([])

const emptyForm = () => ({
  story_id: '',
  title: '',
  theme_key: '',
  canonical_key: '',
  pool_kind: 'named',
  narration_mode: 'canonical',
  mode: 'myth',
  age_band: '',
  full_text: '',
  generation_complete: true
})

const form = reactive(emptyForm())
const aiForm = reactive({
  theme: '',
  pool_kind: 'named',
  narration_mode: 'canonical',
  llm_config_id: '',
  extra_prompt: ''
})

const formRules = {
  title: [{ required: true, message: '请输入标题', trigger: 'blur' }],
  full_text: [{ required: true, message: '请输入正文', trigger: 'blur' }]
}

const poolLabel = (p) => {
  if (p === 'named') return '点名'
  if (p === 'open') return '开放'
  if (p === 'bedtime') return '睡前'
  return p || '—'
}
const poolTagType = (p) => {
  if (p === 'named') return 'danger'
  if (p === 'open') return 'success'
  if (p === 'bedtime') return 'warning'
  return 'info'
}

const formatTime = (v) => {
  if (!v) return '—'
  const d = new Date(v)
  if (Number.isNaN(d.getTime())) return String(v)
  const pad = (n) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`
}

const reload = async () => {
  loading.value = true
  try {
    const params = {
      page: page.value,
      page_size: pageSize,
      q: filters.q || undefined,
      pool_kind: filters.pool_kind || undefined,
      shareable: filters.shareable || undefined
    }
    const { data } = await api.get('/admin/story-assets', { params })
    items.value = data.data?.items || []
    total.value = data.data?.total || 0
  } catch (e) {
    ElMessage.error(e.response?.data?.error || '加载失败')
  } finally {
    loading.value = false
  }
}

const onPageChange = (p) => {
  page.value = p
  reload()
}

const loadLlmConfigs = async () => {
  try {
    const { data } = await api.get('/admin/llm-configs')
    llmConfigs.value = (data.data || []).filter((c) => c.enabled !== false)
  } catch {
    llmConfigs.value = []
  }
}

const resetForm = () => {
  Object.assign(form, emptyForm())
  aliasesText.value = ''
  isEdit.value = false
}

const openCreate = () => {
  resetForm()
  dialogVisible.value = true
}

const openAiCreate = async () => {
  aiForm.theme = ''
  aiForm.pool_kind = 'named'
  aiForm.narration_mode = 'canonical'
  aiForm.llm_config_id = ''
  aiForm.extra_prompt = ''
  if (!llmConfigs.value.length) await loadLlmConfigs()
  aiDialogVisible.value = true
}

const openEdit = async (row) => {
  resetForm()
  isEdit.value = true
  try {
    const { data } = await api.get(`/admin/story-assets/${row.story_id}`)
    const d = data.data || {}
    Object.assign(form, {
      story_id: d.story_id || '',
      title: d.title || '',
      theme_key: d.theme_key || '',
      canonical_key: d.canonical_key || '',
      pool_kind: d.pool_kind || '',
      narration_mode: d.narration_mode || 'canonical',
      mode: d.mode || '',
      age_band: d.age_band || '',
      full_text: d.full_text || '',
      generation_complete: d.generation_complete !== false
    })
    aliasesText.value = (d.aliases || []).join(', ')
    dialogVisible.value = true
  } catch (e) {
    ElMessage.error(e.response?.data?.error || '加载详情失败')
  }
}

const parseAliases = () =>
  aliasesText.value
    .split(/[,，]/)
    .map((s) => s.trim())
    .filter(Boolean)

const buildPayload = () => ({
  story_id: form.story_id || undefined,
  title: form.title,
  theme_key: form.theme_key,
  canonical_key: form.canonical_key,
  aliases: parseAliases(),
  pool_kind: form.pool_kind,
  narration_mode: form.narration_mode,
  mode: form.mode,
  age_band: form.age_band,
  full_text: form.full_text,
  generation_complete: true
})

const submitForm = async () => {
  try {
    await formRef.value?.validate?.()
  } catch {
    return
  }
  submitting.value = true
  try {
    const payload = buildPayload()
    if (isEdit.value) {
      await api.put(`/admin/story-assets/${form.story_id}`, payload)
      ElMessage.success('已保存')
    } else {
      await api.post('/admin/story-assets', payload)
      ElMessage.success('已创建')
    }
    dialogVisible.value = false
    reload()
  } catch (e) {
    ElMessage.error(e.response?.data?.error || '保存失败')
  } finally {
    submitting.value = false
  }
}

const generateIntoForm = async () => {
  const theme = form.theme_key || form.canonical_key || form.title
  if (!theme) {
    ElMessage.warning('请先填写主题或规范名')
    return
  }
  if (!llmConfigs.value.length) await loadLlmConfigs()
  generating.value = true
  try {
    const { data } = await api.post('/admin/story-assets/generate', {
      theme,
      title: form.title,
      pool_kind: form.pool_kind,
      canonical_key: form.canonical_key,
      narration_mode: form.narration_mode,
      mode: form.mode,
      age_band: form.age_band
    })
    const d = data.data || {}
    if (d.title) form.title = d.title
    if (d.theme_key) form.theme_key = d.theme_key
    if (d.canonical_key) form.canonical_key = d.canonical_key
    if (d.full_text) form.full_text = d.full_text
    ElMessage.success('已生成，请检查后保存')
  } catch (e) {
    ElMessage.error(e.response?.data?.error || 'AI 生成失败')
  } finally {
    generating.value = false
  }
}

const runAiCreate = async () => {
  if (!aiForm.theme.trim()) {
    ElMessage.warning('请填写主题')
    return
  }
  generating.value = true
  try {
    const { data } = await api.post('/admin/story-assets/generate', {
      theme: aiForm.theme.trim(),
      pool_kind: aiForm.pool_kind,
      narration_mode: aiForm.narration_mode,
      llm_config_id: aiForm.llm_config_id || undefined,
      extra_prompt: aiForm.extra_prompt,
      canonical_key: aiForm.pool_kind === 'named' ? aiForm.theme.trim() : ''
    })
    const d = data.data || {}
    resetForm()
    Object.assign(form, {
      title: d.title || aiForm.theme,
      theme_key: d.theme_key || aiForm.theme,
      canonical_key: d.canonical_key || (aiForm.pool_kind === 'named' ? aiForm.theme : ''),
      pool_kind: aiForm.pool_kind,
      narration_mode: aiForm.narration_mode,
      mode: aiForm.pool_kind === 'named' ? 'myth' : 'original',
      full_text: d.full_text || '',
      generation_complete: true
    })
    aiDialogVisible.value = false
    dialogVisible.value = true
    ElMessage.success('已生成草稿，请确认后保存')
  } catch (e) {
    ElMessage.error(e.response?.data?.error || 'AI 生成失败')
  } finally {
    generating.value = false
  }
}

const handleDelete = async (row) => {
  try {
    await ElMessageBox.confirm(`确定删除「${row.title || row.story_id}」？将同时清理别名与相关播放记录。`, '删除确认', {
      type: 'warning'
    })
  } catch {
    return
  }
  try {
    await api.delete(`/admin/story-assets/${row.story_id}`)
    ElMessage.success('已删除')
    reload()
  } catch (e) {
    ElMessage.error(e.response?.data?.error || '删除失败')
  }
}

onMounted(() => {
  reload()
  loadLlmConfigs()
})
</script>

<style scoped>
.page-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  margin-bottom: 16px;
  align-items: center;
}
.pager {
  margin-top: 16px;
  display: flex;
  justify-content: flex-end;
}
.text-toolbar {
  width: 100%;
  margin-bottom: 8px;
}
</style>
