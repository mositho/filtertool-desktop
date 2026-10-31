<script setup lang="ts">
defineProps<{ compact?: boolean }>()
</script>

<template>
  <details class="space-y-2">
    <summary class="rounded text-sm text-neutral-400" :class="compact ? 'py-1' : 'px-4 py-3'">Advanced</summary>
    <slot />
  </details>
</template>
