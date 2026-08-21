<script setup>
defineProps({
  toasts: { type: Array, default: () => [] },
})
const emit = defineEmits(['dismiss'])
</script>

<template>
  <div class="pointer-events-none fixed right-4 top-4 z-50 flex w-[min(100%-2rem,360px)] flex-col gap-2">
    <div
      v-for="t in toasts"
      :key="t.id"
      class="pointer-events-auto flex items-start gap-3 border border-line bg-panel2 px-4 py-3"
      :class="t.type === 'error' ? 'border-danger/50' : 'border-cyan/30'"
    >
      <p class="flex-1 text-sm leading-snug">{{ t.text }}</p>
      <button class="text-mute hover:text-ink" type="button" aria-label="关闭" @click="emit('dismiss', t.id)">×</button>
    </div>
  </div>
</template>
