<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, ref, useId, watch } from "vue"
import { recentColors, rememberColor } from "../recentColors"

defineOptions({ inheritAttrs: false })
const props = defineProps<{ modelValue: string; label: string }>()
const emit = defineEmits<{ "update:modelValue": [value: string] }>()
const id = useId()
const trigger = ref<HTMLButtonElement | null>(null)
const panel = ref<HTMLDivElement | null>(null)
const open = ref(false)
const position = ref({ top: "8px", left: "8px" })
const color = ref(props.modelValue)
const hex = ref(props.modelValue)
const hue = ref(0)
const saturation = ref(0)
const brightness = ref(0)
let pending: ReturnType<typeof setTimeout> | undefined
let lastUpdate = -Infinity
let lastEmitted = props.modelValue.toLowerCase()

function sync(value: string) {
  value = value.toLowerCase()
  color.value = hex.value = value
  const rgb = value
    .replace(/^#/, "")
    .match(/../g)
    ?.map((channel) => Number.parseInt(channel, 16) / 255)
  if (!rgb || rgb.length !== 3 || rgb.some(Number.isNaN)) return
  const [r, g, b] = rgb
  const max = Math.max(r, g, b)
  const min = Math.min(r, g, b)
  const delta = max - min
  if (delta) hue.value = ((max === r ? (g - b) / delta : max === g ? (b - r) / delta + 2 : (r - g) / delta + 4) * 60 + 360) % 360
  saturation.value = max ? (delta / max) * 100 : 0
  brightness.value = max * 100
}
sync(props.modelValue)

watch(
  () => props.modelValue,
  (value) => {
    value = value.toLowerCase()
    // Our own updates must not round the HSV controls or move the drag handle.
    if (value === lastEmitted) return
    clearTimeout(pending)
    pending = undefined
    lastEmitted = value
    sync(value)
  },
)

function flush() {
  clearTimeout(pending)
  pending = undefined
  if (color.value === lastEmitted) return
  lastUpdate = Date.now()
  lastEmitted = color.value
  emit("update:modelValue", color.value)
}

function update(value: string) {
  color.value = hex.value = value
  const remaining = 100 - (Date.now() - lastUpdate)
  if (remaining <= 0) flush()
  else if (pending === undefined) pending = setTimeout(flush, remaining)
}

function updateHsv() {
  const h = hue.value / 60
  const s = saturation.value / 100
  const v = brightness.value / 100
  const channel = (n: number) => {
    const k = (n + h) % 6
    return Math.round((v - v * s * Math.max(0, Math.min(k, 4 - k, 1))) * 255)
      .toString(16)
      .padStart(2, "0")
  }
  update(`#${channel(5)}${channel(3)}${channel(1)}`)
}

function editHex(event: Event) {
  hex.value = (event.target as HTMLInputElement).value
  if (!/^#?[0-9a-f]{6}$/i.test(hex.value)) return
  const value = `#${hex.value.replace(/^#/, "").toLowerCase()}`
  sync(value)
  update(value)
}

function finishHex() {
  hex.value = color.value
  flush()
}

function useRecent(color: string) {
  sync(color)
  flush()
}

function pick(event: PointerEvent) {
  const square = event.currentTarget as HTMLElement
  if (event.type === "pointermove" && !square.hasPointerCapture(event.pointerId)) return
  if (event.type === "pointerdown") {
    if (event.button !== 0) return
    square.setPointerCapture(event.pointerId)
  }
  const bounds = square.getBoundingClientRect()
  if (!bounds.width || !bounds.height) return
  saturation.value = Math.max(0, Math.min(100, ((event.clientX - bounds.left) / bounds.width) * 100))
  brightness.value = Math.max(0, Math.min(100, (1 - (event.clientY - bounds.top) / bounds.height) * 100))
  updateHsv()
}

const hueColor = computed(() => `hsl(${hue.value}, 100%, 50%)`)
const sliders = computed(() => [
  { name: "Hue", value: hue.value, max: 360, background: "linear-gradient(to right, #f00, #ff0, #0f0, #0ff, #00f, #f0f, #f00)" },
  { name: "Saturation", value: saturation.value, max: 100, background: `linear-gradient(to right, #fff, ${hueColor.value})` },
  { name: "Brightness", value: brightness.value, max: 100, background: `linear-gradient(to right, #000, ${hueColor.value})` },
])
function slide(name: string, event: Event) {
  const value = Number((event.target as HTMLInputElement).value)
  if (name === "Hue") hue.value = value
  else if (name === "Saturation") saturation.value = value
  else brightness.value = value
  updateHsv()
}

function place() {
  if (!trigger.value || !panel.value) return
  const anchor = trigger.value.getBoundingClientRect()
  const bounds = panel.value.getBoundingClientRect()
  position.value = {
    left: `${Math.max(8, Math.min(anchor.left, window.innerWidth - bounds.width - 8))}px`,
    top: `${anchor.bottom + bounds.height + 6 <= window.innerHeight - 8 ? anchor.bottom + 6 : Math.max(8, anchor.top - bounds.height - 6)}px`,
  }
}
function close(restoreFocus = false) {
  if (!open.value) return
  flush()
  // Record only the final color, not every intermediate shade while dragging.
  // Closing an unchanged color also makes it available to other style fields.
  rememberColor(color.value)
  open.value = false
  if (restoreFocus) trigger.value?.focus({ preventScroll: true })
}
function outside(event: Event) {
  if (!panel.value?.contains(event.target as Node) && !trigger.value?.contains(event.target as Node)) close()
}
function listen(add: boolean) {
  const method = add ? "addEventListener" : "removeEventListener"
  document[method]("mousedown", outside)
  document[method]("focusin", outside)
  window[method]("resize", place)
  window[method]("scroll", place, true)
}
watch(open, async (value) => {
  listen(value)
  if (!value) return
  await nextTick()
  if (!open.value) return
  place()
  panel.value?.querySelector<HTMLInputElement>('input[type="text"]')?.focus({ preventScroll: true })
})
onBeforeUnmount(() => {
  listen(false)
  // Input completion/closing flushes pending edits; unmount must not restore a
  // stale color after a reset, undo, or switch to another filter.
  clearTimeout(pending)
})
</script>

<template>
  <button
    v-bind="$attrs"
    :id="id"
    ref="trigger"
    type="button"
    :aria-label="label"
    :title="`${label}: ${color}`"
    aria-haspopup="dialog"
    :aria-expanded="open"
    :aria-controls="open ? `${id}-picker` : undefined"
    :value="color"
    class="min-h-7 w-full cursor-pointer rounded border border-neutral-600 p-1 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-blue-500"
    @click.stop="open ? close() : (open = true)"
  >
    <span class="block h-full min-h-5 w-full rounded-sm" :style="{ backgroundColor: color }" aria-hidden="true"></span>
  </button>
  <!-- Stay in the editor's DOM subtree so its outside-click/focus handling
       recognizes this nested picker. Fixed positioning avoids scroll clipping. -->
  <div
    v-if="open"
    :id="`${id}-picker`"
    ref="panel"
    role="dialog"
    :aria-label="`Choose ${label}`"
    class="fixed z-[60] w-64 max-w-[calc(100vw-16px)] space-y-3 overflow-y-auto rounded border border-neutral-600 bg-neutral-900 p-3 text-neutral-200 shadow-xl"
    :style="{ ...position, maxHeight: 'calc(100vh - 16px)' }"
    @click.stop
    @keydown.esc.stop.prevent="close(true)"
  >
    <div class="flex items-center gap-2">
      <label :for="`${id}-hex`" class="text-xs">Hex</label>
      <input
        :id="`${id}-hex`"
        type="text"
        :value="hex"
        :aria-label="`${label} hex`"
        :aria-invalid="!/^#?[0-9a-f]{6}$/i.test(hex)"
        spellcheck="false"
        class="min-w-0 flex-1 rounded border border-neutral-600 bg-neutral-950 px-2 py-1 font-mono text-sm focus:border-blue-500 focus:outline-none"
        @input="editHex"
        @change="flush"
        @blur="finishHex"
        @keydown.enter.prevent="close(true)"
      />
      <button
        type="button"
        title="Close color picker"
        aria-label="Close color picker"
        class="flex h-8 w-8 items-center justify-center rounded hover:bg-neutral-700 focus-visible:outline-2 focus-visible:outline-blue-500"
        @click="close(true)"
      >
        <svg class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true">
          <path d="m6 6 12 12M6 18 18 6" />
        </svg>
      </button>
    </div>
    <div
      aria-hidden="true"
      class="relative h-36 touch-none cursor-crosshair rounded"
      :style="{ background: `linear-gradient(to top, #000, transparent), linear-gradient(to right, #fff, transparent), ${hueColor}` }"
      @pointerdown.prevent="pick"
      @pointermove="pick"
      @pointerup="flush"
      @pointercancel="flush"
      @lostpointercapture="flush"
    >
      <span
        class="pointer-events-none absolute h-3 w-3 -translate-x-1/2 -translate-y-1/2 rounded-full border-2 border-white shadow-[0_0_0_1px_black]"
        :style="{ left: `${saturation}%`, top: `${100 - brightness}%` }"
      ></span>
    </div>
    <label v-for="slider in sliders" :key="slider.name" class="flex items-center gap-2 text-xs">
      <span class="w-16">{{ slider.name }}</span>
      <input
        type="range"
        :aria-label="`${label} ${slider.name.toLowerCase()}`"
        min="0"
        :max="slider.max"
        :value="slider.value"
        class="h-3 min-w-0 flex-1 cursor-pointer appearance-none rounded accent-white focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-blue-500"
        :style="{ background: slider.background }"
        @input="slide(slider.name, $event)"
        @change="flush"
        @pointerup="flush"
        @blur="flush"
        @keyup="flush"
      />
    </label>
    <div role="group" aria-label="Recent colors" class="border-t border-neutral-700 pt-2">
      <p class="mb-2 text-xs text-neutral-400">Recent colors</p>
      <div class="grid min-h-[4.25rem] grid-cols-6 content-start gap-1">
        <button
          v-for="recent in recentColors"
          :key="recent"
          type="button"
          :aria-label="`Use ${recent}`"
          :title="recent"
          :aria-pressed="recent === color"
          class="flex h-8 w-8 items-center justify-center rounded border border-neutral-500 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-blue-500"
          :style="{ backgroundColor: recent }"
          @click="useRecent(recent)"
        >
          <svg
            v-if="recent === color"
            class="h-4 w-4 drop-shadow-[0_1px_1px_black]"
            viewBox="0 0 24 24"
            fill="none"
            stroke="white"
            stroke-width="3"
            aria-hidden="true"
          >
            <path d="m5 12 4 4L19 6" />
          </svg>
        </button>
        <span v-if="!recentColors.length" class="col-span-6 text-xs text-neutral-500">No recent colors</span>
      </div>
    </div>
  </div>
</template>
