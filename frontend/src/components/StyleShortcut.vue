<script setup lang="ts">
import { computed, inject, nextTick, onBeforeUnmount, ref, useId, watch } from "vue"
import { namedStylesKey, styleLabel } from "../namedStyles"
import NamedStyleEditor from "./NamedStyleEditor.vue"

const props = defineProps<{ styleKeys: string[] }>()
const context = inject(namedStylesKey, undefined)
const id = useId()
const button = ref<HTMLButtonElement | null>(null)
const panel = ref<HTMLElement | null>(null)
const selected = ref("")
const position = ref({ top: "8px", left: "8px", width: "384px", maxHeight: "calc(100vh - 16px)" })
const keys = computed(() => {
  const available = props.styleKeys.filter((key) => context?.styles.value[key])
  return [...new Set(available)]
})
const open = computed(() => context?.openPopover.value === id)
function close(restoreFocus = false) {
  if (!open.value || !context) return
  context.openPopover.value = null
  if (restoreFocus) void nextTick(() => button.value?.focus({ preventScroll: true }))
}
function place() {
  if (!button.value || !panel.value) return
  const anchor = button.value.getBoundingClientRect()
  const width = Math.min(384, window.innerWidth - 16)
  const height = panel.value.getBoundingClientRect().height
  const below = anchor.bottom + 6
  const top = below + height <= window.innerHeight - 8 ? below : Math.max(8, anchor.top - height - 6)
  position.value = {
    ...position.value,
    width: `${width}px`,
    left: `${Math.max(8, Math.min(anchor.right - width, window.innerWidth - width - 8))}px`,
    top: `${top}px`,
  }
}
function outside(event: Event) {
  if (!panel.value?.contains(event.target as Node) && !button.value?.contains(event.target as Node)) close()
}
function escape(event: Event) {
  if ((event as KeyboardEvent).key === "Escape") {
    event.preventDefault()
    event.stopPropagation()
    close(true)
  }
}
function listen(add: boolean) {
  const method = add ? "addEventListener" : "removeEventListener"
  document[method]("mousedown", outside)
  document[method]("focusin", outside)
  document[method]("keydown", escape)
  window[method]("resize", place)
  window[method]("scroll", place, true)
}
watch(open, async (value) => {
  listen(value)
  if (!value) return
  selected.value = props.styleKeys.find((key) => keys.value.includes(key)) ?? keys.value[0]
  await nextTick()
  if (!open.value) return
  place()
  panel.value?.querySelector<HTMLButtonElement>("button")?.focus({ preventScroll: true })
})
watch(keys, (value) => {
  if (!value.length) close()
  else if (!value.includes(selected.value)) selected.value = value[0]
})
onBeforeUnmount(() => {
  listen(false)
  close()
})
</script>

<template>
  <button
    v-if="context && keys.length"
    ref="button"
    type="button"
    :aria-label="`Edit style: ${styleKeys.map(styleLabel).join(', ')}`"
    title="Edit style for this filter"
    :aria-expanded="open"
    aria-haspopup="dialog"
    :aria-controls="open ? id : undefined"
    class="flex h-6 w-6 shrink-0 items-center justify-center rounded text-neutral-400 hover:bg-neutral-700 hover:text-white"
    @click.stop="context.openPopover.value = open ? null : id"
  >
    <svg class="h-3.5 w-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true">
      <path d="M12 3a9 9 0 1 0 0 18h1.5a2.5 2.5 0 0 0 0-5H13a1.5 1.5 0 0 1 0-3h3a5 5 0 0 0 0-10z" />
      <circle cx="7.5" cy="10" r=".5" />
      <circle cx="10" cy="6.5" r=".5" />
      <circle cx="15" cy="7" r=".5" />
    </svg>
  </button>
  <Teleport to="body">
    <div
      v-if="open"
      :id="id"
      ref="panel"
      role="dialog"
      :aria-label="`Edit ${styleLabel(selected)} style`"
      class="fixed z-50 overflow-y-auto rounded border border-neutral-600 bg-neutral-900 p-4 text-neutral-200 shadow-xl"
      :style="position"
      @click.stop
    >
      <div class="mb-2 flex items-center justify-between gap-2">
        <span class="flex-1 text-sm font-semibold">Filter style</span>
        <button
          v-if="selected"
          type="button"
          :aria-label="`Show ${styleLabel(selected)} in Styles`"
          title="Show in Styles"
          class="flex h-7 w-7 items-center justify-center rounded text-neutral-400 hover:bg-neutral-700 hover:text-white"
          @click="context?.reveal(selected)"
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
            <path d="M15 3h6v6M21 3l-9 9M9 3H5a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-4" />
          </svg>
        </button>
        <button
          type="button"
          aria-label="Close style editor"
          class="rounded px-2 py-1 text-neutral-400 hover:bg-neutral-700 hover:text-white"
          @click="close(true)"
        >
          ✕
        </button>
      </div>
      <p class="mb-3 text-xs text-neutral-400">
        Changes affect every rule in this filter using this style. Reset to inherit Global Settings.
      </p>
      <select
        v-if="keys.length > 1"
        v-model="selected"
        aria-label="Style to edit"
        class="mb-3 w-full rounded border border-neutral-700 bg-neutral-950 px-2 py-1.5 text-sm"
      >
        <option v-for="key in keys" :key="key" :value="key">{{ styleLabel(key) }}</option>
      </select>
      <NamedStyleEditor v-if="selected" :style-key="selected" card />
    </div>
  </Teleport>
</template>
