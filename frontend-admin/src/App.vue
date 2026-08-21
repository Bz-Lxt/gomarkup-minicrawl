<script setup>
import { onMounted, onUnmounted, ref } from 'vue'
import { api, visOrigin } from './api.js'
import ToastHost from './components/ToastHost.vue'
import Dashboard from './pages/Dashboard.vue'
import Tasks from './pages/Tasks.vue'
import SearchLab from './pages/SearchLab.vue'

const page = ref('dash')
const menuOpen = ref(false)
const stats = ref({})
const points = ref([])
const tasks = ref([])
const toasts = ref([])
let timer

function toast(t) {
  const id = Date.now() + Math.random()
  toasts.value = [...toasts.value, { id, type: t.type || 'ok', text: t.text }]
  setTimeout(() => dismiss(id), 5000)
}
function dismiss(id) {
  toasts.value = toasts.value.filter((x) => x.id !== id)
}

async function refresh() {
  try {
    const [s, ts, tk] = await Promise.all([api.stats(), api.timeseries(), api.tasks()])
    stats.value = s
    points.value = ts.points || []
    tasks.value = tk.tasks || []
  } catch (e) {
    toast({ type: 'error', text: e.message })
  }
}

async function create(body) {
  try {
    await api.createTask(body)
    toast({ text: '任务已下发' })
    await refresh()
  } catch (e) {
    toast({ type: 'error', text: e.message })
  }
}

async function act(fn, id, ok) {
  try {
    await fn(id)
    toast({ text: ok })
    await refresh()
  } catch (e) {
    toast({ type: 'error', text: e.message })
  }
}

async function onSearch(q) {
  return api.search(q, true)
}

onMounted(() => {
  refresh()
  timer = setInterval(refresh, 1000)
})
onUnmounted(() => clearInterval(timer))

const nav = [
  { id: 'dash', label: '态势' },
  { id: 'tasks', label: '任务' },
  { id: 'search', label: '检索' },
]
</script>

<template>
  <div class="flex min-h-screen w-full">
    <aside
      class="fixed inset-y-0 left-0 z-40 w-56 border-r border-line bg-panel p-5 transition-transform md:static md:translate-x-0"
      :class="menuOpen ? 'translate-x-0' : '-translate-x-full md:translate-x-0'"
    >
      <p class="font-display text-lg text-amber">MINICRAWL</p>
      <p class="mt-1 font-mono text-[10px] tracking-widest text-mute">COMMAND DECK</p>
      <nav class="mt-8 flex flex-col gap-1">
        <button
          v-for="n in nav"
          :key="n.id"
          class="cut-btn px-3 py-2 text-left text-sm"
          :class="page===n.id ? 'bg-amber text-bg' : 'text-mute hover:text-ink'"
          type="button"
          :data-testid="'nav-' + n.id"
          @click="page=n.id; menuOpen=false"
        >{{ n.label }}</button>
        <a :href="visOrigin()" class="mt-4 px-3 py-2 text-sm text-cyan hover:underline" target="_blank" rel="noreferrer">打开可视化 →</a>
      </nav>
    </aside>
    <div class="flex min-h-screen w-full flex-1 flex-col">
      <header class="flex items-center justify-between border-b border-line px-4 py-3 md:hidden">
        <button class="text-amber" type="button" @click="menuOpen=!menuOpen">菜单</button>
        <span class="font-display">MINICRAWL</span>
      </header>
      <main class="w-full flex-1 p-4 sm:p-6">
        <Dashboard v-if="page==='dash'" :stats="stats" :points="points" />
        <Tasks
          v-else-if="page==='tasks'"
          :tasks="tasks"
          :mode="stats.crawl_mode"
          @create="create"
          @stop="(id)=>act(api.stop,id,'已停止')"
          @pause="(id)=>act(api.pause,id,'已暂停')"
          @resume="(id)=>act(api.resume,id,'已继续')"
          @clear="()=>act(api.clearIndex, undefined, '索引已清空')"
          @toast="toast"
        />
        <SearchLab v-else :searcher="onSearch" />
      </main>
    </div>
    <ToastHost :toasts="toasts" @dismiss="dismiss" />
  </div>
</template>
