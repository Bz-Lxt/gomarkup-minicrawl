<script setup>
import { onMounted } from 'vue'

const props = defineProps({
  open: Boolean,
  title: { type: String, default: '' },
  danger: Boolean,
})
const emit = defineEmits(['close', 'confirm'])

function onKey(e) {
  if (e.key === 'Escape') emit('close')
}
onMounted(() => window.addEventListener('keydown', onKey))
</script>

<template>
  <div v-if="open" class="fixed inset-0 z-50 flex items-center justify-center bg-black/70 p-4" @click.self="emit('close')">
    <div class="w-full max-w-md border border-line bg-panel p-6 shadow-[0_0_40px_rgba(46,230,214,0.08)]">
      <h3 class="font-display text-xl text-amber">{{ title }}</h3>
      <div class="mt-4 text-sm text-ink leading-relaxed">
        <slot />
      </div>
      <div class="mt-6 flex justify-end gap-3">
        <button class="cut-btn border border-line px-4 py-2 text-sm text-mute hover:text-ink" type="button" @click="emit('close')">取消</button>
        <button
          class="cut-btn px-4 py-2 text-sm font-medium"
          :class="danger ? 'bg-danger text-bg' : 'bg-amber text-bg'"
          type="button"
          @click="emit('confirm')"
        >确认</button>
      </div>
    </div>
  </div>
</template>
