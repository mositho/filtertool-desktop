<script setup lang="ts">
import { computed } from "vue"

const props = defineProps<{ modelValue: string[]; options: string[]; label: string }>()
const emit = defineEmits<{ "update:modelValue": [value: string[]] }>()
const shown = computed(() => new Set(props.modelValue))

function toggle(option: string) {
  emit("update:modelValue", shown.value.has(option) ? props.modelValue.filter((item) => item !== option) : [...props.modelValue, option])
}
</script>

<template>
  <div
    role="group"
    :aria-label="label"
    class="flex min-h-9 flex-wrap items-center gap-1.5 rounded border border-neutral-700 bg-neutral-900 p-2"
  >
    <button
      v-for="option in options"
      :key="option"
      type="button"
      :aria-label="option"
      :aria-pressed="shown.has(option)"
      :title="`${shown.has(option) ? 'Hide' : 'Show'} ${option}`"
      class="rounded border px-2 py-1 text-xs transition-colors"
      :class="
        shown.has(option)
          ? 'border-neutral-700 bg-neutral-800 text-neutral-200 hover:border-neutral-500'
          : 'border-neutral-800 bg-neutral-950 text-neutral-400 opacity-60 hover:border-neutral-600 hover:opacity-100'
      "
      @click="toggle(option)"
    >
      {{ option }}<span v-if="!shown.has(option)" aria-hidden="true" class="ml-1 text-amber-400">· Hidden</span>
    </button>
  </div>
</template>
