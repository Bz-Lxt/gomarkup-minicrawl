<script setup>
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { api, adminOrigin } from './api.js'
import ForceGraph from './components/ForceGraph.vue'

const stats = ref({})
const graph = ref({ nodes: [], edges: [] })
const keywords = ref([])
const points = ref([])
const q = ref('inverted')
const hits = ref(null)
const err = ref('')
const panelOpen = ref(true)
const toasts = ref([])
let timer

function toast(text, type = 'ok') {
  const id = Date.now() + Math.random()
  toasts.value.push({ id, text, type })
  setTimeout(() => {
    toasts.value = toasts.value.filter((t) => t.id !== id)
  }, 5000)
}

async function refresh() {
  try {
    const [s, g, k, ts] = await Promise.all([api.stats(), api.graph(), api.keywords(), api.timeseries()])
    stats.value = s
    graph.value = g
    keywords.value = k.keywords || []
    points.value = (ts.points || []).slice(-48)
  } catch (e) {
    toast(e.message, 'error')
  }
}

async function search(term) {
  err.value = ''
  const query = (term || q.value || '').trim()
  if (!query) {
    err.value = '请输入关键词'
    return
  }
  q.value = query
  try {
    hits.value = await api.search(query)
  } catch (e) {
    err.value = e.message
  }
}

function onSelect(node) {
  const hint = (node.title || '').split(/[\s·]/)[0]
  if (hint) search(hint)
}

const spark = computed(() => {
  const pts = points.value
  if (!pts.length) return ''
  const max = Math.max(1, ...pts.map((p) => p.rps || 0))
  return pts.map((p, i) => `${(i / Math.max(pts.length - 1, 1)) * 100},${28 - ((p.rps || 0) / max) * 24}`).join(' ')
})

const maxFreq = computed(() => Math.max(1, ...keywords.value.map((k) => k.freq)))

onMounted(() => {
  refresh()
  timer = setInterval(refresh, 1500)
})
onUnmounted(() => clearInterval(timer))
</script>

<template>
  <div class="flex h-full w-full flex-col">
    <header class="flex w-full flex-wrap items-center gap-3 border-b border-line bg-panel px-4 py-3">
      <div>
        <p class="font-display text-lg text-amber">MINICRAWL SONAR</p>
        <p class="font-mono text-[10px] tracking-[0.25em] text-mute">GRAPH / INDEX / RATE</p>
      </div>
      <div class="flex flex-1 flex-wrap gap-4 font-mono text-xs text-cyan">
        <span>PAGES {{ stats.pages_crawled ?? 0 }}</span>
        <span>DOCS {{ stats.index_docs ?? 0 }}</span>
        <span>RPS {{ Number(stats.current_rps || 0).toFixed(0) }}</span>
        <span>{{ stats.updated_at || '—' }}</span>
      </div>
      <a :href="adminOrigin()" class="text-sm text-amber hover:underline">返回指挥台</a>
      <button class="text-sm text-mute md:hidden" type="button" @click="panelOpen=!panelOpen">情报</button>
    </header>

    <div class="flex min-h-0 w-full flex-1 flex-col md:flex-row">
      <section class="relative min-h-[52vh] w-full flex-1 bg-bg">
        <ForceGraph :graph="graph" @select="onSelect" />
        <p v-if="!(graph.nodes||[]).length" class="pointer-events-none absolute inset-0 flex items-center justify-center text-mute">
          尚无图谱。请先在指挥台启动 fixture 爬取。
        </p>
      </section>

      <aside
        class="w-full overflow-y-auto border-t border-line bg-panel p-4 md:w-[380px] md:border-l md:border-t-0"
        :class="panelOpen ? 'block' : 'hidden md:block'"
      >
        <form class="flex gap-2" @submit.prevent="search()">
          <input v-model="q" class="w-full border border-line bg-bg px-3 py-2 text-sm outline-none focus:border-cyan" placeholder="检索倒排索引" />
          <button class="cut-btn bg-cyan px-3 py-2 text-sm text-bg" type="submit">搜</button>
        </form>
        <p v-if="err" class="mt-2 text-xs text-danger">{{ err }}</p>

        <p class="mt-5 font-mono text-[10px] tracking-widest text-mute">RATE</p>
        <svg viewBox="0 0 100 32" class="mt-2 h-16 w-full" preserveAspectRatio="none">
          <polyline :points="spark" fill="none" stroke="#E8B84A" stroke-width="1.2" />
        </svg>

        <p class="mt-5 font-mono text-[10px] tracking-widest text-mute">TOP TERMS</p>
        <ul class="mt-2 space-y-1">
          <li v-for="k in keywords" :key="k.term">
            <button class="flex w-full items-center gap-2 text-left" type="button" @click="search(k.term)">
              <span class="w-24 truncate font-mono text-xs text-cyan">{{ k.term }}</span>
              <span class="h-2 bg-amber" :style="{ width: `${(k.freq / maxFreq) * 100}%` }" />
              <span class="font-mono text-xs text-mute">{{ k.freq }}</span>
            </button>
          </li>
        </ul>

        <p class="mt-5 font-mono text-[10px] tracking-widest text-mute">HITS</p>
        <ul class="mt-2 space-y-3">
          <li v-for="h in hits?.hits || []" :key="h.doc_id" class="border border-line bg-panel2 p-2">
            <a :href="h.url" class="text-sm text-amber" target="_blank" rel="noreferrer" v-html="h.title" />
            <p class="mt-1 text-xs leading-relaxed text-ink" v-html="h.snippet" />
          </li>
        </ul>
      </aside>
    </div>

    <div class="pointer-events-none fixed right-4 top-16 z-50 flex w-[min(100%-2rem,320px)] flex-col gap-2">
      <div v-for="t in toasts" :key="t.id" class="pointer-events-auto flex gap-2 border border-line bg-panel2 px-3 py-2 text-sm">
        <span class="flex-1">{{ t.text }}</span>
        <button type="button" @click="toasts = toasts.filter(x => x.id!==t.id)">×</button>
      </div>
    </div>
  </div>
</template>
