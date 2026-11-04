<script setup lang="ts">
import { computed, inject } from "vue"
import { DEFAULT_STYLE_SETTINGS } from "../../../src/style-defaults"
import { namedStylesKey, rarityTextPreviews, STYLE_GRID_TEMPLATE, styleLabel, styleSwatch, usesRarityText } from "../namedStyles"
import { styleClipboard } from "../styleClipboard"
import NumberField from "./NumberField.vue"
import ColorField from "./ColorField.vue"

const props = defineProps<{ styleKey: string; card?: boolean }>()
const context = inject(namedStylesKey)!
const style = computed(() => ({ ...DEFAULT_STYLE_SETTINGS, ...context.styles.value[props.styleKey] }))
const rarityText = computed(() => usesRarityText(props.styleKey))
const displayName = computed(() => styleLabel(props.styleKey))
const previews = computed(() => (rarityText.value ? rarityTextPreviews(props.styleKey) : [displayName.value]))
</script>

<template>
  <div
    :class="['@container min-w-0 rounded focus:outline-2 focus:outline-offset-4 focus:outline-blue-500', card ? 'style-card' : '']"
    :data-style-key="styleKey"
    :tabindex="card ? undefined : -1"
  >
    <div class="style-controls grid grid-cols-3 items-center gap-3" :style="{ '--style-grid': STYLE_GRID_TEMPLATE }">
      <span class="col-span-3 truncate text-sm font-medium text-neutral-200 style-name" :title="displayName">{{ displayName }}</span>
      <div v-for="color in ['text', 'background', 'border'] as const" :key="color" class="flex min-w-0 flex-col gap-1">
        <span class="style-label block text-[10px] uppercase text-neutral-500">{{ color === "background" ? "Bg" : color }}</span>
        <ColorField
          v-if="color !== 'text' || !rarityText"
          :label="`${displayName} ${color}`"
          :model-value="String(style[color])"
          class="h-7 w-full cursor-pointer rounded border border-neutral-700 bg-transparent"
          @update:model-value="context.set(styleKey, color, $event)"
        />
        <span v-else class="text-[10px] text-neutral-400" title="Text colors are inherited from Normal, Magic and Rare styles"
          >Rarities</span
        >
      </div>
      <div class="style-tuning col-span-3 grid grid-cols-[minmax(0,1fr)_minmax(0,1fr)_auto] items-end gap-2">
        <label class="min-w-0 space-y-1">
          <span class="style-label block text-[10px] uppercase text-neutral-500">Size</span>
          <NumberField
            :aria-label="`${displayName} size`"
            :model-value="Number(style.size)"
            integer
            :min="20"
            :max="45"
            @update:model-value="context.set(styleKey, 'size', $event)"
          />
        </label>
        <label class="min-w-0 space-y-1">
          <span class="style-label block text-[10px] uppercase text-neutral-500">Opacity</span>
          <NumberField
            :aria-label="`${displayName} opacity`"
            :model-value="Number(style.backgroundOpacity)"
            :step="0.05"
            :min="0"
            :max="1"
            :decimals="2"
            @update:model-value="context.set(styleKey, 'backgroundOpacity', $event)"
          />
        </label>
        <div class="flex items-center justify-end gap-1 style-actions">
          <button
            type="button"
            title="Copy style"
            :aria-label="`Copy ${displayName} style`"
            class="flex h-[34px] w-[34px] shrink-0 items-center justify-center rounded text-neutral-400 hover:bg-neutral-700 hover:text-white"
            @click="styleClipboard = { name: displayName, style: { ...style } }"
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
              <rect x="9" y="9" width="13" height="13" rx="2"></rect>
              <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"></path>
            </svg>
          </button>
          <button
            type="button"
            title="Paste style"
            :aria-label="`Paste ${displayName} style`"
            :disabled="!styleClipboard"
            class="flex h-[34px] w-[34px] shrink-0 items-center justify-center rounded text-neutral-400 hover:bg-neutral-700 hover:text-white disabled:opacity-30"
            @click="styleClipboard && context.paste(styleKey, styleClipboard.style)"
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
              <path d="M16 4h2a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2V6a2 2 0 0 1 2-2h2"></path>
              <rect x="8" y="2" width="8" height="4" rx="1"></rect>
            </svg>
          </button>
          <button
            type="button"
            title="Reset to global"
            :aria-label="`Reset ${displayName} to global`"
            :disabled="!context.overrides.value[styleKey]"
            class="flex h-[34px] w-[34px] shrink-0 items-center justify-center rounded text-neutral-400 hover:bg-neutral-700 hover:text-white disabled:opacity-30"
            @click="context.reset(styleKey)"
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
      <div class="style-preview col-span-3 flex min-w-0 flex-wrap gap-1">
        <component
          :is="card ? 'button' : 'div'"
          v-for="label in previews"
          :key="label"
          :type="card ? 'button' : undefined"
          :aria-label="card ? `Show ${displayName} in Styles (${label} preview)` : undefined"
          :title="card ? 'Show in Styles' : undefined"
          :data-rarity-preview="rarityText ? label : undefined"
          class="flex min-w-0 flex-1 items-center justify-center overflow-hidden rounded border px-1 text-center"
          :style="{
            ...styleSwatch(rarityText ? { ...style, text: context.styles.value[label.toLowerCase()]?.text } : style),
            ...(rarityText ? { fontSize: 'clamp(10px, 1.65cqi, 14px)' } : {}),
          }"
          @click="card && context.reveal(styleKey)"
        >
          <span class="truncate">{{ label }}</span>
        </component>
      </div>
    </div>
  </div>
</template>

<style scoped>
@container (min-width: 48rem) {
  :not(.style-card) > .style-controls {
    grid-template-columns: var(--style-grid);
  }
  :not(.style-card) > .style-controls > .style-name,
  :not(.style-card) > .style-controls > .style-preview {
    grid-column: auto;
  }
  :not(.style-card) > .style-controls .style-label {
    display: none;
  }
  :not(.style-card) > .style-controls > .style-tuning {
    display: contents;
  }
}
</style>
