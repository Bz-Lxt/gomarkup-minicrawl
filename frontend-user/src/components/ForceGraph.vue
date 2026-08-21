<script setup>
import { onMounted, onUnmounted, ref, watch, shallowRef } from 'vue'

const props = defineProps({
  graph: { type: Object, default: () => ({ nodes: [], edges: [] }) },
})
const emit = defineEmits(['select'])

const canvasRef = ref(null)
const hover = ref(null)
const sim = shallowRef({ nodes: [], edges: [], scale: 1, ox: 0, oy: 0 })
let raf = 0
let drag = null
let pan = null
let scan = 0

function rebuild() {
  const w = canvasRef.value?.clientWidth || 800
  const h = canvasRef.value?.clientHeight || 600
  const prev = new Map(sim.value.nodes.map((n) => [n.id, n]))
  const nodes = (props.graph.nodes || []).map((n, i) => {
    const old = prev.get(n.id)
    const ang = (i / Math.max(props.graph.nodes.length, 1)) * Math.PI * 2
    return {
      ...n,
      x: old?.x ?? w / 2 + Math.cos(ang) * (80 + (n.degree || 1) * 8),
      y: old?.y ?? h / 2 + Math.sin(ang) * (80 + (n.degree || 1) * 8),
      vx: old?.vx ?? 0,
      vy: old?.vy ?? 0,
    }
  })
  const byId = new Map(nodes.map((n) => [n.id, n]))
  const edges = (props.graph.edges || [])
    .map((e) => ({ a: byId.get(e.from), b: byId.get(e.to) }))
    .filter((e) => e.a && e.b)
  sim.value = { ...sim.value, nodes, edges }
}

function step() {
  const { nodes, edges } = sim.value
  const n = nodes.length
  if (!n) return
  for (let i = 0; i < n; i++) {
    for (let j = i + 1; j < n; j++) {
      const a = nodes[i]
      const b = nodes[j]
      let dx = a.x - b.x
      let dy = a.y - b.y
      let d2 = dx * dx + dy * dy || 1
      const f = 900 / d2
      dx *= f
      dy *= f
      a.vx += dx
      a.vy += dy
      b.vx -= dx
      b.vy -= dy
    }
  }
  for (const e of edges) {
    const dx = e.b.x - e.a.x
    const dy = e.b.y - e.a.y
    e.a.vx += dx * 0.008
    e.a.vy += dy * 0.008
    e.b.vx -= dx * 0.008
    e.b.vy -= dy * 0.008
  }
  const cx = (canvasRef.value?.clientWidth || 800) / 2
  const cy = (canvasRef.value?.clientHeight || 600) / 2
  for (const node of nodes) {
    if (drag?.id === node.id) continue
    node.vx += (cx - node.x) * 0.002
    node.vy += (cy - node.y) * 0.002
    node.vx *= 0.82
    node.vy *= 0.82
    node.x += node.vx
    node.y += node.vy
  }
}

function draw() {
  const canvas = canvasRef.value
  if (!canvas) return
  const ctx = canvas.getContext('2d')
  const dpr = window.devicePixelRatio || 1
  const w = canvas.clientWidth
  const h = canvas.clientHeight
  if (canvas.width !== w * dpr || canvas.height !== h * dpr) {
    canvas.width = w * dpr
    canvas.height = h * dpr
  }
  ctx.setTransform(dpr, 0, 0, dpr, 0, 0)
  ctx.clearRect(0, 0, w, h)

  const { nodes, edges, scale, ox, oy } = sim.value
  ctx.save()
  ctx.translate(ox, oy)
  ctx.scale(scale, scale)

  ctx.strokeStyle = 'rgba(46,230,214,0.12)'
  ctx.lineWidth = 1
  for (const e of edges) {
    ctx.beginPath()
    ctx.moveTo(e.a.x, e.a.y)
    ctx.lineTo(e.b.x, e.b.y)
    ctx.stroke()
  }

  scan += 0.01
  ctx.strokeStyle = 'rgba(232,184,74,0.18)'
  ctx.beginPath()
  ctx.arc(w / 2 / scale - ox / scale, h / 2 / scale - oy / scale, 40 + (scan % 1) * 180, 0, Math.PI * 2)
  ctx.stroke()

  for (const node of nodes) {
    const crawled = node.status === 'crawled'
    ctx.fillStyle = crawled ? '#2EE6D6' : '#E8B84A'
    ctx.globalAlpha = crawled ? 0.95 : 0.55
    const r = 3 + Math.min(node.degree || 1, 8) * 0.6
    ctx.beginPath()
    ctx.arc(node.x, node.y, r, 0, Math.PI * 2)
    ctx.fill()
    ctx.globalAlpha = 1
  }
  ctx.restore()
}

function loop() {
  step()
  draw()
  raf = requestAnimationFrame(loop)
}

function toWorld(ev) {
  const rect = canvasRef.value.getBoundingClientRect()
  const { scale, ox, oy } = sim.value
  return {
    x: (ev.clientX - rect.left - ox) / scale,
    y: (ev.clientY - rect.top - oy) / scale,
  }
}

function hit(p) {
  let best = null
  let bestD = 14
  for (const n of sim.value.nodes) {
    const d = Math.hypot(n.x - p.x, n.y - p.y)
    if (d < bestD) {
      best = n
      bestD = d
    }
  }
  return best
}

function onMove(ev) {
  const p = toWorld(ev)
  if (drag) {
    drag.x = p.x
    drag.y = p.y
    drag.vx = 0
    drag.vy = 0
    return
  }
  if (pan) {
    sim.value.ox += ev.movementX
    sim.value.oy += ev.movementY
    return
  }
  hover.value = hit(p)
}

function onDown(ev) {
  const p = toWorld(ev)
  const n = hit(p)
  if (n) {
    drag = n
    emit('select', n)
  } else {
    pan = true
  }
}

function onUp() {
  drag = null
  pan = null
}

function onWheel(ev) {
  ev.preventDefault()
  const factor = ev.deltaY > 0 ? 0.92 : 1.08
  sim.value.scale = Math.min(4, Math.max(0.3, sim.value.scale * factor))
}

watch(() => props.graph, rebuild, { deep: true })

onMounted(() => {
  rebuild()
  const c = canvasRef.value
  c.addEventListener('mousemove', onMove)
  c.addEventListener('mousedown', onDown)
  window.addEventListener('mouseup', onUp)
  c.addEventListener('wheel', onWheel, { passive: false })
  raf = requestAnimationFrame(loop)
})

onUnmounted(() => {
  cancelAnimationFrame(raf)
  const c = canvasRef.value
  if (c) {
    c.removeEventListener('mousemove', onMove)
    c.removeEventListener('mousedown', onDown)
    c.removeEventListener('wheel', onWheel)
  }
  window.removeEventListener('mouseup', onUp)
})
</script>

<template>
  <div class="relative h-full w-full">
    <canvas ref="canvasRef" class="h-full w-full cursor-crosshair" />
    <div
      v-if="hover"
      class="pointer-events-none absolute left-4 top-4 max-w-sm border border-cyan/30 bg-panel/90 p-3 text-xs"
    >
      <p class="font-display text-cyan">{{ hover.title || '未命名节点' }}</p>
      <p class="mt-1 break-all font-mono text-mute">{{ hover.url }}</p>
      <p class="mt-1 text-amber">{{ hover.status }} · deg {{ hover.degree }}</p>
    </div>
  </div>
</template>
