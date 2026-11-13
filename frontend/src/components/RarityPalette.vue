<script setup lang="ts">
import { PALETTE_KEYS, type StyleMap } from "../namedStyles"
import ColorField from "./ColorField.vue"

withDefaults(defineProps<{ styles: StyleMap; overrides: StyleMap; keys?: string[]; resetLabel: string }>(), {
  keys: () => PALETTE_KEYS,
})
defineEmits<{ set: [key: string, value: string]; reset: [key: string] }>()
</script>

<template>
  <div class="mb-3 rounded border border-neutral-800 p-3">
    <h3 class="mb-2 text-sm font-semibold text-neutral-200">Shared rarity text colours</h3>
    <div class="flex flex-wrap gap-3">
      <div
        v-for="key in keys"
        :key="key"
        :data-style-key="key"
        tabindex="-1"
        class="flex min-w-0 items-center gap-2 rounded focus:outline-2 focus:outline-offset-4 focus:outline-blue-500"
      >
        <ColorField
          :label="`Shared rarity text ${key.slice('sharedRarity'.length)}`"
          :model-value="String(styles[key]?.text ?? '#FFFFFF')"
          class="h-8 w-10 rounded border border-neutral-700"
          @update:model-value="$emit('set', key, $event)"
        />
        <span class="text-sm text-neutral-300">{{ key.slice("sharedRarity".length).toLowerCase() }}</span>
        <button
          type="button"
          :aria-label="`Reset shared ${key.slice('sharedRarity'.length)} text ${resetLabel}`"
          :title="`Reset ${resetLabel}`"
          :disabled="!overrides[key]"
          class="flex h-[34px] w-[34px] items-center justify-center rounded text-neutral-400 hover:bg-neutral-700 hover:text-white focus-visible:outline-2 focus-visible:outline-blue-500 disabled:opacity-30"
          @click="$emit('reset', key)"
        >
          <svg
            class="h-4 w-4"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
            stroke-linecap="round"
            stroke-linejoin="round"
            aria-hidden="true"
          >
            <polyline points="1 4 1 10 7 10"></polyline>
            <path d="M3.51 15a9 9 0 1 0 2.13-9.36L1 10"></path>
          </svg>
        </button>
      </div>
    </div>
  </div>
</template>
