<template>
  <div class="p-6">
    <div class="flex items-center justify-between mb-6">
      <div>
        <h1 class="text-2xl font-bold text-white">{{ icon }} {{ title }}</h1>
        <p class="text-gray-400 text-sm mt-1">共 {{ accounts.length }} 个账号</p>
      </div>
      <div class="flex gap-2">
        <button v-if="selectedIds.length > 0" @click="deleteSelected" class="btn btn-danger btn-sm">
          删除 ({{ selectedIds.length }})
        </button>
        <button v-if="canActivate && selectedIds.length === 1" @click="activateSelected" class="btn btn-success btn-sm">
          切换激活
        </button>
        <button @click="showAdd = true" class="btn btn-primary btn-sm">+ 添加账号</button>
      </div>
    </div>

    <div class="card overflow-hidden">
      <table class="w-full text-sm">
        <thead class="bg-gray-800/50">
          <tr>
            <th class="p-3 w-8"><input type="checkbox" class="accent-blue-500" @change="toggleAll" /></th>
            <th class="p-3 text-left text-gray-400">邮箱</th>
            <th class="p-3 text-left text-gray-400">名称</th>
            <th class="p-3 text-left text-gray-400" v-if="canActivate">状态</th>
            <th class="p-3 text-left text-gray-400">标签</th>
            <th class="p-3 text-left text-gray-400">创建时间</th>
            <th class="p-3 text-left text-gray-400">操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-if="loading"><td :colspan="canActivate ? 7 : 6" class="p-8 text-center text-gray-500">加载中...</td></tr>
          <tr v-else-if="accounts.length === 0"><td :colspan="canActivate ? 7 : 6" class="p-8 text-center text-gray-500">暂无账号</td></tr>
          <tr v-for="a in accounts" :key="a.id" class="border-b border-gray-700/50 hover:bg-gray-800/30">
            <td class="p-3"><input type="checkbox" class="accent-blue-500" :checked="selectedIds.includes(a.id)" @change="toggleSelect(a.id)" /></td>
            <td class="p-3 text-gray-100">{{ a.email }}</td>
            <td class="p-3 text-gray-400">{{ a.name || '-' }}</td>
            <td class="p-3" v-if="canActivate">
              <span :class="a.active ? 'badge badge-green' : 'badge badge-gray'">{{ a.active ? '激活' : '未激活' }}</span>
            </td>
            <td class="p-3">
              <span v-if="a.tag_name" class="px-2 py-0.5 rounded text-xs" :style="{ backgroundColor: a.tag_color || '#4B5563', color: '#fff' }">
                {{ a.tag_name }}
              </span>
              <span v-else class="text-gray-600">-</span>
            </td>
            <td class="p-3 text-gray-400 text-xs">{{ formatDate(a.created_at) }}</td>
            <td class="p-3">
              <div class="flex gap-1">
                <button v-if="canActivate" @click="activate(a)" class="btn btn-secondary btn-xs" title="激活">⚡</button>
                <button @click="edit(a)" class="btn btn-secondary btn-xs" title="编辑">✏️</button>
                <button @click="del(a)" class="btn btn-danger btn-xs" title="删除">🗑️</button>
              </div>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Add/Edit Modal -->
    <div v-if="showAdd" class="modal-overlay" @click.self="closeModal">
      <div class="modal-content">
        <div class="modal-header">
          <h3 class="font-semibold text-white">{{ editing ? '编辑账号' : '添加账号' }}</h3>
          <button @click="closeModal" class="text-gray-400 hover:text-white">✕</button>
        </div>
        <div class="modal-body space-y-3">
          <div><label class="label">邮箱 *</label><input v-model="form.email" class="input" placeholder="email@example.com" /></div>
          <div><label class="label">{{ tokenLabel }} *</label><textarea v-model="form[tokenField]" class="input h-20 resize-none font-mono text-xs" :placeholder="tokenPlaceholder"></textarea></div>
          <div><label class="label">名称 (可选)</label><input v-model="form.name" class="input" /></div>
          <div class="grid grid-cols-2 gap-3">
            <div><label class="label">标签名</label><input v-model="form.tag_name" class="input" /></div>
            <div><label class="label">标签颜色</label><input v-model="form.tag_color" type="color" class="input h-10 p-1" /></div>
          </div>
        </div>
        <div class="modal-footer">
          <button @click="closeModal" class="btn btn-secondary">取消</button>
          <button @click="save" class="btn btn-primary">保存</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, inject } from 'vue'

const props = defineProps({
  title: String,
  icon: String,
  api: Object,
  canActivate: { type: Boolean, default: false },
  tokenLabel: { type: String, default: 'Access Token' },
  tokenField: { type: String, default: 'access_token' },
  tokenPlaceholder: { type: String, default: '' },
})

const notify = inject('notify')
const accounts = ref([])
const loading = ref(false)
const selectedIds = ref([])
const showAdd = ref(false)
const editing = ref(null)
const form = ref({})

function formatDate(d) {
  if (!d) return '-'
  return new Date(d).toLocaleDateString('zh-CN')
}
function toggleAll(e) {
  selectedIds.value = e.target.checked ? accounts.value.map(a => a.id) : []
}
function toggleSelect(id) {
  const i = selectedIds.value.indexOf(id)
  if (i === -1) selectedIds.value.push(id)
  else selectedIds.value.splice(i, 1)
}
function closeModal() {
  showAdd.value = false
  editing.value = null
  form.value = {}
}

async function load() {
  loading.value = true
  try { accounts.value = await props.api.list() }
  catch (e) { notify(e.message, 'error') }
  finally { loading.value = false }
}

function edit(a) {
  editing.value = a
  form.value = { ...a }
  showAdd.value = true
}

async function save() {
  try {
    const payload = {
      email: typeof form.value.email === 'string' ? form.value.email.trim() : '',
      [props.tokenField]: typeof form.value[props.tokenField] === 'string' ? form.value[props.tokenField].trim() : '',
      name: typeof form.value.name === 'string' ? form.value.name.trim() || null : null,
      tag_name: typeof form.value.tag_name === 'string' ? form.value.tag_name.trim() || null : null,
      tag_color: typeof form.value.tag_color === 'string' ? form.value.tag_color.trim() || null : null,
    }
    if (!payload.email || !payload[props.tokenField]) {
      notify('请先填写邮箱和令牌', 'error')
      return
    }
    if (editing.value) {
      await props.api.update(editing.value.id, payload)
    } else {
      await props.api.add(payload)
    }
    closeModal()
    notify('保存成功', 'success')
    await load()
  } catch (e) { notify(e.message, 'error') }
}

async function del(a) {
  if (!confirm('确认删除?')) return
  try {
    await props.api.delete(a.id)
    accounts.value = accounts.value.filter(x => x.id !== a.id)
    notify('删除成功', 'success')
  } catch (e) { notify(e.message, 'error') }
}

async function deleteSelected() {
  if (!confirm(`删除 ${selectedIds.value.length} 个账号?`)) return
  try {
    await props.api.deleteMany(selectedIds.value)
    accounts.value = accounts.value.filter(a => !selectedIds.value.includes(a.id))
    selectedIds.value = []
    notify('删除成功', 'success')
  } catch (e) { notify(e.message, 'error') }
}

async function activate(a) {
  try {
    await props.api.activate(a.id)
    accounts.value.forEach(x => x.active = x.id === a.id)
    notify('已激活', 'success')
  } catch (e) { notify(e.message, 'error') }
}

async function activateSelected() {
  if (selectedIds.value.length === 1) {
    const a = accounts.value.find(x => x.id === selectedIds.value[0])
    if (a) await activate(a)
  }
}

onMounted(load)
</script>
