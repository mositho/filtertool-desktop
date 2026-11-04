<script setup lang="ts">
import { computed, inject, nextTick, ref } from "vue"
import { groupStyleSections, namedStylesKey, STYLE_ALIASES, STYLE_GRID_TEMPLATE } from "../namedStyles"
import { useCollapsedState } from "../viewState"
import NamedStyleEditor from "./NamedStyleEditor.vue"

const props = defineProps<{ filterName: string }>()
const context = inject(namedStylesKey)!
const panel = ref<HTMLElement | null>(null)
const sections = computed(() =>
  groupStyleSections(Object.keys(context.styles.value))
    .map((section) => ({ ...section, keys: section.keys.filter((key) => context.styles.value[key]) }))
    .filter((section) => section.keys.length),
)
const collapsed = useCollapsedState(
  computed(() => `filtertool:styles:${props.filterName}`),
  () => ({ styles: true }),
)
function setAll(value: boolean) {
  collapsed.value = { styles: false, ...Object.fromEntries(sections.value.map((section) => [section.title, value])) }
}
async function reveal(key: string) {
  const target = STYLE_ALIASES[key] ?? key
  const section = sections.value.find((section) => section.keys.includes(target))
  if (!section) return
  context.openPopover.value = null
  collapsed.value.styles = false
  collapsed.value[section.title] = false
  await nextTick()
  const row = [...(panel.value?.querySelectorAll<HTMLElement>("[data-style-key]") ?? [])].find((row) => row.dataset.styleKey === target)
  row?.focus({ preventScroll: true })
  row?.scrollIntoView?.({ block: "center", behavior: "smooth" })
}
defineExpose({
  reveal,
  collapse: () => {
    collapsed.value.styles = true
  },
  expand: () => {
    collapsed.value.styles = false
  },
})
</script>

<template>
  <section ref="panel" class="@container min-w-0 rounded border border-neutral-800 bg-neutral-900/40" data-editor-group="styles">
    <button
      type="button"
      :aria-expanded="!collapsed.styles"
      class="flex w-full items-center justify-between px-4 py-3 text-left hover:bg-neutral-900/40"
      @click="collapsed.styles = !collapsed.styles"
    >
      <span class="text-lg font-semibold uppercase tracking-wide text-neutral-200">Styles</span>
      <span class="text-neutral-400">{{ collapsed.styles ? "▸" : "▾" }}</span>
    </button>
    <div v-show="!collapsed.styles" class="px-4 pb-4">
      <p class="mb-2 text-xs text-neutral-500">Colors, sizes and opacity for this filter. Unchanged properties inherit Global Settings.</p>
      <div class="mb-3 flex justify-end gap-2">
        <button
          type="button"
          class="rounded border border-neutral-700 px-2 py-0.5 text-xs text-neutral-300 hover:bg-neutral-800"
          @click="setAll(true)"
        >
          Collapse all
        </button>
        <button
          type="button"
          class="rounded border border-neutral-700 px-2 py-0.5 text-xs text-neutral-300 hover:bg-neutral-800"
          @click="setAll(false)"
        >
          Expand all
        </button>
      </div>
      <div v-for="section in sections" :key="section.title" class="mb-3 rounded border border-neutral-800">
        <button
          type="button"
          :aria-expanded="collapsed[section.title] === false"
          class="flex w-full items-center justify-between px-3 py-2 text-left text-sm font-semibold uppercase tracking-wide text-neutral-400 hover:bg-neutral-900/40"
          @click="collapsed[section.title] = collapsed[section.title] === false"
        >
          <span>{{ section.title }}</span
          ><span>{{ collapsed[section.title] === false ? "▾" : "▸" }}</span>
        </button>
        <div v-if="collapsed[section.title] === false" class="space-y-3 border-t border-neutral-800 px-3 py-2">
          <div
            class="hidden items-center gap-3 text-[10px] uppercase text-neutral-500 @min-[48rem]:grid"
            :style="{ gridTemplateColumns: STYLE_GRID_TEMPLATE }"
          >
            <span></span><span>Text</span><span>Bg</span><span>Border</span><span>Size</span><span>Opacity</span><span></span
            ><span class="text-right">Preview</span>
          </div>
          <NamedStyleEditor v-for="key in section.keys" :key="key" :style-key="key" />
        </div>
      </div>
    </div>
  </section>
</template>
