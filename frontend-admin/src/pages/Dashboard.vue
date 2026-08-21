<script setup>
import { computed } from 'vue'

const props = defineProps({
  stats: { type: Object, default: () => ({}) },
  points: { type: Array, default: () => [] },
})

const cards = computed(() => [
  { k: 'PAGES', v: props.stats.pages_crawled ?? 0, s: '已爬页面' },
  { k: 'QUEUE', v: props.stats.queue_length ?? 0, s: '队列深度' },
  { k: 'DOCS', v: props.stats.index_docs ?? 0, s: '索引文档' },
  { k: 'TERMS', v: props.stats.index_terms ?? 0, s: '关键词' },
  { k: 'RPS', v: Number(props.stats.current_rps || 0).toFixed(0), s: '当前速率' },
  { k: 'WORK', v: `${props.stats.workers?.active ?? 0}/${props.stats.workers?.configured ?? 0}`, s: 'Worker' },
])

const path = computed(() => {
  const pts = props.points.slice(-40)
  if (!pts.length) return ''
  const max = Math.max(1, ...pts.map((p) => p.rps || 0))
  return pts.map((p, i) => {
    const x = (i / Math.max(pts.length - 1, 1)) * 100
    const y = 36 - ((p.rps || 0) / max) * 32
    return `${x},${y}`
  }).join(' ')
})
</script>

<template>
  <div class="flex w-full flex-col gap-6">
    <header class="border-b border-line pb-4">
      <p class="font-mono text-xs tracking-[0.3em] text-cyan">SONAR / OVERVIEW</p>
      <h1 class="mt-1 font-display text-3xl text-ink">爬取态势</h1>
      <p class="mt-2 text-sm text-mute">更新于 {{ stats.updated_at || '—' }} · 模式 {{ stats.crawl_mode || '—' }}</p>
    </header>
    <div class="grid w-full grid-cols-2 gap-3 md:grid-cols-3 xl:grid-cols-6">
      <article v-for="c in cards" :key="c.k" class="border border-line bg-panel p-4">
        <p class="font-mono text-[10px] tracking-widest text-mute">{{ c.k }}</p>
        <p class="mt-2 font-display text-3xl text-amber">{{ c.v }}</p>
        <p class="mt-1 text-xs text-mute">{{ c.s }}</p>
      </article>
    </div>
    <section class="border border-line bg-panel p-4">
      <p class="font-mono text-xs text-cyan">RATE WINDOW</p>
      <svg viewBox="0 0 100 40" class="mt-3 h-32 w-full" preserveAspectRatio="none">
        <polyline :points="path" fill="none" stroke="#2EE6D6" stroke-width="0.8" />
      </svg>
    </section>
  </div>
</template>
