<script setup lang="ts">
defineProps<{
  title: string
  expanded: boolean
  heading?: boolean
  compact?: boolean
}>()
const emit = defineEmits<{ toggle: [] }>()
</script>

<template>
  <div class="flex items-center rounded text-sm font-semibold text-neutral-400" :class="compact ? '' : 'mb-1'">
    <button
      type="button"
      class="flex min-w-0 flex-1 items-center gap-2 text-left"
      :class="compact ? 'py-1' : 'py-3'"
      :aria-expanded="expanded"
      @click="emit('toggle')"
    >
      <span class="min-w-0" :class="heading ? 'uppercase tracking-wide' : ''">{{ title }}</span>
      <span aria-hidden="true" class="order-first shrink-0">{{ expanded ? "▾" : "▸" }}</span>
    </button>
  </div>
</template>
