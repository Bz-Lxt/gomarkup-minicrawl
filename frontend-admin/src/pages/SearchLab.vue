<script setup>
import { ref } from 'vue'

const props = defineProps({
  searcher: { type: Function, required: true },
})

const q = ref('minicrawl')
const loading = ref(false)
const result = ref(null)
const error = ref('')

async function run() {
  error.value = ''
  result.value = null
  if (!q.value.trim()) {
    error.value = '请输入关键词'
    return
  }
  loading.value = true
  try {
    result.value = await props.searcher(q.value.trim())
  } catch (e) {
    error.value = e.message || '检索失败'
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="flex w-full flex-col gap-5">
    <header>
      <p class="font-mono text-xs tracking-[0.3em] text-cyan">SEARCH LAB</p>
      <h1 class="font-display text-3xl">检索试验台</h1>
    </header>
    <form class="flex w-full flex-col gap-3 sm:flex-row" @submit.prevent="run">
      <input v-model="q" data-testid="search-input" class="w-full border border-line bg-panel px-4 py-3 text-sm outline-none focus:border-amber" placeholder="关键词，如 inverted / 爬虫" />
      <button class="cut-btn bg-cyan px-6 py-3 text-sm font-medium text-bg" type="submit" data-testid="search-submit">{{ loading ? '检索中…' : '检索' }}</button>
    </form>
    <p v-if="error" class="text-sm text-danger">{{ error }}</p>
    <p v-if="result" class="text-sm text-mute">共 {{ result.total }} 条 · 查询「{{ result.query }}」</p>
    <ul class="flex w-full flex-col gap-3">
      <li v-for="h in result?.hits || []" :key="h.doc_id" class="border border-line bg-panel p-4">
        <a :href="h.url" class="font-medium text-amber hover:underline" target="_blank" rel="noreferrer">
          <span v-html="h.title"></span>
        </a>
        <p class="mt-1 font-mono text-xs text-mute break-all">{{ h.url }}</p>
        <p class="mt-2 text-sm leading-relaxed" data-testid="search-snippet" v-html="h.snippet"></p>
        <p class="mt-2 font-mono text-xs text-cyan">score {{ h.score }}</p>
      </li>
    </ul>
  </div>
</template>
