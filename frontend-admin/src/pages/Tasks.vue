<script setup>
import { reactive, ref, computed } from 'vue'
import Modal from '../components/Modal.vue'

const props = defineProps({
  tasks: { type: Array, default: () => [] },
  mode: { type: String, default: 'mock' },
})
const emit = defineEmits(['create', 'stop', 'pause', 'resume', 'clear', 'toast'])

const form = reactive({
  seeds: '',
  use_fixture: true,
  workers: 8,
  global_rps: 0,
  host_rps: 0,
  max_depth: 6,
  max_pages: 200,
})
const errors = reactive({})
const confirmStop = ref(null)
const confirmClear = ref(false)

const sorted = computed(() =>
  [...props.tasks].sort((a, b) => (b.created_at || '').localeCompare(a.created_at || '')),
)

function validate() {
  Object.keys(errors).forEach((k) => delete errors[k])
  if (!form.use_fixture && props.mode === 'real' && !form.seeds.trim()) {
    errors.seeds = '真实模式必须提供种子 URL'
  }
  if (form.seeds.trim()) {
    const lines = form.seeds.split('\n').map((s) => s.trim()).filter(Boolean)
    const bad = lines.find((u) => !/^https?:\/\//i.test(u))
    if (bad) errors.seeds = `非法 URL：${bad}`
  }
  if (form.workers < 1 || form.workers > 64) errors.workers = 'Worker 数量须为 1–64'
  if (form.max_depth < 1 || form.max_depth > 20) errors.max_depth = '深度须为 1–20'
  if (form.max_pages < 1 || form.max_pages > 5000) errors.max_pages = '页数须为 1–5000'
  if (form.global_rps < 0) errors.global_rps = '速率不能为负'
  return Object.keys(errors).length === 0
}

function submit() {
  if (!validate()) {
    emit('toast', { type: 'error', text: '请修正表单错误后再提交' })
    return
  }
  const seeds = form.seeds.split('\n').map((s) => s.trim()).filter(Boolean)
  emit('create', {
    seeds,
    use_fixture: form.use_fixture,
    workers: Number(form.workers),
    global_rps: Number(form.global_rps),
    host_rps: Number(form.host_rps),
    max_depth: Number(form.max_depth),
    max_pages: Number(form.max_pages),
  })
}

const statusTone = (s) => ({
  running: 'text-amber',
  paused: 'text-cyan',
  completed: 'text-ok',
  failed: 'text-danger',
  stopped: 'text-mute',
}[s] || 'text-ink')
</script>

<template>
  <div class="flex w-full flex-col gap-6 lg:flex-row">
    <form class="w-full border border-line bg-panel p-5 lg:w-[42%]" @submit.prevent="submit">
      <p class="font-mono text-xs tracking-[0.3em] text-cyan">MISSION BRIEF</p>
      <h2 class="mt-1 font-display text-2xl">下发爬取任务</h2>
      <label class="mt-4 block text-sm">种子 URL（每行一条）<span v-if="mode==='real' && !form.use_fixture" class="text-danger"> *</span></label>
      <textarea v-model="form.seeds" rows="4" class="mt-1 w-full border border-line bg-bg p-3 text-sm outline-none focus:border-amber" placeholder="https://example.com" />
      <p v-if="errors.seeds" class="mt-1 text-xs text-danger">{{ errors.seeds }}</p>
      <label class="mt-3 flex items-center gap-2 text-sm">
        <input v-model="form.use_fixture" type="checkbox" />
        使用内置 fixture 站点（离线验收）
      </label>
      <div class="mt-4 grid grid-cols-2 gap-3">
        <div>
          <label class="text-xs text-mute">Workers *</label>
          <input v-model.number="form.workers" type="number" min="1" max="64" class="mt-1 w-full border border-line bg-bg px-3 py-2 text-sm focus:border-amber" />
          <p v-if="errors.workers" class="text-xs text-danger">{{ errors.workers }}</p>
        </div>
        <div>
          <label class="text-xs text-mute">最大深度 *</label>
          <input v-model.number="form.max_depth" type="number" min="1" max="20" class="mt-1 w-full border border-line bg-bg px-3 py-2 text-sm focus:border-amber" />
          <p v-if="errors.max_depth" class="text-xs text-danger">{{ errors.max_depth }}</p>
        </div>
        <div>
          <label class="text-xs text-mute">最大页数 *</label>
          <input v-model.number="form.max_pages" type="number" min="1" max="5000" class="mt-1 w-full border border-line bg-bg px-3 py-2 text-sm focus:border-amber" />
          <p v-if="errors.max_pages" class="text-xs text-danger">{{ errors.max_pages }}</p>
        </div>
        <div>
          <label class="text-xs text-mute">全局 RPS（0=不限）</label>
          <input v-model.number="form.global_rps" type="number" min="0" class="mt-1 w-full border border-line bg-bg px-3 py-2 text-sm focus:border-amber" />
          <p v-if="errors.global_rps" class="text-xs text-danger">{{ errors.global_rps }}</p>
        </div>
        <div class="col-span-2">
          <label class="text-xs text-mute">每域 RPS</label>
          <input v-model.number="form.host_rps" type="number" min="0" class="mt-1 w-full border border-line bg-bg px-3 py-2 text-sm focus:border-amber" />
        </div>
      </div>
      <div class="mt-5 flex flex-wrap gap-3">
        <button class="cut-btn bg-amber px-5 py-2 text-sm font-medium text-bg" type="submit" data-testid="btn-start-task">启动任务</button>
        <button class="cut-btn border border-danger/60 px-5 py-2 text-sm text-danger" type="button" @click="confirmClear = true">清空索引</button>
      </div>
    </form>

    <div class="w-full overflow-x-auto border border-line bg-panel p-5">
      <p class="font-mono text-xs tracking-[0.3em] text-cyan">TASK LOG</p>
      <table class="mt-3 w-full min-w-[640px] text-left text-sm">
        <thead class="text-mute">
          <tr>
            <th class="py-2">ID</th>
            <th>状态</th>
            <th>已爬</th>
            <th>失败</th>
            <th>创建时间</th>
            <th>操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="t in sorted" :key="t.id" class="border-t border-line/80">
            <td class="py-3 font-mono text-xs">{{ t.id }}</td>
            <td :class="statusTone(t.status)">
              {{ t.status }}
              <p v-if="t.error" class="text-xs text-danger">{{ t.error }}</p>
            </td>
            <td>{{ t.crawled }}</td>
            <td :class="t.failures > 0 ? 'text-danger' : 'text-mute'">{{ t.failures || 0 }}</td>
            <td class="font-mono text-xs">{{ t.created_at }}</td>
            <td class="space-x-2">
              <button v-if="t.status==='running'" class="text-cyan hover:underline" type="button" @click="emit('pause', t.id)">暂停</button>
              <button v-if="t.status==='paused'" class="text-amber hover:underline" type="button" @click="emit('resume', t.id)">继续</button>
              <button v-if="t.status==='running' || t.status==='paused'" class="text-danger hover:underline" type="button" @click="confirmStop = t">停止</button>
              <span v-else class="text-mute">—</span>
            </td>
          </tr>
          <tr v-if="!sorted.length">
            <td colspan="6" class="py-8 text-center text-mute">尚无任务</td>
          </tr>
        </tbody>
      </table>
    </div>

    <Modal :open="!!confirmStop" title="停止任务" danger @close="confirmStop=null" @confirm="emit('stop', confirmStop.id); confirmStop=null">
      确认停止任务 {{ confirmStop?.id }}？进行中的抓取会被取消。
    </Modal>
    <Modal :open="confirmClear" title="清空索引" danger @close="confirmClear=false" @confirm="emit('clear'); confirmClear=false">
      将清除内存中的全部文档与倒排索引，且不可恢复（重启亦会丢失，这是产品固有特性）。
    </Modal>
  </div>
</template>
